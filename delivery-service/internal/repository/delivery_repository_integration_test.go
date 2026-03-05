//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"delivery-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openDeliveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := getenv("DELIVERY_DB_HOST", "localhost")
	port := getenv("DELIVERY_DB_PORT", "5436")
	user := getenv("DB_USER", "user")
	password := getenv("DB_PASSWORD", "secretpassword")
	dbname := getenv("DELIVERY_DB_NAME", "delivery_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to delivery-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.Delivery{}, &domain.DeliveryOutboxMessage{}); err != nil {
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

func TestDeliveryRepository_CreateUpdateAndOutbox_Integration(t *testing.T) {
	db := openDeliveryTestDB(t)
	repo := NewPostgresRepository(db)

	delivery := &domain.Delivery{OrderID: uint(time.Now().UnixNano() % 100000), Status: "RECEIVED"}
	ctx := context.Background()
	if err := repo.CreateDelivery(ctx, delivery); err != nil {
		t.Fatalf("CreateDelivery() error = %v", err)
	}

	if err := repo.UpdateDeliveryStatus(delivery.ID, "DELIVERED"); err != nil {
		t.Fatalf("UpdateDeliveryStatus() error = %v", err)
	}

	if err := repo.CreateOutboxMessage(ctx, "delivered", &domain.Delivery{ID: delivery.ID, OrderID: delivery.OrderID}); err != nil {
		t.Fatalf("CreateOutboxMessage() error = %v", err)
	}

	pending, err := repo.GetPendingOutboxMessages(ctx)
	if err != nil {
		t.Fatalf("GetPendingOutboxMessages() error = %v", err)
	}
	if len(pending) == 0 {
		t.Fatal("expected at least one pending outbox message")
	}

	if err := repo.MarkOutboxMessageAsPublished(ctx, pending[0].ID); err != nil {
		t.Fatalf("MarkOutboxMessageAsPublished() error = %v", err)
	}

	updated, err := repo.GetDeliveryByID(delivery.ID)
	if err != nil {
		t.Fatalf("GetDeliveryByID() error = %v", err)
	}
	if updated.Status != "DELIVERED" {
		t.Fatalf("expected status DELIVERED, got %s", updated.Status)
	}
}
