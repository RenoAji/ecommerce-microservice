//go:build integration
// +build integration

package repository

import (
	"fmt"
	"os"
	"testing"
	"time"
	"user-service/internal/domain"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func openUserTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	host := getenv("USER_DB_HOST_TEST", "localhost")
	port := getenv("USER_DB_PORT_TEST", "5432")
	user := getenv("USER_DB_USER_TEST", "user")
	password := getenv("USER_DB_PASSWORD_TEST", "secretpassword")
	dbname := getenv("USER_DB_NAME_TEST", "user_db_test")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
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
