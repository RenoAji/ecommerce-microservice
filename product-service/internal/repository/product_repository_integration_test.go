//go:build integration
// +build integration

package repository

import (
	"fmt"
	"testing"
	"time"

	"product-service/internal/config"
	"product-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openProductTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := config.LoadTestConfig()
	dsn := cfg.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to product-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.Category{}, &domain.Product{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
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
