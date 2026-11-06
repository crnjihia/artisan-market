// path: backend/notification/src/emailProvider.ts

import nodemailer from 'nodemailer';
import { EmailResult } from './types';

export interface EmailProviderConfig {
  host?: string;
  port?: number;
  user?: string;
  pass?: string;
  from?: string;
  useMocks?: boolean;
}

export class EmailProvider {
  private from: string;
  private useMocks: boolean;
  private transporter: nodemailer.Transporter | null = null;

  constructor(config?: EmailProviderConfig) {
    this.useMocks =
      config?.useMocks ??
      (process.env.USE_MOCKS || 'true').toLowerCase() === 'true';

    const host = config?.host || process.env.EMAIL_SMTP_HOST || 'localhost';
    const port = config?.port || Number(process.env.EMAIL_SMTP_PORT || '1025');
    const user = config?.user || process.env.EMAIL_SMTP_USER;
    const pass = config?.pass || process.env.EMAIL_SMTP_PASS;
    this.from = config?.from || process.env.EMAIL_FROM || 'Artisan Market <no-reply@artisan-market.local>';

    if (!this.useMocks) {
      this.transporter = nodemailer.createTransport({
        host,
        port,
        secure: port === 465,
        auth: user ? { user, pass } : undefined,
      });
    }
  }

  async sendEmail(to: string, subject: string, text: string, html?: string): Promise<EmailResult> {
    if (!to || !to.includes('@')) {
      return {
        success: false,
        recipient: to,
        error: `Invalid email recipient address: "${to}"`,
      };
    }

    if (this.useMocks || !this.transporter) {
      const mockId = `MAIL-MOCK-${Date.now()}-${Math.floor(Math.random() * 10000)}`;
      console.log(`[Email Provider - MOCK] To: ${to} | Subject: ${subject} | ID: ${mockId}`);
      return {
        success: true,
        recipient: to,
        messageId: mockId,
      };
    }

    try {
      const info = await this.transporter.sendMail({
        from: this.from,
        to,
        subject,
        text,
        html: html || `<p>${text.replace(/\n/g, '<br/>')}</p>`,
      });

      return {
        success: true,
        recipient: to,
        messageId: info.messageId,
      };
    } catch (err: any) {
      console.error(`[Email Provider Error] Failed to send email to ${to}:`, err.message);
      return {
        success: false,
        recipient: to,
        error: err.message,
      };
    }
  }
}
