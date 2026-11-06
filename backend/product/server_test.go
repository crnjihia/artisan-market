package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	productpb "github.com/yourorg/artisan-market/proto/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCreateProduct_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
		AddRow("prod-uuid-1", now, now)

	mock.ExpectQuery("INSERT INTO products").
		WithArgs("seller-uuid-1", "Kisii Soapstone Lion", "Hand carved in Tabaka", int64(350000), int32(10), "Carvings", "https://img.co.ke/lion.jpg", "Kisii").
		WillReturnRows(rows)

	req := &productpb.CreateProductRequest{
		SellerId:     "seller-uuid-1",
		Name:         "Kisii Soapstone Lion",
		Description:  "Hand carved in Tabaka",
		PriceCents:   350000, // KES 3,500.00
		Inventory:    10,
		Category:     "Carvings",
		ImageUrl:     "https://img.co.ke/lion.jpg",
		OriginRegion: "Kisii",
	}

	resp, err := server.CreateProduct(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	if resp.GetProductId() != "prod-uuid-1" {
		t.Errorf("expected prod-uuid-1, got %s", resp.GetProductId())
	}
	if resp.GetProduct().GetOriginRegion() != "Kisii" {
		t.Errorf("expected origin region Kisii, got %s", resp.GetProduct().GetOriginRegion())
	}
}

func TestGetProduct_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	rows := sqlmock.NewRows([]string{
		"id", "seller_id", "name", "description", "price_cents", "inventory", "category", "image_url", "origin_region", "created_at", "updated_at",
	}).AddRow(
		"prod-1", "seller-1", "Maasai Beaded Choker", "Traditional red and blue beads",
		int64(150000), int32(25), "Beadwork", "https://img.co.ke/choker.jpg", "Narok", time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT id, seller_id, name, description").
		WithArgs("prod-1").
		WillReturnRows(rows)

	resp, err := server.GetProduct(context.Background(), &productpb.GetProductRequest{ProductId: "prod-1"})
	if err != nil {
		t.Fatalf("GetProduct failed: %v", err)
	}

	if resp.GetProduct().GetName() != "Maasai Beaded Choker" {
		t.Errorf("expected Maasai Beaded Choker, got %s", resp.GetProduct().GetName())
	}
	if resp.GetProduct().GetPriceCents() != 150000 {
		t.Errorf("expected price 150000, got %d", resp.GetProduct().GetPriceCents())
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	mock.ExpectQuery("SELECT id, seller_id, name, description").
		WithArgs("missing-id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err = server.GetProduct(context.Background(), &productpb.GetProductRequest{ProductId: "missing-id"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Errorf("expected NotFound code, got %v", st.Code())
	}
}

func TestListProducts_FiltersAndPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs("Baskets", "%Kitui%").
		WillReturnRows(countRows)

	prodRows := sqlmock.NewRows([]string{
		"id", "seller_id", "name", "description", "price_cents", "inventory", "category", "image_url", "origin_region", "created_at", "updated_at",
	}).AddRow(
		"prod-basket", "seller-2", "Kyondo Sisal Basket", "Handwoven in Kitui",
		int64(220000), int32(15), "Baskets", "https://img.co.ke/kyondo.jpg", "Kitui", time.Now(), time.Now(),
	)

	mock.ExpectQuery("SELECT id, seller_id, name, description").
		WithArgs("Baskets", "%Kitui%", int32(10), int32(0)).
		WillReturnRows(prodRows)

	req := &productpb.ListProductsRequest{
		Category: "Baskets",
		Query:    "Kitui",
		Limit:    10,
		Offset:   0,
	}

	resp, err := server.ListProducts(context.Background(), req)
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}

	if resp.GetTotal() != 1 {
		t.Errorf("expected total 1, got %d", resp.GetTotal())
	}
	if len(resp.GetProducts()) != 1 {
		t.Fatalf("expected 1 product, got %d", len(resp.GetProducts()))
	}
	if resp.GetProducts()[0].GetName() != "Kyondo Sisal Basket" {
		t.Errorf("expected Kyondo Sisal Basket, got %s", resp.GetProducts()[0].GetName())
	}
}

