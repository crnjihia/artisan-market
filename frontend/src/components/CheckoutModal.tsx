// path: frontend/src/components/CheckoutModal.tsx
'use client';

import React, { useState } from 'react';
import { useCart } from '../context/CartContext';

export function CheckoutModal() {
  const { isCheckoutOpen, setIsCheckoutOpen, items, totalCents, clearCart } = useCart();
  const [phone, setPhone] = useState('0712345678');
  const [email, setEmail] = useState('buyer@example.com');
  const [method, setMethod] = useState<'mpesa' | 'stripe'>('mpesa');
  const [lang, setLang] = useState<'en' | 'sw'>('en');

  // Checkout states: 'idle' | 'processing' | 'success' | 'failed'
  const [status, setStatus] = useState<'idle' | 'processing' | 'success' | 'failed'>('idle');
  const [receipt, setReceipt] = useState<string>('');
  const [errorMessage, setErrorMessage] = useState<string>('');

  if (!isCheckoutOpen) return null;

  const handleClose = () => {
    if (status === 'processing') return;
    setIsCheckoutOpen(false);
    setStatus('idle');
  };

  const handleConfirmCheckout = async () => {
    setStatus('processing');
    setErrorMessage('');

    try {
      // Simulate STK Push or Stripe Intent creation & live polling
      await new Promise((resolve) => setTimeout(resolve, 2000));

      const mockReceipt = `NLJ${Math.floor(100000 + Math.random() * 900000)}SV`;
      setReceipt(mockReceipt);
      setStatus('success');
      clearCart();
    } catch (err: any) {
      setStatus('failed');
      setErrorMessage(err.message || 'Payment initiation failed. Please try again.');
    }
  };

  return (
    <div className="modal-overlay" onClick={handleClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3>
            {lang === 'sw' ? 'Kamilisha Malipo (Checkout)' : 'Complete Your Checkout'}
          </h3>
          <button className="close-btn" onClick={handleClose}>
            &times;
          </button>
        </div>

        {status === 'success' ? (
          <div className="checkout-success">
            <div className="success-icon">✓</div>
            <h2 className="success-title">Payment succeeded!</h2>
            <p className="success-subtitle">
              {lang === 'sw'
                ? 'Malipo yako yamethibitishwa kikamilifu kupitia M-Pesa!'
                : 'Your payment was successfully confirmed!'}
            </p>
            <div className="receipt-card">
              <p>
                <strong>M-Pesa Receipt:</strong> <code>{receipt}</code>
              </p>
              <p>
                <strong>Amount Paid:</strong> KES {(totalCents / 100).toLocaleString('en-KE')}
              </p>
              <p>
                <strong>Phone:</strong> {phone}
              </p>
            </div>
            <p className="sms-hint">
              {lang === 'sw'
                ? 'Ujumbe mfupi wa SMS na barua pepe ya uthibitisho imetumwa.'
                : 'A confirmation SMS and email receipt have been dispatched.'}
            </p>
            <button className="done-btn" onClick={handleClose}>
              {lang === 'sw' ? 'Funga (Continue Shopping)' : 'Done (Continue Shopping)'}
            </button>
          </div>
        ) : (
          <div className="checkout-form">
            <div className="language-toggle">
              <span>Lugha / Language:</span>
              <button
                type="button"
                className={`lang-btn ${lang === 'en' ? 'active' : ''}`}
                onClick={() => setLang('en')}
              >
                English
              </button>
              <button
                type="button"
                className={`lang-btn ${lang === 'sw' ? 'active' : ''}`}
                onClick={() => setLang('sw')}
              >
                Kiswahili
              </button>
            </div>

            <div className="order-summary-box">
              <div className="summary-row">
                <span>Items:</span>
                <span>{items.length} artisan items</span>
              </div>
              <div className="summary-row font-bold">
                <span>Total Amount:</span>
                <span className="total-highlight">
                  KES {(totalCents / 100).toLocaleString('en-KE')}
                </span>
              </div>
            </div>

            <div className="payment-method-selector">
              <label className="method-label">
                <input
                  type="radio"
                  name="paymentMethod"
                  value="mpesa"
                  checked={method === 'mpesa'}
                  onChange={() => setMethod('mpesa')}
                />
                <span className="method-content">
                  <strong>M-Pesa (Lipa na M-Pesa Online STK Push)</strong>
                  <span className="method-desc">
                    {lang === 'sw'
                      ? 'Ombi la papo hapo litatumwa kwenye simu yako ya Safaricom'
                      : 'Instant STK push prompt sent to your Safaricom mobile'}
                  </span>
                </span>
              </label>

              <label className="method-label">
                <input
                  type="radio"
                  name="paymentMethod"
                  value="stripe"
                  checked={method === 'stripe'}
                  onChange={() => setMethod('stripe')}
                />
                <span className="method-content">
                  <strong>Credit / Debit Card (Stripe)</strong>
                  <span className="method-desc">Visa, Mastercard, American Express</span>
                </span>
              </label>
            </div>

            <div className="input-group">
              <label>
                {lang === 'sw' ? 'Nambari ya Simu ya M-Pesa:' : 'M-Pesa Mobile Number:'}
              </label>
              <input
                type="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="0712345678 or +254..."
                disabled={status === 'processing'}
              />
              <span className="input-hint">Kenyan format: +2547XXXXXXXX or 07XXXXXXXX</span>
            </div>

            <div className="input-group">
              <label>
                {lang === 'sw' ? 'Barua Pepe ya Risiti:' : 'Email Address for Receipt:'}
              </label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="buyer@example.com"
                disabled={status === 'processing'}
              />
            </div>

            {errorMessage && <div className="error-alert">{errorMessage}</div>}

            {status === 'processing' ? (
              <div className="processing-indicator">
                <div className="spinner"></div>
                <p>
                  <strong>
                    {lang === 'sw'
                      ? 'Ombi la M-Pesa limetumwa! Thibitisha kwenye simu yako...'
                      : 'STK push sent! Please enter your M-Pesa PIN on your phone...'}
                  </strong>
                </p>
                <p className="polling-text">Awaiting Daraja confirmation callback...</p>
              </div>
            ) : (
              <div className="modal-actions">
                <button
                  type="button"
                  className="cancel-btn"
                  onClick={handleClose}
                >
                  {lang === 'sw' ? 'Ghairi' : 'Cancel'}
                </button>
                <button
                  type="button"
                  className="confirm-btn"
                  onClick={handleConfirmCheckout}
                >
                  Confirm
                </button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

