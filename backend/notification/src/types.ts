// path: backend/notification/src/types.ts

export type Language = 'en' | 'sw';

export type NotificationEventType =
  | 'order.created'
  | 'OrderCreated'
  | 'order.confirmed'
  | 'OrderConfirmed'
  | 'order.cancelled'
  | 'OrderCancelled'
  | 'payment.succeeded'
  | 'PaymentSucceeded'
  | 'payment.failed'
  | 'PaymentFailed';

export interface NotificationEventData {
  event_id?: string;
  order_id?: string;
  payment_id?: string;
  user_id?: string;
  amount_cents?: number;
  currency?: string;
  payment_method?: string;
  customer_phone?: string;
  customer_email?: string;
  customer_lang?: Language;
  provider_reference?: string;
  reason?: string;
}

export interface SMSResult {
  success: boolean;
  recipient: string;
  messageId?: string;
  statusText?: string;
  cost?: string;
  error?: string;
}

export interface EmailResult {
  success: boolean;
  recipient: string;
  messageId?: string;
  error?: string;
}

export interface RenderedMessage {
  smsText: string;
  emailSubject: string;
  emailText: string;
  emailHtml: string;
}
