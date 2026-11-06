// path: backend/notification/src/smsProvider.ts

import { SMSResult } from './types';

export function normalizeKenyanPhone(raw: string): string {
  let cleaned = raw.trim().replace(/[\s-]/g, '');
  if (cleaned.startsWith('07') || cleaned.startsWith('01')) {
    cleaned = '+254' + cleaned.slice(1);
  } else if (cleaned.startsWith('254')) {
    cleaned = '+' + cleaned;
  }

  const kenyanRegex = /^\+254(7|1)\d{8}$/;
  if (!kenyanRegex.test(cleaned)) {
    throw new Error(`Invalid Kenyan phone number format: "${raw}". Must be +2547XXXXXXXX or +2541XXXXXXXX.`);
  }

  return cleaned;
}

export interface SMSProviderConfig {
  username?: string;
  apiKey?: string;
  from?: string;
  useMocks?: boolean;
}

export class SMSProvider {
  private username: string;
  private apiKey: string;
  private from?: string;
  private useMocks: boolean;

  constructor(config?: SMSProviderConfig) {
    this.username = config?.username || process.env.AFRICA_TALKING_USERNAME || 'sandbox';
    this.apiKey = config?.apiKey || process.env.AFRICA_TALKING_API_KEY || '';
    this.from = config?.from || process.env.AFRICA_TALKING_SENDER_ID || undefined;
    this.useMocks =
      config?.useMocks ??
      (process.env.USE_MOCKS || 'true').toLowerCase() === 'true';
  }

  async sendSMS(rawPhone: string, message: string): Promise<SMSResult> {
    const phone = normalizeKenyanPhone(rawPhone);

    if (this.useMocks || !this.apiKey) {
      const mockId = `AT-MOCK-${Date.now()}-${Math.floor(Math.random() * 10000)}`;
      console.log(`[SMS Provider - MOCK] Recipient: ${phone} | ID: ${mockId}\n${message}`);
      return {
        success: true,
        recipient: phone,
        messageId: mockId,
        statusText: 'Sent (Mock)',
      };
    }

    const isSandbox = this.username.toLowerCase() === 'sandbox';
    const endpoint = isSandbox
      ? 'https://api.sandbox.africastalking.com/version1/messaging'
      : 'https://api.africastalking.com/version1/messaging';

    const params = new URLSearchParams();
    params.append('username', this.username);
    params.append('to', phone);
    params.append('message', message);
    if (this.from && !isSandbox) {
      params.append('from', this.from);
    }

    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          apiKey: this.apiKey,
          Accept: 'application/json',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: params.toString(),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Africa's Talking API error (${response.status}): ${errorText}`);
      }

      const body = (await response.json()) as any;
      const recipients = body?.SMSMessageData?.Recipients;
      if (Array.isArray(recipients) && recipients.length > 0) {
        const first = recipients[0];
        return {
          success: first.status === 'Success' || first.statusCode === 101,
          recipient: first.number || phone,
          messageId: first.messageId,
          statusText: first.status,
          cost: first.cost,
        };
      }

      return {
        success: true,
        recipient: phone,
        messageId: `AT-${Date.now()}`,
        statusText: 'Sent',
      };
    } catch (err: any) {
      console.error(`[SMS Provider Error] Failed to send SMS to ${phone}:`, err.message);
      return {
        success: false,
        recipient: phone,
        error: err.message,
      };
    }
  }
}
