// path: frontend/src/components/CartDrawer.tsx
'use client';

import React from 'react';
import { useCart } from '../context/CartContext';

export function CartDrawer() {
  const {
    items,
    isCartOpen,
    setIsCartOpen,
    removeFromCart,
    updateQuantity,
    totalCents,
    setIsCheckoutOpen,
  } = useCart();

  if (!isCartOpen) return null;

  const handleOpenCheckout = () => {
    setIsCartOpen(false);
    setIsCheckoutOpen(true);
  };

  return (
    <div className="drawer-overlay" onClick={() => setIsCartOpen(false)}>
      <div className="drawer-content" onClick={(e) => e.stopPropagation()}>
        <div className="drawer-header">
          <h2>Kikapu Chako (Your Cart)</h2>
          <button className="close-btn" onClick={() => setIsCartOpen(false)}>
            &times;
          </button>
        </div>

        {items.length === 0 ? (
          <div className="empty-cart">
            <p>Your cart is empty.</p>
            <p className="text-muted">Kikapu chako hakina bidhaa kwa sasa.</p>
          </div>
        ) : (
          <>
            <div className="cart-items-list">
              {items.map(({ product, quantity }) => (
                <div key={product.id} className="cart-item">
                  <img src={product.image_url} alt={product.name} className="cart-item-img" />
                  <div className="cart-item-info">
                    <h4>{product.name}</h4>
                    <p className="cart-item-price">
                      KES {(product.price_cents / 100).toLocaleString('en-KE')}
                    </p>
                    <div className="quantity-controls">
                      <button onClick={() => updateQuantity(product.id, quantity - 1)}>-</button>
                      <span>{quantity}</span>
                      <button onClick={() => updateQuantity(product.id, quantity + 1)}>+</button>
                      <button
                        className="remove-btn"
                        onClick={() => removeFromCart(product.id)}
                        title="Remove"
                      >
                        Futa
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            <div className="drawer-footer">
              <div className="subtotal-row">
                <span>Jumla (Total):</span>
                <span className="total-price">
                  KES {(totalCents / 100).toLocaleString('en-KE')}
                </span>
              </div>
              <p className="tax-notice">Prices include VAT where applicable.</p>
              <button
                className="checkout-btn"
                onClick={handleOpenCheckout}
                disabled={items.length === 0}
              >
                Checkout
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

