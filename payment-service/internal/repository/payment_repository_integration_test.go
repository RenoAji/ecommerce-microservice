//go:build integration
// +build integration

package repository

import (
	"fmt"
	"os"
	"testing"
	"time"

	"payment-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openPaymentTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := getenv("PAYMENT_DB_HOST", "localhost")
	port := getenv("PAYMENT_DB_PORT", "5435")
	user := getenv("DB_USER", "user")
	password := getenv("DB_PASSWORD", "secretpassword")
	dbname := getenv("PAYMENT_DB_NAME", "payment_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to payment-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.Payment{}); err != nil {
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

func TestPaymentRepository_AddAndUpdateStatus_Integration(t *testing.T) {
	db := openPaymentTestDB(t)
	repo := NewPostgresRepository(db)

	orderID := uint(time.Now().UnixNano() % 100000)
	if err := repo.AddPayment(orderID, 9999, "https://example.com/pay", "PENDING"); err != nil {
		t.Fatalf("AddPayment() error = %v", err)
	}

	if err := repo.UpdatePaymentStatus(orderID, "SUCCESS"); err != nil {
		t.Fatalf("UpdatePaymentStatus() error = %v", err)
	}

	got, err := repo.GetPaymentByOrderID(orderID)
	if err != nil {
		t.Fatalf("GetPaymentByOrderID() error = %v", err)
	}
	if got.Status != "SUCCESS" {
		t.Fatalf("expected payment status SUCCESS, got %s", got.Status)
	}
}
