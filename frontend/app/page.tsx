// path: frontend/app/page.tsx
'use client';

import React, { useState, useMemo } from 'react';
import Link from 'next/link';
import { MOCK_PRODUCTS } from '../src/data/mockProducts';
import { useCart } from '../src/context/CartContext';
import { Product } from '../src/types';
import { publicAssetPath } from '../src/publicAssets';

export default function HomePage() {
  const { addToCart } = useCart();
  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [sortBy, setSortBy] = useState<'featured' | 'price-asc' | 'price-desc' | 'location'>('featured');

  // Compute category counts dynamically
  const categoriesWithCounts = useMemo(() => {
    const counts: Record<string, number> = { All: MOCK_PRODUCTS.length };
    MOCK_PRODUCTS.forEach((p) => {
      counts[p.category] = (counts[p.category] || 0) + 1;
    });
    const uniqueCategories = ['All', ...Array.from(new Set(MOCK_PRODUCTS.map((p) => p.category)))];
    return uniqueCategories.map((cat) => ({
      name: cat,
      count: counts[cat] || 0,
    }));
  }, []);

  // Filter and Sort products
  const filteredProducts = useMemo(() => {
    return MOCK_PRODUCTS.filter((product) => {
      const matchesCategory =
        selectedCategory === 'All' || product.category === selectedCategory;
      const q = searchQuery.toLowerCase().trim();
      const matchesSearch =
        !q ||
        product.name.toLowerCase().includes(q) ||
        product.artisan_name.toLowerCase().includes(q) ||
        product.artisan_location.toLowerCase().includes(q) ||
        product.category.toLowerCase().includes(q) ||
        (product.swahiliName && product.swahiliName.toLowerCase().includes(q)) ||
        product.description.toLowerCase().includes(q);
      return matchesCategory && matchesSearch;
    }).sort((a, b) => {
      if (sortBy === 'price-asc') return a.price_cents - b.price_cents;
      if (sortBy === 'price-desc') return b.price_cents - a.price_cents;
      if (sortBy === 'location') return a.artisan_location.localeCompare(b.artisan_location);
      return 0; // 'featured' retains curated order
    });
  }, [selectedCategory, searchQuery, sortBy]);

  return (
    <div className="storefront-page">
      {/* Compact Hero Banner Section */}
      <section className="hero-section compact-hero">
        <div className="hero-content">
          <div className="hero-pill">
            <span className="pill-dot"></span>
            Direct From Kenyan Artisan Workshops
          </div>
          <h1 className="hero-title">
            Handcrafted in Kenya, <span className="hero-highlight">Cherished Worldwide</span>
          </h1>
          <p className="hero-subtitle">
            Bidhaa halisi za kitamaduni kutoka kwa mafundi mashuhuri wa Nairobi, Machakos, Mombasa, Nakuru na Kisii.
            Every purchase directly funds artisan community cooperatives with instant Safaricom M-Pesa.
          </p>

          <div className="hero-stats-bar">
            <div className="stat-item">
              <span className="stat-number">12</span>
              <span className="stat-label">Master Crafts</span>
            </div>
            <div className="stat-divider"></div>
            <div className="stat-item">
              <span className="stat-number">100%</span>
              <span className="stat-label">Ethical Direct Trade</span>
            </div>
            <div className="stat-divider"></div>
            <div className="stat-item">
              <span className="stat-number">M-PESA</span>
              <span className="stat-label">Live STK Push Checkout</span>
            </div>
          </div>
        </div>
      </section>

      {/* Main Product Catalogue Section */}
      <div className="container">
        {/* Controls, Filters & Search Bar */}
        <div className="catalogue-controls">
          <div className="search-bar-wrapper">
            <span className="search-icon">🔍</span>
            <input
              type="text"
              className="search-input"
              placeholder="Search products, materials, or artisans (Tafuta bidhaa)..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              aria-label="Search artisan products"
            />
            {searchQuery && (
              <button
                type="button"
                className="clear-search-btn"
                onClick={() => setSearchQuery('')}
                aria-label="Clear search"
              >
                ✕
              </button>
            )}
          </div>

          <div className="sort-selector-wrapper">
            <label htmlFor="sort-dropdown" className="sort-label">Sort by:</label>
            <select
              id="sort-dropdown"
              className="sort-dropdown"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
            >
              <option value="featured">Featured Curated</option>
              <option value="price-asc">Price: Low to High</option>
              <option value="price-desc">Price: High to Low</option>
              <option value="location">Artisan Region</option>
            </select>
          </div>
        </div>

        {/* Category Pills Bar */}
        <div className="category-scroll-container">
          <div className="category-pills">
            {categoriesWithCounts.map((cat) => (
              <button
                key={cat.name}
                type="button"
                className={`category-pill ${selectedCategory === cat.name ? 'active' : ''}`}
                onClick={() => setSelectedCategory(cat.name)}
              >
                <span className="pill-name">{cat.name}</span>
                <span className="pill-badge">{cat.count}</span>
              </button>
            ))}
          </div>
        </div>

        {/* Results Metadata */}
        <div className="results-header">
          <div className="results-count">
            Showing <strong>{filteredProducts.length}</strong> of {MOCK_PRODUCTS.length} authentic pieces
            {selectedCategory !== 'All' && <span> in <em>{selectedCategory}</em></span>}
            {searchQuery && <span> matching &ldquo;{searchQuery}&rdquo;</span>}
          </div>
          {(selectedCategory !== 'All' || searchQuery) && (
            <button
              type="button"
              className="reset-filters-link"
              onClick={() => {
                setSelectedCategory('All');
                setSearchQuery('');
              }}
            >
              Reset all filters
            </button>
          )}
        </div>

        {/* Product Cards Grid (4x3 on Desktop) */}
        {filteredProducts.length > 0 ? (
          <div className="product-grid">
            {filteredProducts.map((product) => (
              <article key={product.id} className="product-card">
                {/* Image & Badges Container */}
                <div className="product-image-frame">
                  <Link href={`/products/${product.id}`} className="image-link" tabIndex={-1}>
                    <img
                      src={publicAssetPath(product.image_url)}
                      alt={product.name}
                      className="product-card-image"
                      loading="lazy"
                    />
                  </Link>

                  {/* Top Left: Category Badge */}
                  <span className="card-badge category-badge">
                    {product.category}
                  </span>

                  {/* Top Right: Stock Status */}
                  <span
                    className={`card-badge stock-badge ${
                      product.stock_quantity <= 5 ? 'stock-low' : 'stock-available'
                    }`}
                  >
                    {product.stock_quantity <= 5
                      ? `⚡ Only ${product.stock_quantity} left`
                      : `✓ In Stock (${product.stock_quantity})`}
                  </span>
                </div>

                {/* Card Content & Details */}
                <div className="product-card-body">
                  {product.swahiliName && (
                    <div className="product-swahili-tag">
                      {product.swahiliName}
                    </div>
                  )}

                  <Link href={`/products/${product.id}`} className="title-link">
                    <h3 className="product-card-title" title={product.name}>
                      {product.name}
                    </h3>
                  </Link>

                  {/* Artisan & Region Provenance */}
                  <div className="product-artisan-provenance">
                    <span className="artisan-icon">👨‍🎨</span>
                    <span className="artisan-text">
                      <strong>{product.artisan_name}</strong>
                    </span>
                    <span className="provenance-dot">&bull;</span>
                    <span className="location-text">
                      📍 {product.artisan_location}
                    </span>
                  </div>

                  {/* Description Snippet */}
                  <p className="product-card-snippet">
                    {product.description}
                  </p>

                  {/* Footer / Price & Add to Cart */}
                  <div className="product-card-footer">
                    <div className="price-container">
                      <span className="price-currency">KES</span>
                      <span className="price-amount">
                        {(product.price_cents / 100).toLocaleString('en-KE')}
                      </span>
                    </div>

                    <button
                      type="button"
                      className="add-to-cart-action-btn"
                      onClick={() => addToCart(product)}
                      aria-label={`Add ${product.name} to cart`}
                    >
                      <span className="action-btn-icon">🛒</span>
                      <span>Add to Cart</span>
                    </button>
                  </div>
                </div>
              </article>
            ))}
          </div>
        ) : (
          /* Empty Search State */
          <div className="empty-state-box">
            <div className="empty-icon">🏺</div>
            <h3 className="empty-title">No artisan pieces match your search</h3>
            <p className="empty-subtitle">
              We couldn&rsquo;t find anything matching &ldquo;{searchQuery}&rdquo;. Try another craft category or clear your search term.
            </p>
            <button
              type="button"
              className="btn-primary"
              onClick={() => {
                setSelectedCategory('All');
                setSearchQuery('');
              }}
            >
              Show All 12 Artisan Masterpieces
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
