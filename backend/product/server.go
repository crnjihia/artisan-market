package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	productpb "github.com/yourorg/artisan-market/proto/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ProductServiceServer implements productpb.ProductServiceServer.
type ProductServiceServer struct {
	productpb.UnimplementedProductServiceServer
	db *sql.DB
}

// NewProductServiceServer constructs a ProductServiceServer instance.
func NewProductServiceServer(db *sql.DB) *ProductServiceServer {
	return &ProductServiceServer{db: db}
}

// CreateProduct inserts a new artisan product into the catalog.
func (s *ProductServiceServer) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	if req.GetSellerId() == "" {
		return nil, status.Error(codes.InvalidArgument, "seller_id is required")
	}
	name := strings.TrimSpace(req.GetName())
	if name == "" {
		return nil, status.Error(codes.InvalidArgument, "product name is required")
	}
	if req.GetPriceCents() < 0 {
		return nil, status.Error(codes.InvalidArgument, "price_cents cannot be negative")
	}
	if req.GetInventory() < 0 {
		return nil, status.Error(codes.InvalidArgument, "inventory cannot be negative")
	}
	category := strings.TrimSpace(req.GetCategory())
	if category == "" {
		category = "Handcraft"
	}
	originRegion := strings.TrimSpace(req.GetOriginRegion())
	if originRegion == "" {
		originRegion = "Kenya"
	}

	var productID string
	var createdAt, updatedAt time.Time

	query := `
		INSERT INTO products (seller_id, name, description, price_cents, inventory, category, image_url, origin_region)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	err := s.db.QueryRowContext(
		ctx, query,
		req.GetSellerId(), name, req.GetDescription(), req.GetPriceCents(),
		req.GetInventory(), category, req.GetImageUrl(), originRegion,
	).Scan(&productID, &createdAt, &updatedAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error inserting product: %v", err)
	}

	product := &productpb.Product{
		Id:               productID,
		SellerId:         req.GetSellerId(),
		Name:             name,
		Description:      req.GetDescription(),
		PriceCents:       req.GetPriceCents(),
		Inventory:        req.GetInventory(),
		Category:         category,
		ImageUrl:         req.GetImageUrl(),
		OriginRegion:     originRegion,
		CreatedAtUnixMs: createdAt.UnixMilli(),
		UpdatedAtUnixMs: updatedAt.UnixMilli(),
	}

	return &productpb.CreateProductResponse{
		ProductId: productID,
		Product:   product,
	}, nil
}

// GetProduct retrieves a single product by its unique UUID.
func (s *ProductServiceServer) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	if req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	var p productpb.Product
	var desc, img sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT id, seller_id, name, description, price_cents, inventory, category, image_url, origin_region, created_at, updated_at
		FROM products
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, req.GetProductId()).Scan(
		&p.Id, &p.SellerId, &p.Name, &desc, &p.PriceCents, &p.Inventory, &p.Category, &img, &p.OriginRegion, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "product %s not found", req.GetProductId())
		}
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	p.Description = desc.String
	p.ImageUrl = img.String
	p.CreatedAtUnixMs = createdAt.UnixMilli()
	p.UpdatedAtUnixMs = updatedAt.UnixMilli()

	return &productpb.GetProductResponse{Product: &p}, nil
}

// ListProducts returns paginated and searchable products.
func (s *ProductServiceServer) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}

	conditions := []string{"1=1"}
	args := []any{}
	argIdx := 1

	if cat := strings.TrimSpace(req.GetCategory()); cat != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, cat)
		argIdx++
	}

	if q := strings.TrimSpace(req.GetQuery()); q != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d OR origin_region ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+q+"%")
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Total count
	var total int32
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM products WHERE %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "database error counting products: %v", err)
	}

	// Paginated query
	query := fmt.Sprintf(`
		SELECT id, seller_id, name, description, price_cents, inventory, category, image_url, origin_region, created_at, updated_at
		FROM products
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error querying products: %v", err)
	}
	defer rows.Close()

	var products []*productpb.Product
	for rows.Next() {
		var p productpb.Product
		var desc, img sql.NullString
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&p.Id, &p.SellerId, &p.Name, &desc, &p.PriceCents, &p.Inventory, &p.Category, &img, &p.OriginRegion, &createdAt, &updatedAt,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "row scan error: %v", err)
		}

		p.Description = desc.String
		p.ImageUrl = img.String
		p.CreatedAtUnixMs = createdAt.UnixMilli()
		p.UpdatedAtUnixMs = updatedAt.UnixMilli()
		products = append(products, &p)
	}

	return &productpb.ListProductsResponse{
		Products: products,
		Total:    total,
	}, nil
}

