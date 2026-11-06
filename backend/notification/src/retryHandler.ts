// path: backend/notification/src/retryHandler.ts

import { Producer } from 'kafkajs';

export interface RetryOptions {
  maxRetries?: number;
  initialDelayMs?: number;
  backoffMultiplier?: number;
}

export interface DLQPayload {
  channel: 'sms' | 'email';
  recipient: string;
  eventType: string;
  payload: any;
  error: string;
  attempts: number;
  failedAtUnixMs: number;
}

export class RetryHandler {
  private maxRetries: number;
  private initialDelayMs: number;
  private backoffMultiplier: number;
  private dlqProducer?: Producer;
  private dlqTopic: string;

  constructor(options?: RetryOptions, dlqProducer?: Producer, dlqTopic: string = 'notification.dlq') {
    this.maxRetries = options?.maxRetries ?? 3;
    this.initialDelayMs = options?.initialDelayMs ?? 100;
    this.backoffMultiplier = options?.backoffMultiplier ?? 2;
    this.dlqProducer = dlqProducer;
    this.dlqTopic = dlqTopic;
  }

  async executeWithRetry<T extends { success: boolean; error?: string }>(
    channel: 'sms' | 'email',
    recipient: string,
    eventType: string,
    rawPayload: any,
    operation: () => Promise<T>
  ): Promise<T> {
    let delay = this.initialDelayMs;
    let lastError = 'Operation failed';
    let attempts = 0;

    for (let attempt = 1; attempt <= this.maxRetries; attempt++) {
      attempts = attempt;
      try {
        const result = await operation();
        if (result.success) {
          return result;
        }
        lastError = result.error || 'Unknown delivery failure';
      } catch (err: any) {
        lastError = err.message || 'Operation threw an unhandled error';
      }

      if (attempt < this.maxRetries) {
        await new Promise((resolve) => setTimeout(resolve, delay));
        delay *= this.backoffMultiplier;
      }
    }

    // Retries exhausted – dispatch to Dead Letter Queue (DLQ)
    await this.sendToDLQ({
      channel,
      recipient,
      eventType,
      payload: rawPayload,
      error: lastError,
      attempts,
      failedAtUnixMs: Date.now(),
    });

    return {
      success: false,
      recipient,
      error: `Exhausted ${this.maxRetries} retries: ${lastError}`,
    } as unknown as T;
  }

  private async sendToDLQ(dlqData: DLQPayload): Promise<void> {
    console.error(
      `[DLQ Router] Dispatching dead-letter event to topic "${this.dlqTopic}":`,
      JSON.stringify(dlqData)
    );

    if (this.dlqProducer) {
      try {
        await this.dlqProducer.send({
          topic: this.dlqTopic,
          messages: [
            {
              key: dlqData.recipient,
              value: JSON.stringify(dlqData),
              headers: {
                event_type: 'notification.dlq',
                failed_channel: dlqData.channel,
              },
            },
          ],
        });
      } catch (err: any) {
        console.error(`[DLQ Router Error] Failed to write to DLQ Kafka topic:`, err.message);
      }
    }
  }
}

