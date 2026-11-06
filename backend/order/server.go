package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	orderpb "github.com/yourorg/artisan-market/proto/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderServiceServer implements orderpb.OrderServiceServer.
type OrderServiceServer struct {
	orderpb.UnimplementedOrderServiceServer
	db *sql.DB
}

// NewOrderServiceServer constructs an OrderServiceServer instance.
func NewOrderServiceServer(db *sql.DB) *OrderServiceServer {
	return &OrderServiceServer{db: db}
}

// CreateOrder initiates a new order aggregate and records an order.created outbox event.
func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if len(req.GetItems()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one item is required")
	}
	method := strings.ToLower(strings.TrimSpace(req.GetPaymentMethod()))
	if method != "mpesa" && method != "stripe" {
		return nil, status.Error(codes.InvalidArgument, "payment_method must be 'mpesa' or 'stripe'")
	}
	phone := strings.TrimSpace(req.GetCustomerPhone())
	if phone == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_phone is required")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	orderID := uuid.New().String()
	correlationID := uuid.New().String()

	// Calculate total amount and prepare items
	var totalCents int64
	type itemData struct {
		productID string
		name      string
		quantity  int32
		price     int64
	}
	var items []itemData

	for _, reqItem := range req.GetItems() {
		if reqItem.GetQuantity() <= 0 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid quantity %d for item %s", reqItem.GetQuantity(), reqItem.GetProductId())
		}
		// Default price approximation if product service is async; typically 2500 KES (250000 cents)
		unitPrice := int64(250000)
		productName := "Handcrafted Artisan Item"

		itemTotal := unitPrice * int64(reqItem.GetQuantity())
		totalCents += itemTotal

		items = append(items, itemData{
			productID: reqItem.GetProductId(),
			name:      productName,
			quantity:  reqItem.GetQuantity(),
			price:     unitPrice,
		})
	}

	// 1. Insert order record
	insertOrderQuery := `
		INSERT INTO orders (id, user_id, total_cents, currency, status, payment_method, customer_phone, customer_email, saga_step)
		VALUES ($1, $2, $3, 'KES', 'pending', $4, $5, $6, 'ORDER_CREATED')
	`
	if _, err := tx.ExecContext(ctx, insertOrderQuery, orderID, req.GetUserId(), totalCents, method, phone, req.GetCustomerEmail()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert order: %v", err)
	}

	// 2. Insert order items
	insertItemQuery := `
		INSERT INTO order_items (order_id, product_id, product_name, quantity, unit_price_cents)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, it := range items {
		if _, err := tx.ExecContext(ctx, insertItemQuery, orderID, it.productID, it.name, it.quantity, it.price); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to insert order item: %v", err)
		}
	}

	// 3. Atomically write order.created domain event to outbox table
	outboxPayload := map[string]any{
		"order_id":       orderID,
		"user_id":        req.GetUserId(),
		"total_cents":    totalCents,
		"currency":       "KES",
		"payment_method": method,
		"customer_phone": phone,
		"customer_email": req.GetCustomerEmail(),
		"items":          items,
	}

	if err := QueueOutboxEvent(ctx, tx, orderID, "order.created", "order.events", outboxPayload, correlationID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record outbox event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit order transaction: %v", err)
	}

	return &orderpb.CreateOrderResponse{
		OrderId:    orderID,
		Status:     "pending",
		TotalCents: totalCents,
	}, nil
}

// GetOrder returns order aggregate details with line items.
func (s *OrderServiceServer) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	var o orderpb.Order
	var email sql.NullString
	var createdAt, updatedAt time.Time

	query := `
		SELECT id, user_id, total_cents, currency, status, payment_method, customer_phone, customer_email, saga_step, created_at, updated_at
		FROM orders
		WHERE id = $1
	`
	err := s.db.QueryRowContext(ctx, query, req.GetOrderId()).Scan(
		&o.Id, &o.UserId, &o.TotalCents, &o.Currency, &o.Status,
		&o.PaymentMethod, &o.CustomerPhone, &email, &o.SagaStep, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetOrderId())
		}
		return nil, status.Errorf(codes.Internal, "database error querying order: %v", err)
	}

	o.CustomerEmail = email.String
	o.CreatedAtUnixMs = createdAt.UnixMilli()
	o.UpdatedAtUnixMs = updatedAt.UnixMilli()

	// Query line items
	itemRows, err := s.db.QueryContext(ctx, "SELECT product_id, product_name, quantity, unit_price_cents FROM order_items WHERE order_id = $1", req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error querying order items: %v", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item orderpb.OrderItem
		if err := itemRows.Scan(&item.ProductId, &item.ProductName, &item.Quantity, &item.UnitPriceCents); err != nil {
			return nil, status.Errorf(codes.Internal, "error scanning item row: %v", err)
		}
		o.Items = append(o.Items, &item)
	}

	return &orderpb.GetOrderResponse{Order: &o}, nil
}

// ListOrders returns all orders for a buyer.
func (s *OrderServiceServer) ListOrders(ctx context.Context, req *orderpb.ListOrdersRequest) (*orderpb.ListOrdersResponse, error) {
	limit := req.GetLimit()
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := req.GetOffset()
	if offset < 0 {
		offset = 0
	}

	var conditions []string
	var args []any
	argIdx := 1

	if req.GetUserId() != "" {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIdx))
		args = append(args, req.GetUserId())
		argIdx++
	}

	whereClause := "1=1"
	if len(conditions) > 0 {
		whereClause = strings.Join(conditions, " AND ")
	}

	var total int32
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM orders WHERE %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "database error counting orders: %v", err)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, total_cents, currency, status, payment_method, customer_phone, customer_email, saga_step, created_at, updated_at
		FROM orders
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error listing orders: %v", err)
	}
	defer rows.Close()

	var orders []*orderpb.Order
	for rows.Next() {
		var o orderpb.Order
		var email sql.NullString
		var createdAt, updatedAt time.Time

		if err := rows.Scan(
			&o.Id, &o.UserId, &o.TotalCents, &o.Currency, &o.Status,
			&o.PaymentMethod, &o.CustomerPhone, &email, &o.SagaStep, &createdAt, &updatedAt,
		); err != nil {
			return nil, status.Errorf(codes.Internal, "row scan error: %v", err)
		}
		o.CustomerEmail = email.String
		o.CreatedAtUnixMs = createdAt.UnixMilli()
		o.UpdatedAtUnixMs = updatedAt.UnixMilli()
		orders = append(orders, &o)
	}

	return &orderpb.ListOrdersResponse{
		Orders: orders,
		Total:  total,
	}, nil
}

// CancelOrder marks an order as cancelled and writes an order.cancelled event to outbox to trigger inventory release.
func (s *OrderServiceServer) CancelOrder(ctx context.Context, req *orderpb.CancelOrderRequest) (*orderpb.CancelOrderResponse, error) {
	if req.GetOrderId() == "" {
		return nil, status.Error(codes.InvalidArgument, "order_id is required")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	query := `UPDATE orders SET status = 'cancelled', saga_step = 'ORDER_CANCELLED', updated_at = now() WHERE id = $1`
	res, err := tx.ExecContext(ctx, query, req.GetOrderId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to cancel order: %v", err)
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "order %s not found", req.GetOrderId())
	}

	// Outbox event to trigger compensation in product service
	payload := map[string]any{
		"order_id": req.GetOrderId(),
		"reason":   req.GetReason(),
	}
	if err := QueueOutboxEvent(ctx, tx, req.GetOrderId(), "order.cancelled", "order.events", payload, uuid.New().String()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to record cancellation outbox: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit cancellation: %v", err)
	}

	return &orderpb.CancelOrderResponse{
		Success: true,
		Status:  "cancelled",
	}, nil
}
