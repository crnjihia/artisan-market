// path: frontend/README.md
# Artisan Market Storefront (`frontend`)

Next.js 14 (App Router) storefront and artisan seller dashboard for the Kenyan Artisan Marketplace. Features authentic handcrafted goods from Nairobi, Machakos, Mombasa, and Kisii, with an optimistic cart state, Safaricom Daraja M-Pesa STK push checkout, live payment polling, and bilingual (EN/SW) interfaces.

## Features
- **Artisan Catalogue**: Browse handcrafted goods across multiple Kenyan crafts (Baskets & Bags, Jewelry, Kitchenware, Textiles, Sculptures) priced in Kenya Shillings (`KES`).
- **Product Detail Page**: Detailed artisan biographical attribution, materials provenance, and workshop stock levels.
- **Optimistic Cart**: Instant client-side cart updates with asynchronous reservation validation and rollback on stock exhaustion.
- **Safaricom M-Pesa Checkout**: Modal checkout with Kenyan mobile number normalization (`+254...`, `07...`), STK push simulation/polling, and live confirmation with M-Pesa receipt numbers.
- **Artisan Seller Dashboard**: Real-time sales statistics, active listings, instant M-Pesa B2C payout status, and listing creation.

## Commands

```bash
# Install dependencies
npm install

# Run local development server (http://localhost:3050)
npm run dev

# Production build & verification
npm run build
```

