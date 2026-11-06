// path: frontend/app/seller/page.tsx
'use client';

import React, { useState } from 'react';
import { MOCK_PRODUCTS } from '../../src/data/mockProducts';
import { Product } from '../../src/types';

export default function SellerDashboard() {
  const [products, setProducts] = useState<Product[]>(MOCK_PRODUCTS);
  const [showAddForm, setShowAddForm] = useState(false);

  const [newName, setNewName] = useState('');
  const [newCat, setNewCat] = useState('Baskets & Bags');
  const [newPriceKes, setNewPriceKes] = useState('');
  const [newStock, setNewStock] = useState('10');
  const [newDesc, setNewDesc] = useState('');

  const handleCreateProduct = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newName || !newPriceKes) return;

    const newProd: Product = {
      id: `prod-${Date.now()}`,
      name: newName,
      description: newDesc || 'Locally crafted authentic Kenyan artisan piece.',
      price_cents: Math.round(parseFloat(newPriceKes) * 100),
      category: newCat,
      artisan_name: 'Christopher Njihia (Nairobi Workshop)',
      artisan_location: 'Nairobi, Kenya',
      stock_quantity: parseInt(newStock, 10) || 5,
      image_url: '/images/products/Sisal_Weave_Baskets.jpg',
    };

    setProducts([newProd, ...products]);
    setNewName('');
    setNewPriceKes('');
    setNewDesc('');
    setShowAddForm(false);
  };

  return (
    <div className="container" style={{ marginTop: '1rem' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '2rem', flexWrap: 'wrap', gap: '1rem' }}>
        <div>
          <h1 style={{ fontSize: '2.25rem', fontWeight: 900, color: 'var(--text-main)', letterSpacing: '-0.5px' }}>
            Artisan Seller Dashboard
          </h1>
          <p style={{ color: 'var(--text-muted)', marginTop: '0.25rem' }}>
            Karibu, <strong>Christopher Njihia</strong> &bull; Nairobi Artisan Studio (Vendor ID: VEND-254-01)
          </p>
        </div>

        <button
          type="button"
          className="add-to-cart-action-btn"
          style={{ padding: '0.75rem 1.4rem' }}
          onClick={() => setShowAddForm(!showAddForm)}
        >
          {showAddForm ? '✕ Close Form' : '+ Add New Listing (Weka Bidhaa)'}
        </button>
      </div>

      {/* Stats Overview */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))', gap: '1.5rem', marginBottom: '2.5rem' }}>
        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border)', borderRadius: '16px', padding: '1.5rem', boxShadow: 'var(--card-shadow)' }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>Total Sales (Mauzo)</span>
          <div style={{ fontSize: '1.75rem', fontWeight: 900, color: 'var(--primary)', margin: '0.35rem 0' }}>KES 284,500</div>
          <span style={{ fontSize: '0.75rem', color: 'var(--secondary)', fontWeight: 700 }}>↑ +24% this month</span>
        </div>

        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border)', borderRadius: '16px', padding: '1.5rem', boxShadow: 'var(--card-shadow)' }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>Active Listings</span>
          <div style={{ fontSize: '1.75rem', fontWeight: 900, color: 'var(--text-main)', margin: '0.35rem 0' }}>{products.length} Items</div>
          <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Across 9 Kenyan crafts</span>
        </div>

        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border)', borderRadius: '16px', padding: '1.5rem', boxShadow: 'var(--card-shadow)' }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>Orders Completed</span>
          <div style={{ fontSize: '1.75rem', fontWeight: 900, color: 'var(--text-main)', margin: '0.35rem 0' }}>52 Orders</div>
          <span style={{ fontSize: '0.75rem', color: 'var(--secondary)', fontWeight: 700 }}>100% Fulfilled</span>
        </div>

        <div style={{ background: 'var(--surface-card)', border: '1px solid var(--border)', borderRadius: '16px', padding: '1.5rem', boxShadow: 'var(--card-shadow)' }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>M-Pesa Payout Account</span>
          <div style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--text-main)', margin: '0.5rem 0' }}>+254 712 345 678</div>
          <span style={{ fontSize: '0.75rem', color: 'var(--accent)', fontWeight: 700 }}>⚡ Instant Daraja B2C</span>
        </div>
      </div>

      {/* New Product Form */}
      {showAddForm && (
        <div style={{ background: 'var(--surface-card)', padding: '2rem', borderRadius: '18px', border: '1px solid var(--border)', marginBottom: '2.5rem', boxShadow: 'var(--card-shadow)' }}>
          <h3 style={{ marginBottom: '1.25rem', fontWeight: 800, color: 'var(--text-main)' }}>Ongeza Bidhaa Mpya (Create New Product)</h3>
          <form onSubmit={handleCreateProduct}>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))', gap: '1.25rem', marginBottom: '1.25rem' }}>
              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Product Title:
                </label>
                <input
                  type="text"
                  required
                  style={{ width: '100%', padding: '0.75rem', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)', color: 'var(--text-main)', outline: 'none' }}
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="e.g. Hand-carved Ebony Elephant"
                />
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Category:
                </label>
                <select
                  style={{ width: '100%', padding: '0.75rem', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)', color: 'var(--text-main)', outline: 'none' }}
                  value={newCat}
                  onChange={(e) => setNewCat(e.target.value)}
                >
                  <option value="Baskets & Bags">Baskets & Bags</option>
                  <option value="Kitchenware">Kitchenware</option>
                  <option value="Ceramics">Ceramics</option>
                  <option value="Jewelry">Jewelry</option>
                  <option value="Textiles">Textiles</option>
                  <option value="Home & Decor">Home & Decor</option>
                  <option value="Lighting & Decor">Lighting & Decor</option>
                  <option value="Furniture">Furniture</option>
                  <option value="Apparel">Apparel</option>
                  <option value="Pantry">Pantry</option>
                </select>
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Price (KES):
                </label>
                <input
                  type="number"
                  required
                  min="50"
                  style={{ width: '100%', padding: '0.75rem', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)', color: 'var(--text-main)', outline: 'none' }}
                  value={newPriceKes}
                  onChange={(e) => setNewPriceKes(e.target.value)}
                  placeholder="3500"
                />
              </div>

              <div>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                  Stock Quantity:
                </label>
                <input
                  type="number"
                  required
                  min="1"
                  style={{ width: '100%', padding: '0.75rem', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)', color: 'var(--text-main)', outline: 'none' }}
                  value={newStock}
                  onChange={(e) => setNewStock(e.target.value)}
                />
              </div>
            </div>

            <div style={{ marginBottom: '1.25rem' }}>
              <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 700, marginBottom: '0.4rem', color: 'var(--text-main)' }}>
                Description & Craft Materials:
              </label>
              <textarea
                rows={3}
                style={{ width: '100%', padding: '0.75rem', border: '1px solid var(--border)', borderRadius: '8px', background: 'var(--surface)', color: 'var(--text-main)', outline: 'none' }}
                value={newDesc}
                onChange={(e) => setNewDesc(e.target.value)}
                placeholder="Handcrafted using locally sourced reclaimed materials..."
              />
            </div>

            <button
              type="submit"
              className="confirm-btn"
              style={{ padding: '0.75rem 2rem', width: 'auto' }}
            >
              Publish Listing (Chapisha)
            </button>
          </form>
        </div>
      )}

      {/* Inventory Table */}
      <h3 style={{ marginBottom: '1.25rem', fontWeight: 800, color: 'var(--text-main)' }}>Your Inventory & Catalogue</h3>
      <div style={{ background: 'var(--surface-card)', borderRadius: '18px', border: '1px solid var(--border)', overflow: 'hidden', boxShadow: 'var(--card-shadow)' }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
          <thead>
            <tr style={{ background: 'var(--surface-subtle)', borderBottom: '1px solid var(--border)', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
              <th style={{ padding: '1rem 1.25rem' }}>Bidhaa (Product)</th>
              <th style={{ padding: '1rem 1.25rem' }}>Aina (Category)</th>
              <th style={{ padding: '1rem 1.25rem' }}>Bei (Price)</th>
              <th style={{ padding: '1rem 1.25rem' }}>Idadi (Stock)</th>
              <th style={{ padding: '1rem 1.25rem' }}>Hali (Status)</th>
            </tr>
          </thead>
          <tbody>
            {products.map((item) => (
              <tr key={item.id} style={{ borderBottom: '1px solid var(--divider)' }}>
                <td style={{ padding: '1rem 1.25rem', display: 'flex', alignItems: 'center', gap: '1rem' }}>
                  <img
                    src={item.image_url}
                    alt={item.name}
                    style={{ width: '48px', height: '48px', objectFit: 'cover', borderRadius: '8px', border: '1px solid var(--border)' }}
                  />
                  <div>
                    <strong style={{ color: 'var(--text-main)', display: 'block', fontSize: '0.95rem' }}>{item.name}</strong>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>📍 {item.artisan_location}</div>
                  </div>
                </td>
                <td style={{ padding: '1rem 1.25rem', color: 'var(--text-muted)', fontSize: '0.9rem' }}>{item.category}</td>
                <td style={{ padding: '1rem 1.25rem', color: 'var(--primary)', fontWeight: 800 }}>
                  KES {(item.price_cents / 100).toLocaleString('en-KE')}
                </td>
                <td style={{ padding: '1rem 1.25rem', color: 'var(--text-muted)', fontSize: '0.9rem' }}>{item.stock_quantity} available</td>
                <td style={{ padding: '1rem 1.25rem' }}>
                  <span
                    className={`card-badge ${item.stock_quantity > 5 ? 'stock-available' : 'stock-low'}`}
                    style={{ position: 'static' }}
                  >
                    {item.stock_quantity > 5 ? 'In Stock' : 'Low Stock'}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
