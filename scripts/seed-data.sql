-- path: scripts/seed-data.sql
-- Seed 12 authentic Kenyan artisan products matching local imagery
INSERT INTO products (id, seller_id, name, description, price_cents, inventory, category, image_url, origin_region)
VALUES 
(
    '00000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    'Heritage Handwoven Sisal Weave Storage Baskets',
    'Set of two resilient storage baskets crafted by skilled Kamba weavers using hand-stripped natural sisal fiber, dyed with indigenous tea leaves and wattle bark.',
    380000,
    14,
    'Baskets & Bags',
    '/images/products/Sisal_Weave_Baskets.jpg',
    'Machakos'
),
(
    '00000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    'Recycled Wool & Sisal Artisan Kiondo Tote Bag',
    'Contemporary eco-friendly artisan tote blending warm recycled highlands merino wool with strong sisal weaving and supple vegetable-tanned Kenyan cowhide handles.',
    450000,
    9,
    'Baskets & Bags',
    '/images/products/recycled-wool-sisal-bag.jpeg',
    'Nairobi'
),
(
    '00000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000002',
    'Hand-Polished Coastal Coconut Shell Serving Bowls',
    'Reclaimed coconut shells gathered from coastal palm groves along the Indian Ocean, hand-sanded to a mirror sheen and protected with organic cold-pressed coconut oil.',
    160000,
    22,
    'Kitchenware',
    '/images/products/Coconut_Shell_Bowls.jpg',
    'Kwale'
),
(
    '00000000-0000-0000-0000-000000000004',
    'a0000000-0000-0000-0000-000000000002',
    'Carved Wild Olive Wood Cooking Utensil Set (3-Piece)',
    'Carved by master carvers from pruned Kenyan wild olive hardwood (Olea europaea). Features dense marbling, natural antibacterial oils, and velvet-smooth ergonomic handles.',
    240000,
    18,
    'Kitchenware',
    '/images/products/Olive_Wood_Cooking_Utensil.jpg',
    'Machakos'
),
(
    '00000000-0000-0000-0000-000000000005',
    'a0000000-0000-0000-0000-000000000003',
    'Artisan Studio Ceramic Coffee Mug with Woven Base',
    'High-fired stoneware clay mug glazed in a warm speckled oatmeal finish, cradled in a removable handwoven reed coaster base handcrafted in the Great Rift Valley.',
    185000,
    16,
    'Ceramics',
    '/images/products/WeaveSkart_Coffee_Mug.jpg',
    'Nakuru'
),
(
    '00000000-0000-0000-0000-000000000006',
    'a0000000-0000-0000-0000-000000000003',
    'Hand-Carved Mahogany Animal Amulet & Bead Necklace',
    'Reclaimed African mahogany carved into wildlife talisman charms, combined with sand-cast Kibera brass beads on an adjustable hand-waxed Kenyan cotton cord.',
    220000,
    26,
    'Jewelry',
    '/images/products/Wooden_Necklace_Animal_Amulet.jpg',
    'Nairobi'
),
(
    '00000000-0000-0000-0000-000000000007',
    'a0000000-0000-0000-0000-000000000004',
    'Heritage Handwoven Maasai Highland Shúkà Blanket',
    'Iconic double-sided brushed acrylic check blanket in rich highland scarlet and cobalt hues, traditional to pastoralist communities and exceptionally warm for cool evenings.',
    520000,
    12,
    'Textiles',
    '/images/products/handwoven-blankets.jpg',
    'Narok'
),
(
    '00000000-0000-0000-0000-000000000008',
    'a0000000-0000-0000-0000-000000000004',
    'Dahlia Blossom Pure Cotton Crochet Accent Cushion',
    'Intricately stitched floral dahlia motif crocheted by hand using 100% Kenyan ring-spun combed cotton. Adds three-dimensional tactile warmth and artisanal elegance to any sofa.',
    320000,
    11,
    'Home & Decor',
    '/images/products/Dahlia_Pattern_Cotton_Crochet_Cushion.jpg',
    'Thika'
),
(
    '00000000-0000-0000-0000-000000000009',
    'a0000000-0000-0000-0000-000000000005',
    'Hand-Loomed Highland Artisan Poncho Wrap',
    'Dense heavyweight wool-blend wrap woven on traditional wooden floor looms at the foot of Mount Kenya. Finished with knotted fringe and dyed with earth mineral pigments.',
    680000,
    7,
    'Apparel',
    '/images/products/Navajo_Poncho.jpg',
    'Nanyuki'
),
(
    '00000000-0000-0000-0000-000000000010',
    'a0000000-0000-0000-0000-000000000005',
    'Tokra Handwoven Bamboo & Rattan Pendant Light',
    'Architectural statement lampshade hand-woven from sun-cured Lake Victoria river reeds and split bamboo cane. Filters ceiling light into soft, radiant organic lattice patterns.',
    750000,
    5,
    'Lighting & Decor',
    '/images/products/Tokra_Pendant_Light.png',
    'Kisumu'
),
(
    '00000000-0000-0000-0000-000000000011',
    'a0000000-0000-0000-0000-000000000005',
    'Mimic Handcrafted Hardwood & Cane Accent Chair',
    'Mid-century silhouette built from sustainably seasoned Kenyan cypress with a hand-woven octagonal natural cane backrest. High-resilience linen upholstered seat.',
    2400000,
    4,
    'Furniture',
    '/images/products/Mimic_Office_Chair.png',
    'Nairobi'
),
(
    '00000000-0000-0000-0000-000000000012',
    'a0000000-0000-0000-0000-000000000005',
    'Artisan Farmhouse Marinated Buffalo Cheese (350g Jar)',
    'Creamy small-batch buffalo milk curd steeped in cold-pressed virgin olive oil, wild Mount Kenya rosemary, crushed garlic, and whole pink peppercorns.',
    145000,
    20,
    'Pantry',
    '/images/products/Marinated_Buffalo_Cheese.jpg',
    'Naivasha'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    price_cents = EXCLUDED.price_cents,
    inventory = EXCLUDED.inventory,
    category = EXCLUDED.category,
    image_url = EXCLUDED.image_url,
    origin_region = EXCLUDED.origin_region;
