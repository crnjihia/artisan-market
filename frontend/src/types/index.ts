// path: frontend/src/types/index.ts

export interface Product {
  id: string;
  name: string;
  swahiliName?: string;
  description: string;
  price_cents: number;
  category: string;
  artisan_name: string;
  artisan_location: string;
  stock_quantity: number;
  image_url: string;
}

export interface CartItem {
  product: Product;
  quantity: number;
}

export interface CheckoutRequest {
  customer_phone: string;
  customer_email?: string;
  payment_method: 'mpesa' | 'stripe';
  customer_lang: 'en' | 'sw';
}

export interface PaymentStatus {
  payment_id: string;
  order_id: string;
  status: 'pending' | 'succeeded' | 'failed';
  receipt_number?: string;
  failure_reason?: string;
}

