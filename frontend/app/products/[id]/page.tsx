// path: frontend/app/products/[id]/page.tsx
'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { MOCK_PRODUCTS } from '../../../src/data/mockProducts';
import { useCart } from '../../../src/context/CartContext';

export default function ProductDetailPage() {
  const params = useParams();
  const id = params?.id as string;
  const { addToCart } = useCart();
  const [quantity, setQuantity] = useState(1);

  const product = MOCK_PRODUCTS.find((p) => p.id === id);

  if (!product) {
    return (
      <div className="container" style={{ textAlign: 'center', padding: '6rem 1rem' }}>
        <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>🏺</div>
        <h2 style={{ fontSize: '1.75rem', fontWeight: 800 }}>Product Not Found &bull; Bidhaa Haipatikani</h2>
        <p style={{ marginTop: '0.75rem', color: 'var(--text-muted)' }}>
          The requested artisan piece could not be found in our catalogue.
        </p>
        <Link href="/" className="btn-primary" style={{ display: 'inline-block', marginTop: '2rem' }}>
          &larr; Back to Artisan Catalogue
        </Link>
      </div>
    );
  }

  return (
    <div className="container product-detail-container">
      {/* Breadcrumbs */}
      <nav className="detail-breadcrumbs" aria-label="Breadcrumb">
        <Link href="/" className="breadcrumb-link">Catalogue</Link>
        <span className="breadcrumb-separator">/</span>
        <span className="breadcrumb-current">{product.category}</span>
        <span className="breadcrumb-separator">/</span>
        <span className="breadcrumb-current active">{product.name}</span>
      </nav>

      <div className="product-detail-layout">
        {/* Left Column: Image Showcase */}
        <div className="detail-image-showcase">
          <div className="detail-image-wrapper">
            <img
              src={product.image_url}
              alt={product.name}
              className="detail-main-image"
            />
            <span className="detail-category-chip">{product.category}</span>
          </div>

          <div className="detail-authenticity-card">
            <div className="auth-card-icon">🇰🇪</div>
            <div className="auth-card-text">
              <strong>100% Authentic Kenyan Craftsmanship</strong>
              <p>Hand-verified by our curation guild in Nairobi. Crafted using ethically sourced regional materials.</p>
            </div>
          </div>
        </div>

        {/* Right Column: Information & Actions */}
        <div className="detail-info-panel">
          {product.swahiliName && (
            <div className="detail-swahili-badge">
              Jina la Asili: <em>{product.swahiliName}</em>
            </div>
          )}

          <h1 className="detail-product-title">{product.name}</h1>

          {/* Price Bar */}
          <div className="detail-price-box">
            <span className="detail-currency">KES</span>
            <span className="detail-amount">
              {(product.price_cents / 100).toLocaleString('en-KE')}
            </span>
            <span className="detail-tax-note">Inclusive of direct artisan remuneration</span>
          </div>

          {/* Artisan Provenance Box */}
          <div className="artisan-provenance-card">
            <div className="artisan-header-row">
              <span className="artisan-avatar-icon">👨‍🎨</span>
              <div>
                <h4 className="artisan-card-name">{product.artisan_name}</h4>
                <p className="artisan-card-loc">📍 {product.artisan_location}</p>
              </div>
            </div>
            <p className="artisan-card-description">
              Every purchase through Artisan Market is disbursed directly via Safaricom M-Pesa to support the artisan&rsquo;s family and local apprentice cooperatives.
            </p>
            <div className="stock-status-line">
              <span className="stock-dot"></span>
              <span><strong>{product.stock_quantity} pieces available</strong> at Nairobi fulfillment hub</span>
            </div>
          </div>

          {/* Description */}
          <div className="detail-description-section">
            <h3>Craft Description & Materials</h3>
            <p className="detail-description-text">{product.description}</p>
          </div>

          {/* Quantity and Add to Cart Row */}
          <div className="detail-purchase-controls">
            <div className="quantity-stepper">
              <button
                type="button"
                className="stepper-btn"
                onClick={() => setQuantity(Math.max(1, quantity - 1))}
                aria-label="Decrease quantity"
              >
                &minus;
              </button>
              <span className="stepper-value">{quantity}</span>
              <button
                type="button"
                className="stepper-btn"
                onClick={() => setQuantity(Math.min(product.stock_quantity, quantity + 1))}
                aria-label="Increase quantity"
              >
                &#43;
              </button>
            </div>

            <button
              type="button"
              className="detail-add-cart-btn"
              onClick={() => addToCart(product, quantity)}
            >
              <span>🛒 Add {quantity} to Cart</span>
              <span className="btn-price-preview">
                &bull; KES {((product.price_cents * quantity) / 100).toLocaleString('en-KE')}
              </span>
            </button>
          </div>

          {/* Guarantee Badges */}
          <div className="detail-guarantees-grid">
            <div className="guarantee-cell">
              <span className="guarantee-icon">⚡</span>
              <div>
                <strong>Lipa na M-Pesa</strong>
                <p>Instant phone checkout</p>
              </div>
            </div>
            <div className="guarantee-cell">
              <span className="guarantee-icon">📦</span>
              <div>
                <strong>Careful Packaging</strong>
                <p>Protected artisan shipping</p>
              </div>
            </div>
            <div className="guarantee-cell">
              <span className="guarantee-icon">🤝</span>
              <div>
                <strong>Direct Impact</strong>
                <p>No platform middlemen</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
