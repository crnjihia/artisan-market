// path: backend/notification/src/__tests__/notification.test.ts

import { normalizeKenyanPhone, SMSProvider } from '../smsProvider';
import { EmailProvider } from '../emailProvider';
import { renderTemplate, formatKES } from '../templates';
import { RetryHandler } from '../retryHandler';
import { NotificationConsumer } from '../kafkaConsumer';
import { NotificationEventData } from '../types';

describe('Notification Service Unit Tests', () => {
  describe('Phone Normalization', () => {
    it('normalizes Kenyan Safaricom/Airtel phones to E.164 +254...', () => {
      expect(normalizeKenyanPhone('0712345678')).toBe('+254712345678');
      expect(normalizeKenyanPhone('+254712345678')).toBe('+254712345678');
      expect(normalizeKenyanPhone('254712345678')).toBe('+254712345678');
      expect(normalizeKenyanPhone('0112345678')).toBe('+254112345678');
      expect(normalizeKenyanPhone('+254 712 345 678')).toBe('+254712345678');
    });

    it('rejects invalid Kenyan phone numbers', () => {
      expect(() => normalizeKenyanPhone('020123456')).toThrow();
      expect(() => normalizeKenyanPhone('+12025550123')).toThrow();
      expect(() => normalizeKenyanPhone('invalid')).toThrow();
    });
  });

  describe('Currency & Template Formatting', () => {
    it('formats KES amounts correctly', () => {
      expect(formatKES(350000)).toBe('KES 3,500');
      expect(formatKES(0)).toBe('KES 0');
    });

    it('renders English templates for order and payment events', () => {
      const data: NotificationEventData = {
        order_id: 'ord-12345678',
        amount_cents: 250000,
        provider_reference: 'NLJ7RT61SV',
      };

      const created = renderTemplate('order.created', data, 'en');
      expect(created.smsText).toContain('KES 2,500');
      expect(created.smsText).toContain('received');
      expect(created.emailSubject).toContain('Received');

      const confirmed = renderTemplate('order.confirmed', data, 'en');
      expect(confirmed.smsText).toContain('confirmed');

      const paymentSuccess = renderTemplate('payment.succeeded', data, 'en');
      expect(paymentSuccess.smsText).toContain('NLJ7RT61SV');
      expect(paymentSuccess.smsText).toContain('Asante sana');
    });

    it('renders Swahili templates when customer_lang is "sw"', () => {
      const data: NotificationEventData = {
        order_id: 'ord-87654321',
        amount_cents: 180000,
        provider_reference: 'QGH786XYZ',
        reason: 'salio haitoshi',
      };

      const created = renderTemplate('order.created', data, 'sw');
      expect(created.smsText).toContain('limepokelewa');
      expect(created.smsText).toContain('M-Pesa');
      expect(created.emailSubject).toContain('Limepokelewa');

      const confirmed = renderTemplate('order.confirmed', data, 'sw');
      expect(confirmed.smsText).toContain('limethibitishwa');

      const paymentFailed = renderTemplate('payment.failed', data, 'sw');
      expect(paymentFailed.smsText).toContain('halikufanikiwa');
    });
  });

  describe('Providers (Mock Mode)', () => {
    it('SMSProvider successfully delivers mock SMS', async () => {
      const sms = new SMSProvider({ useMocks: true });
      const result = await sms.sendSMS('+254712345678', 'Test message');
      expect(result.success).toBe(true);
      expect(result.recipient).toBe('+254712345678');
      expect(result.messageId).toMatch(/^AT-MOCK-/);
    });

    it('EmailProvider successfully delivers mock email', async () => {
      const email = new EmailProvider({ useMocks: true });
      const result = await email.sendEmail('buyer@artisan-market.local', 'Receipt', 'Body');
      expect(result.success).toBe(true);
      expect(result.recipient).toBe('buyer@artisan-market.local');
      expect(result.messageId).toMatch(/^MAIL-MOCK-/);
    });

    it('EmailProvider rejects malformed email address', async () => {
      const email = new EmailProvider({ useMocks: true });
      const result = await email.sendEmail('not-an-email', 'Subject', 'Body');
      expect(result.success).toBe(false);
      expect(result.error).toContain('Invalid email recipient');
    });
  });

  describe('Retry Handler & Dead Letter Queue (DLQ)', () => {
    it('retries failing operations with backoff and succeeds on transient error', async () => {
      const retryHandler = new RetryHandler({ maxRetries: 3, initialDelayMs: 10, backoffMultiplier: 2 });
      let calls = 0;

      const result = await retryHandler.executeWithRetry(
        'sms',
        '+254712345678',
        'order.created',
        {},
        async () => {
          calls++;
          if (calls < 2) {
            return { success: false, recipient: '+254712345678', error: 'transient network timeout' };
          }
          return { success: true, recipient: '+254712345678', messageId: 'OK-123' };
        }
      );

      expect(calls).toBe(2);
      expect(result.success).toBe(true);
    });

    it('exhausts retries and routes to DLQ on persistent failure', async () => {
      const mockProducer = {
        send: jest.fn().mockResolvedValue([{}]),
      } as any;

      const retryHandler = new RetryHandler(
        { maxRetries: 2, initialDelayMs: 10, backoffMultiplier: 2 },
        mockProducer,
        'notification.dlq'
      );

      let calls = 0;
      const result = await retryHandler.executeWithRetry(
        'email',
        'invalid@example.com',
        'payment.failed',
        { test: 'data' },
        async () => {
          calls++;
          return { success: false, recipient: 'invalid@example.com', error: 'SMTP connection refused' };
        }
      );

      expect(calls).toBe(2);
      expect(result.success).toBe(false);
      expect(result.error).toContain('Exhausted 2 retries');
      expect(mockProducer.send).toHaveBeenCalledTimes(1);
      expect(mockProducer.send).toHaveBeenCalledWith(
        expect.objectContaining({
          topic: 'notification.dlq',
        })
      );
    });
  });

  describe('Notification Consumer Dispatching', () => {
    it('correctly dispatches notifications for an OrderCreated event', async () => {
      const smsProvider = new SMSProvider({ useMocks: true });
      const emailProvider = new EmailProvider({ useMocks: true });

      const smsSpy = jest.spyOn(smsProvider, 'sendSMS');
      const emailSpy = jest.spyOn(emailProvider, 'sendEmail');

      const consumer = new NotificationConsumer(smsProvider, emailProvider, ['localhost:9092']);

      const eventData: NotificationEventData = {
        order_id: 'ord-test-456',
        amount_cents: 450000,
        customer_phone: '+254712345678',
        customer_email: 'buyer@artisan.co.ke',
        customer_lang: 'sw',
      };

      await consumer.handleEvent('order.created', eventData);

      expect(smsSpy).toHaveBeenCalledWith(
        '+254712345678',
        expect.stringContaining('Habari!')
      );
      expect(emailSpy).toHaveBeenCalledWith(
        'buyer@artisan.co.ke',
        expect.stringContaining('Limepokelewa'),
        expect.any(String),
        expect.any(String)
      );
    });
  });
});