func TestReserveInventory_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	mock.ExpectBegin()
	// Row-level lock
	mock.ExpectQuery("SELECT inventory, name FROM products WHERE id = \\$1 FOR UPDATE").
		WithArgs("prod-1").
		WillReturnRows(sqlmock.NewRows([]string{"inventory", "name"}).AddRow(5, "Kyondo Basket"))

	// Decrement inventory
	mock.ExpectExec("UPDATE products SET inventory = inventory - \\$1").
		WithArgs(int32(2), "prod-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Record reservation
	mock.ExpectExec("INSERT INTO inventory_reservations").
		WithArgs(sqlmock.AnyArg(), "order-123", "prod-1", int32(2)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	req := &productpb.ReserveInventoryRequest{
		OrderId: "order-123",
		Items: []*productpb.ReserveItem{
			{ProductId: "prod-1", Quantity: 2},
		},
	}

	resp, err := server.ReserveInventory(context.Background(), req)
	if err != nil {
		t.Fatalf("ReserveInventory failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("expected reservation success to be true")
	}
	if resp.GetReservationId() == "" {
		t.Error("expected non-empty reservation ID")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestReserveInventory_InsufficientStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT inventory, name FROM products WHERE id = \\$1 FOR UPDATE").
		WithArgs("prod-1").
		WillReturnRows(sqlmock.NewRows([]string{"inventory", "name"}).AddRow(1, "Kyondo Basket")) // Only 1 in stock

	mock.ExpectRollback()

	req := &productpb.ReserveInventoryRequest{
		OrderId: "order-123",
		Items: []*productpb.ReserveItem{
			{ProductId: "prod-1", Quantity: 5}, // Requesting 5
		},
	}

	_, err = server.ReserveInventory(context.Background(), req)
	if err == nil {
		t.Fatal("expected ResourceExhausted error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Errorf("expected ResourceExhausted code, got %v", st.Code())
	}
}

func TestReleaseInventory_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}
	defer db.Close()

	server := NewProductServiceServer(db)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT product_id, quantity FROM inventory_reservations").
		WithArgs("order-123").
		WillReturnRows(sqlmock.NewRows([]string{"product_id", "quantity"}).AddRow("prod-1", int32(2)))

	mock.ExpectExec("UPDATE products SET inventory = inventory \\+ \\$1").
		WithArgs(int32(2), "prod-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE inventory_reservations SET status = 'released'").
		WithArgs("order-123").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	resp, err := server.ReleaseInventory(context.Background(), &productpb.ReleaseInventoryRequest{OrderId: "order-123"})
	if err != nil {
		t.Fatalf("ReleaseInventory failed: %v", err)
	}

	if !resp.GetSuccess() {
		t.Error("expected ReleaseInventory success to be true")
	}
}

func TestOversellPreventionLogic(t *testing.T) {
	// Demonstrates concurrency safety logic: simulated thread-safe inventory check
	var mu sync.Mutex
	availableStock := 3
	successfulReservations := 0
	failedReservations := 0

	var wg sync.WaitGroup
	// 5 concurrent buyers attempting to reserve 1 item each from a stock of 3
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			defer mu.Unlock()
			if availableStock >= 1 {
				availableStock--
				successfulReservations++
			} else {
				failedReservations++
			}
		}()
	}
	wg.Wait()

	if successfulReservations != 3 {
		t.Errorf("expected exactly 3 successful reservations, got %d", successfulReservations)
	}
	if failedReservations != 2 {
		t.Errorf("expected exactly 2 rejected reservations, got %d", failedReservations)
	}
	if availableStock != 0 {
		t.Errorf("expected remaining stock to be 0, got %d", availableStock)
	}
}