// UpdateProduct updates product details.
func (s *ProductServiceServer) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	if req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if req.GetPriceCents() < 0 {
		return nil, status.Error(codes.InvalidArgument, "price_cents cannot be negative")
	}
	if req.GetInventory() < 0 {
		return nil, status.Error(codes.InvalidArgument, "inventory cannot be negative")
	}

	var p productpb.Product
	var desc, img sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		UPDATE products
		SET name = $1, description = $2, price_cents = $3, inventory = $4, category = $5, image_url = $6, updated_at = now()
		WHERE id = $7
		RETURNING id, seller_id, name, description, price_cents, inventory, category, image_url, origin_region, created_at, updated_at
	`
	err := s.db.QueryRowContext(
		ctx, query,
		req.GetName(), req.GetDescription(), req.GetPriceCents(), req.GetInventory(), req.GetCategory(), req.GetImageUrl(), req.GetProductId(),
	).Scan(
		&p.Id, &p.SellerId, &p.Name, &desc, &p.PriceCents, &p.Inventory, &p.Category, &img, &p.OriginRegion, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "product %s not found", req.GetProductId())
		}
		return nil, status.Errorf(codes.Internal, "database error updating product: %v", err)
	}

	p.Description = desc.String
	p.ImageUrl = img.String
	p.CreatedAtUnixMs = createdAt.UnixMilli()
	p.UpdatedAtUnixMs = updatedAt.UnixMilli()

	return &productpb.UpdateProductResponse{Product: &p}, nil
}

// DeleteProduct removes a product from the database.
func (s *ProductServiceServer) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}

	res, err := s.db.ExecContext(ctx, "DELETE FROM products WHERE id = $1", req.GetProductId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error deleting product: %v", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "product %s not found", req.GetProductId())
	}

	return &productpb.DeleteProductResponse{Success: true}, nil
}

// ReserveInventory reserves stock for items in an order using PostgreSQL row-level locks (SELECT ... FOR UPDATE).
// If any item has insufficient inventory, the entire transaction is rolled back with ResourceExhausted.
func (s *ProductServiceServer) ReserveInventory(ctx context.Context, req *productpb.ReserveInventoryRequest) (*productpb.ReserveInventoryResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}
	if len(req.GetItems()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "items list cannot be empty")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	reservationID := uuid.New().String()

	for _, item := range req.GetItems() {
		if item.GetQuantity() <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid quantity %d for product %s", item.GetQuantity(), item.GetProductId())
		}

		// Row-level exclusive lock prevents race conditions and overselling
		var currentInventory int32
		var productName string
		selectQuery := `SELECT inventory, name FROM products WHERE id = $1 FOR UPDATE`
		err := tx.QueryRowContext(ctx, selectQuery, item.GetProductId()).Scan(&currentInventory, &productName)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, status.Errorf(codes.NotFound, "product %s not found", item.GetProductId())
			}
			return nil, status.Errorf(codes.Internal, "failed to lock product %s: %v", item.GetProductId(), err)
		}

		if currentInventory < item.GetQuantity() {
			return nil, status.Errorf(
				codes.ResourceExhausted,
				"insufficient inventory for product '%s' (%s): available %d, requested %d",
				productName, item.GetProductId(), currentInventory, item.GetQuantity(),
			)
		}

		// Decrement inventory
		updateQuery := `UPDATE products SET inventory = inventory - $1, updated_at = now() WHERE id = $2`
		if _, err := tx.ExecContext(ctx, updateQuery, item.GetQuantity(), item.GetProductId()); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decrement inventory: %v", err)
		}

		// Record reservation
		insertResQuery := `
			INSERT INTO inventory_reservations (id, order_id, product_id, quantity, status)
			VALUES ($1, $2, $3, $4, 'reserved')
		`
		if _, err := tx.ExecContext(ctx, insertResQuery, reservationID, req.GetOrderId(), item.GetProductId(), item.GetQuantity()); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to record reservation: %v", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit inventory reservation: %v", err)
	}

	return &productpb.ReserveInventoryResponse{
		ReservationId: reservationID,
		Success:       true,
	}, nil
}

// ReleaseInventory restores stock for an order compensation when a payment fails or times out.
func (s *ProductServiceServer) ReleaseInventory(ctx context.Context, req *productpb.ReleaseInventoryRequest) (*productpb.ReleaseInventoryResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Find active reservations
	selectQuery := `
		SELECT product_id, quantity
		FROM inventory_reservations
		WHERE order_id = $1 AND status = 'reserved'
		FOR UPDATE
	`
	rows, err := tx.QueryContext(ctx, selectQuery, req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query reservations: %v", err)
	}
	defer rows.Close()

	type releaseItem struct {
		productID string
		quantity  int32
	}
	var itemsToRelease []releaseItem

	for rows.Next() {
		var item releaseItem
		if err := rows.Scan(&item.productID, &item.quantity); err != nil {
			return nil, status.Errorf(codes.Internal, "error scanning reservation row: %v", err)
		}
		itemsToRelease = append(itemsToRelease, item)
	}

	// Restore product inventory
	for _, item := range itemsToRelease {
		updateStockQuery := `UPDATE products SET inventory = inventory + $1, updated_at = now() WHERE id = $2`
		if _, err := tx.ExecContext(ctx, updateStockQuery, item.quantity, item.productID); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to restore inventory for product %s: %v", item.productID, err)
		}
	}

	// Mark reservation status as released
	updateResQuery := `UPDATE inventory_reservations SET status = 'released', updated_at = now() WHERE order_id = $1 AND status = 'reserved'`
	if _, err := tx.ExecContext(ctx, updateResQuery, req.GetOrderId()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update reservation status: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit inventory release: %v", err)
	}

	return &productpb.ReleaseInventoryResponse{Success: true}, nil
}
