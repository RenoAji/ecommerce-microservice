//go:build integration
// +build integration

package repository

import (
	"fmt"
	"os"
	"testing"
	"time"
	"user-service/internal/config"
	"user-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := config.LoadTestConfig()

	dsn := cfg.GetDSN()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Skipf("skipping integration test, cannot connect to user-db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
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

func TestUserRepository_SaveAndFindByEmail_Integration(t *testing.T) {
	db := openUserTestDB(t)
	repo := NewPostgresRepository(db)

	email := fmt.Sprintf("integration-%d@example.com", time.Now().UnixNano())
	user := &domain.User{Username: email, Email: email, Password: "hashed", Role: "user"}

	if err := repo.Save(user); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := repo.FindByEmail(email)
	if err != nil {
		t.Fatalf("FindByEmail() error = %v", err)
	}
	if got.Email != email {
		t.Fatalf("expected email %s, got %s", email, got.Email)
	}
}
