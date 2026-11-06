// path: backend/notification/src/kafkaConsumer.ts

import { Kafka, Consumer, Producer, logLevel } from 'kafkajs';
import { SMSProvider } from './smsProvider';
import { EmailProvider } from './emailProvider';
import { RetryHandler } from './retryHandler';
import { renderTemplate } from './templates';
import { NotificationEventData, Language } from './types';

export class NotificationConsumer {
  private kafka: Kafka;
  private consumer: Consumer;
  private producer: Producer;
  private smsProvider: SMSProvider;
  private emailProvider: EmailProvider;
  private retryHandler: RetryHandler;
  private processedEvents: Set<string> = new Set();
  private maxCacheSize: number = 10000;

  constructor(
    smsProvider?: SMSProvider,
    emailProvider?: EmailProvider,
    brokers?: string[]
  ) {
    const brokerList =
      brokers || (process.env.KAFKA_BROKERS || 'localhost:9092').split(',');

    this.kafka = new Kafka({
      clientId: 'artisan-notification-service',
      brokers: brokerList,
      logLevel: logLevel.ERROR,
    });

    this.consumer = this.kafka.consumer({
      groupId: process.env.NOTIFICATION_GROUP_ID || 'notification-service-group',
    });

    this.producer = this.kafka.producer();
    this.smsProvider = smsProvider || new SMSProvider();
    this.emailProvider = emailProvider || new EmailProvider();
    this.retryHandler = new RetryHandler(
      { maxRetries: 3, initialDelayMs: 100, backoffMultiplier: 2 },
      this.producer,
      process.env.NOTIFICATION_DLQ_TOPIC || 'notification.dlq'
    );
  }

  async start(): Promise<void> {
    await this.producer.connect();
    await this.consumer.connect();

    await this.consumer.subscribe({ topic: 'order.events', fromBeginning: false });
    await this.consumer.subscribe({ topic: 'payment.events', fromBeginning: false });

    console.log('[Notification Service] Subscribed to topics: order.events, payment.events');

    await this.consumer.run({
      eachMessage: async ({ topic, partition, message }) => {
        const rawValue = message.value?.toString();
        if (!rawValue) return;

        let eventType = '';
        if (message.headers) {
          const headerType = message.headers['event_type'];
          if (headerType) {
            eventType = headerType.toString();
          }
        }

        let parsed: any;
        try {
          parsed = JSON.parse(rawValue);
        } catch (e) {
          console.warn('[Notification Service] Malformed JSON payload:', rawValue);
          return;
        }

        if (!eventType && parsed.event_type) {
          eventType = parsed.event_type;
        }
        if (!eventType) {
          eventType = topic;
        }

        const eventId =
          parsed.event_id ||
          (message.key ? message.key.toString() : `${topic}-${partition}-${message.offset}`);

        if (this.processedEvents.has(eventId)) {
          // Idempotent duplicate bypass
          return;
        }

        this.recordProcessedEvent(eventId);
        await this.handleEvent(eventType, parsed);
      },
    });
  }

  async stop(): Promise<void> {
    await this.consumer.disconnect();
    await this.producer.disconnect();
  }

  public async handleEvent(eventType: string, data: NotificationEventData): Promise<void> {
    const lang: Language = data.customer_lang === 'sw' ? 'sw' : 'en';
    const rendered = renderTemplate(eventType, data, lang);

    const promises: Promise<any>[] = [];

    // 1. SMS Notification (If phone number provided)
    const phone = data.customer_phone || (process.env.USE_MOCKS === 'true' ? '+254712345678' : undefined);
    if (phone) {
      promises.push(
        this.retryHandler.executeWithRetry('sms', phone, eventType, data, () =>
          this.smsProvider.sendSMS(phone, rendered.smsText)
        )
      );
    }

    // 2. Email Notification (If email address provided)
    const email = data.customer_email;
    if (email) {
      promises.push(
        this.retryHandler.executeWithRetry('email', email, eventType, data, () =>
          this.emailProvider.sendEmail(email, rendered.emailSubject, rendered.emailText, rendered.emailHtml)
        )
      );
    }

    await Promise.allSettled(promises);
  }

  private recordProcessedEvent(eventId: string) {
    if (this.processedEvents.size >= this.maxCacheSize) {
      // Clear oldest half
      const entries = Array.from(this.processedEvents);
      for (let i = 0; i < entries.length / 2; i++) {
        this.processedEvents.delete(entries[i]);
      }
    }
    this.processedEvents.add(eventId);
  }
}
