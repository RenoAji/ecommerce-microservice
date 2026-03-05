//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"order-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openOrderTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := getenv("ORDER_DB_HOST", "localhost")
	port := getenv("ORDER_DB_PORT", "5434")
	user := getenv("DB_USER", "user")
	password := getenv("DB_PASSWORD", "secretpassword")
	dbname := getenv("ORDER_DB_NAME", "order_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to order-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.Order{}, &domain.OrderItem{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestOrderRepository_AddAndGetByID_Integration(t *testing.T) {
	db := openOrderTestDB(t)
	repo := NewPostgresRepository(db)

	order := &domain.Order{
		UserID:      77,
		TotalAmount: 3500,
		Status:      "RECEIVED",
		Items: []domain.OrderItem{{ProductID: uint(time.Now().UnixNano() % 100000), Name: "item", Quantity: 2, Price: 1750}},
	}

	if err := repo.AddOrder(context.Background(), order); err != nil {
		t.Fatalf("AddOrder() error = %v", err)
	}

	got, err := repo.GetOrderByID(context.Background(), fmt.Sprintf("%d", order.ID))
	if err != nil {
		t.Fatalf("GetOrderByID() error = %v", err)
	}
	if got.ID != order.ID {
		t.Fatalf("expected order ID %d, got %d", order.ID, got.ID)
	}
}
