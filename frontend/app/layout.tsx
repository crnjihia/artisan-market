// path: frontend/app/layout.tsx
'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import './globals.css';
import { CartProvider, useCart } from '../src/context/CartContext';
import { ThemeProvider, useTheme } from '../src/context/ThemeContext';
import { CartDrawer } from '../src/components/CartDrawer';
import { CheckoutModal } from '../src/components/CheckoutModal';

function Navigation() {
  const { itemCount, setIsCartOpen, toastMessage } = useCart();
  const { theme, toggleTheme } = useTheme();
  const pathname = usePathname();

  return (
    <>
      {/* Top Heritage Utility Bar */}
      <aside className="top-utility-bar">
        <div className="top-utility-inner">
          <div className="utility-left">
            <span className="utility-badge">Direct Trade</span>
            <span className="utility-text">
              Direct from Kenyan Master Artisans &bull; Instant Safaricom M-Pesa &bull; Hand-Inspected Quality
            </span>
          </div>
          <div className="utility-right">
            <span className="utility-currency">Currency: <strong>KES (Ksh)</strong></span>
            <span className="utility-sep">&bull;</span>
            <span className="utility-location">📍 Nairobi Hub</span>
          </div>
        </div>
      </aside>

      {/* Main Glassmorphic Header */}
      <header className="navbar-header">
        <div className="navbar-container">
          {/* Brand Identity */}
          <div className="navbar-brand">
            <Link href="/" className="brand-link">
              <div className="brand-emblem-box">
                <span className="brand-emblem">🏺</span>
              </div>
              <div className="brand-naming">
                <div className="brand-title-line">
                  <span className="brand-name-text">Artisan Market</span>
                  <span className="brand-accent-dot"></span>
                </div>
                <span className="brand-tagline">Soko la Sanaa la Kenya</span>
              </div>
            </Link>
          </div>

          {/* Center Segmented Navigation */}
          <nav className="navbar-nav-group" aria-label="Main Navigation">
            <Link
              href="/"
              className={`nav-pill-item ${pathname === '/' ? 'active' : ''}`}
            >
              <span className="nav-pill-icon">🛍️</span>
              <span>Catalogue</span>
            </Link>
            <Link
              href="/seller"
              className={`nav-pill-item ${pathname === '/seller' ? 'active' : ''}`}
            >
              <span className="nav-pill-icon">👨‍🎨</span>
              <span>Artisan Portal</span>
            </Link>
          </nav>

          {/* Right Action Suite */}
          <div className="navbar-actions">
            {/* Dark / Light Mode Switcher */}
            <button
              type="button"
              className="theme-switch-btn"
              onClick={toggleTheme}
              aria-label={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
              title={`Switch to ${theme === 'light' ? 'dark' : 'light'} mode`}
            >
              <span className="theme-switch-icon">{theme === 'light' ? '🌙' : '☀️'}</span>
              <span className="theme-switch-label">{theme === 'light' ? 'Dark' : 'Light'}</span>
            </button>

            {/* Shopping Cart Pill Button */}
            <button
              id="cart-toggle"
              className="navbar-cart-btn"
              onClick={() => setIsCartOpen(true)}
              aria-label="View Shopping Cart"
            >
              <span className="cart-icon">🛒</span>
              <span className="cart-label">Cart</span>
              <span className="cart-count-badge">{itemCount}</span>
            </button>
          </div>
        </div>
      </header>

      {/* Toast Notification */}
      {toastMessage && (
        <div className="toast-notification">
          <span className="toast-icon">✨</span>
          <span>{toastMessage}</span>
        </div>
      )}

      <CartDrawer />
      <CheckoutModal />
    </>
  );
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <title>Artisan Market &bull; Authentic Kenyan Handcrafted Masterpieces</title>
        <meta
          name="description"
          content="Soko la Sanaa la Kenya. Shop authentic handcrafted goods directly from Nairobi, Machakos, Mombasa, and Kisii artisans with Safaricom Daraja M-Pesa."
        />
        <meta name="theme-color" content="#c2410c" />
      </head>
      <body>
        <ThemeProvider>
          <CartProvider>
            <Navigation />
            <main className="main-content">{children}</main>
            <footer className="site-footer">
              <div className="footer-container">
                <div className="footer-brand-column">
                  <div className="footer-logo">🏺 Artisan Market</div>
                  <p className="footer-bio">
                    Direct marketplace connecting independent Kenyan master artisans to global collectors.
                    Supporting sustainable community guilds across Machakos, Kisii, Mombasa, and Nairobi.
                  </p>
                  <div className="footer-badges">
                    <span className="footer-badge">✓ 100% Kenyan Made</span>
                    <span className="footer-badge">✓ Fair Trade Certified</span>
                    <span className="footer-badge">✓ Safaricom M-Pesa Verified</span>
                  </div>
                </div>

                <div className="footer-nav-column">
                  <h4>Artisan Craft Guilds</h4>
                  <ul>
                    <li>Ukambani Woodcarvers Cooperative</li>
                    <li>Tabaka Kisii Soapstone Quarries</li>
                    <li>Kariokor Sisal & Leather Artisans</li>
                    <li>Lake Victoria Reedcraft Collective</li>
                  </ul>
                </div>

                <div className="footer-nav-column">
                  <h4>Payments & Security</h4>
                  <ul>
                    <li>Safaricom Daraja M-Pesa (STK Push)</li>
                    <li>Stripe Global Card Payments</li>
                    <li>Direct B2C Artisan Mobile Disbursements</li>
                    <li>Zero Platform Middleman Markups</li>
                  </ul>
                </div>
              </div>

              <div className="footer-bottom">
                <p>
                  &copy; {new Date().getFullYear()} Artisan Market Kenya. Built with pride in Nairobi. Handcrafted in Africa &bull; Delivered worldwide.
                </p>
              </div>
            </footer>
          </CartProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
