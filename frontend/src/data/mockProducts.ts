// path: frontend/src/data/mockProducts.ts

import { Product } from '../types';

export const MOCK_PRODUCTS: Product[] = [
  {
    id: 'prod-1',
    name: 'Heritage Handwoven Sisal Weave Storage Baskets',
    swahiliName: 'Vikapu vya Asili vya Mkonge vya Ukambani',
    description: 'Set of two resilient storage baskets crafted by skilled Kamba weavers using hand-stripped natural sisal fiber, dyed with indigenous tea leaves and wattle bark.',
    price_cents: 380000, // KES 3,800
    category: 'Baskets & Bags',
    artisan_name: 'Mama Grace Wanjiku',
    artisan_location: 'Machakos Town',
    stock_quantity: 14,
    image_url: '/images/products/Sisal_Weave_Baskets.jpg',
  },
  {
    id: 'prod-2',
    name: 'Recycled Wool & Sisal Artisan Kiondo Tote Bag',
    swahiliName: 'Kiondo cha Sufu na Mkonge chenye Ngozi',
    description: 'Contemporary eco-friendly artisan tote blending warm recycled highlands merino wool with strong sisal weaving and supple vegetable-tanned Kenyan cowhide handles.',
    price_cents: 450000, // KES 4,500
    category: 'Baskets & Bags',
    artisan_name: 'Agnes Kalekye',
    artisan_location: 'Kariokor Market, Nairobi',
    stock_quantity: 9,
    image_url: '/images/products/recycled-wool-sisal-bag.jpeg',
  },
  {
    id: 'prod-3',
    name: 'Hand-Polished Coastal Coconut Shell Serving Bowls',
    swahiliName: 'Bakuli za Nazi za Upwa wa Bahari',
    description: 'Reclaimed coconut shells gathered from coastal palm groves along the Indian Ocean, hand-sanded to a mirror sheen and protected with organic cold-pressed coconut oil.',
    price_cents: 160000, // KES 1,600
    category: 'Kitchenware',
    artisan_name: 'Ali Juma Omari',
    artisan_location: 'Diani Beach, Kwale',
    stock_quantity: 22,
    image_url: '/images/products/Coconut_Shell_Bowls.jpg',
  },
  {
    id: 'prod-4',
    name: 'Carved Wild Olive Wood Cooking Utensil Set (3-Piece)',
    swahiliName: 'Miiko ya Mzeituni ya Kupikia',
    description: 'Carved by master carvers from pruned Kenyan wild olive hardwood (Olea europaea). Features dense marbling, natural antibacterial oils, and velvet-smooth ergonomic handles.',
    price_cents: 240000, // KES 2,400
    category: 'Kitchenware',
    artisan_name: 'John Mutua',
    artisan_location: 'Wamunyu Woodcarvers, Machakos',
    stock_quantity: 18,
    image_url: '/images/products/Olive_Wood_Cooking_Utensil.jpg',
  },
  {
    id: 'prod-5',
    name: 'Artisan Studio Ceramic Coffee Mug with Woven Base',
    swahiliName: 'Kikombe cha Udongo chenye Kikapu cha Rattan',
    description: 'High-fired stoneware clay mug glazed in a warm speckled oatmeal finish, cradled in a removable handwoven reed coaster base handcrafted in the Great Rift Valley.',
    price_cents: 185000, // KES 1,850
    category: 'Ceramics',
    artisan_name: 'Chebet Kiprono',
    artisan_location: 'Rift Valley Pottery, Nakuru',
    stock_quantity: 16,
    image_url: '/images/products/WeaveSkart_Coffee_Mug.jpg',
  },
  {
    id: 'prod-6',
    name: 'Hand-Carved Mahogany Animal Amulet & Bead Necklace',
    swahiliName: 'Mkufu wa Hirizi ya Wanyama na Shaba',
    description: 'Reclaimed African mahogany carved into wildlife talisman charms, combined with sand-cast Kibera brass beads on an adjustable hand-waxed Kenyan cotton cord.',
    price_cents: 220000, // KES 2,200
    category: 'Jewelry',
    artisan_name: 'Kev Omondi',
    artisan_location: 'Kibera Arts Collective, Nairobi',
    stock_quantity: 26,
    image_url: '/images/products/Wooden_Necklace_Animal_Amulet.jpg',
  },
  {
    id: 'prod-7',
    name: 'Heritage Handwoven Maasai Highland Shúkà Blanket',
    swahiliName: 'Shuka ya Kimasai ya Pamba Nzito',
    description: 'Iconic double-sided brushed acrylic check blanket in rich highland scarlet and cobalt hues, traditional to pastoralist communities and exceptionally warm for cool evenings.',
    price_cents: 520000, // KES 5,200
    category: 'Textiles',
    artisan_name: 'Naisula Lemayian',
    artisan_location: 'Narok Cultural Centre',
    stock_quantity: 12,
    image_url: '/images/products/handwoven-blankets.jpg',
  },
  {
    id: 'prod-8',
    name: 'Dahlia Blossom Pure Cotton Crochet Accent Cushion',
    swahiliName: 'Mto wa Maua wa Pamba ya Thika',
    description: 'Intricately stitched floral dahlia motif crocheted by hand using 100% Kenyan ring-spun combed cotton. Adds three-dimensional tactile warmth and artisanal elegance to any sofa.',
    price_cents: 320000, // KES 3,200
    category: 'Home & Decor',
    artisan_name: 'Esther Mwangi',
    artisan_location: 'Thika Weavers Guild',
    stock_quantity: 11,
    image_url: '/images/products/Dahlia_Pattern_Cotton_Crochet_Cushion.jpg',
  },
  {
    id: 'prod-9',
    name: 'Hand-Loomed Highland Artisan Poncho Wrap',
    swahiliName: 'Vazi la Sweta la Milimani la Sufu',
    description: 'Dense heavyweight wool-blend wrap woven on traditional wooden floor looms at the foot of Mount Kenya. Finished with knotted fringe and dyed with earth mineral pigments.',
    price_cents: 680000, // KES 6,800
    category: 'Apparel',
    artisan_name: 'Wambui Kanyi',
    artisan_location: 'Nanyuki Spinners & Weavers',
    stock_quantity: 7,
    image_url: '/images/products/Navajo_Poncho.jpg',
  },
  {
    id: 'prod-10',
    name: 'Tokra Handwoven Bamboo & Rattan Pendant Light',
    swahiliName: 'Taa ya Kusuka ya Rattan na Mianzi',
    description: 'Architectural statement lampshade hand-woven from sun-cured Lake Victoria river reeds and split bamboo cane. Filters ceiling light into soft, radiant organic lattice patterns.',
    price_cents: 750000, // KES 7,500
    category: 'Lighting & Decor',
    artisan_name: 'Peter Otieno',
    artisan_location: 'Kisumu Lakecraft Workshop',
    stock_quantity: 5,
    image_url: '/images/products/Tokra_Pendant_Light.png',
  },
  {
    id: 'prod-11',
    name: 'Mimic Handcrafted Hardwood & Cane Accent Chair',
    swahiliName: 'Kiti cha Mbao Ngumu na Rattan cha Kisasa',
    description: 'Mid-century silhouette built from sustainably seasoned Kenyan cypress with a hand-woven octagonal natural cane backrest. High-resilience linen upholstered seat.',
    price_cents: 2400000, // KES 24,000
    category: 'Furniture',
    artisan_name: 'David Ndung\'u',
    artisan_location: 'Industrial Area Craft Guild, Nairobi',
    stock_quantity: 4,
    image_url: '/images/products/Mimic_Office_Chair.png',
  },
  {
    id: 'prod-12',
    name: 'Artisan Farmhouse Marinated Buffalo Cheese (350g Jar)',
    swahiliName: 'Jibini ya Nyati iliyotiwa Vikolezo vya Asili',
    description: 'Creamy small-batch buffalo milk curd steeped in cold-pressed virgin olive oil, wild Mount Kenya rosemary, crushed garlic, and whole pink peppercorns.',
    price_cents: 145000, // KES 1,450
    category: 'Pantry',
    artisan_name: 'Beatrice Chepkoech',
    artisan_location: 'Naivasha Artisan Dairies',
    stock_quantity: 20,
    image_url: '/images/products/Marinated_Buffalo_Cheese.jpg',
  },
];
