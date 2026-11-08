import { MOCK_PRODUCTS } from '../../../src/data/mockProducts';
import ProductDetailClient from './ProductDetailClient';

export function generateStaticParams() {
  return MOCK_PRODUCTS.map((product) => ({ id: product.id }));
}

export default function ProductDetailPage() {
  return <ProductDetailClient />;
}
