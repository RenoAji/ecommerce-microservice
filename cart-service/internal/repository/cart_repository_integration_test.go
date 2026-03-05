//go:build integration
// +build integration

package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cart-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

func openCartRedis(t *testing.T) *redis.Client {
	t.Helper()

	host := getenv("REDIS_HOST", "localhost")
	port := getenv("REDIS_PORT", "6379")
	password := os.Getenv("REDIS_PASSWORD")

	client := redis.NewClient(&redis.Options{Addr: host + ":" + port, Password: password, DB: 0})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("skipping integration test, cannot connect to redis: %v", err)
	}
	return client
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func TestRedisCartRepository_SaveGetAndClear_Integration(t *testing.T) {
	client := openCartRedis(t)
	repo := NewRedisCartRepository(client)

	userID := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	item := &domain.CartItem{ProductID: 12, Name: "Keyboard", Quantity: 2, Price: 500}

	ctx := context.Background()
	if err := repo.SaveCart(ctx, userID, item); err != nil {
		t.Fatalf("SaveCart() error = %v", err)
	}

	items, err := repo.GetCart(ctx, userID)
	if err != nil {
		t.Fatalf("GetCart() error = %v", err)
	}
	if len(items) != 1 || items[0].ProductID != 12 {
		t.Fatalf("unexpected cart items: %#v", items)
	}

	if err := repo.ClearCart(ctx, userID); err != nil {
		t.Fatalf("ClearCart() error = %v", err)
	}
}
