//go:build integration
// +build integration

package repository

import (
	"fmt"
	"os"
	"testing"
	"time"

	"product-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openProductTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := getenv("PRODUCT_DB_HOST", "localhost")
	port := getenv("PRODUCT_DB_PORT", "5433")
	user := getenv("DB_USER", "user")
	password := getenv("DB_PASSWORD", "secretpassword")
	dbname := getenv("PRODUCT_DB_NAME", "product_db")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to product-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}); err != nil {
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

func TestProductRepository_CreateAndGet_Integration(t *testing.T) {
	db := openProductTestDB(t)
	repo := NewPostgresRepository(db)

	category := &domain.Category{Name: fmt.Sprintf("cat-%d", time.Now().UnixNano())}
	if err := repo.CreateCategory(category); err != nil {
		t.Fatalf("CreateCategory() error = %v", err)
	}

	req := &domain.CreateProductRequest{
		Name:        fmt.Sprintf("product-%d", time.Now().UnixNano()),
		Description: "integration",
		Price:       1999,
		Stock:       7,
		CategoryIDs: []uint{category.ID},
	}
	if err := repo.SaveProduct(req); err != nil {
		t.Fatalf("SaveProduct() error = %v", err)
	}

	products, _, err := repo.ListAll("", "", "", "", "", "", 1, 10)
	if err != nil {
		t.Fatalf("ListAll() error = %v", err)
	}
	if len(products) == 0 {
		t.Fatal("expected at least one product in list")
	}
}
