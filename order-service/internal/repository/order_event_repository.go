package repository

import (
	"context"
	"encoding/json"
	"order-service/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderEventRepository interface {
	PublishOrderCreatedEvent(ctx context.Context, event *domain.OrderCreatedEvent) error
}

type RedisRepository struct {
	redisClient *redis.Client
}

func NewRedisRepository(redisClient *redis.Client) *RedisRepository {
	return &RedisRepository{redisClient: redisClient}
}

func (r *RedisRepository) PublishOrderCreatedEvent(ctx context.Context, event *domain.OrderCreatedEvent) error {
    // 1. Serialize the items to JSON or a flat map
    // Redis Streams values must be strings, integers, or floats
    itemsJson, _ := json.Marshal(event.Items)

    msg := map[string]interface{}{
        "order_id":     event.OrderID,
        "user_id":      event.UserID,
        "total_amount": event.TotalAmount,
        "items":        string(itemsJson),
        "created_at":   time.Now().Format(time.RFC3339),
    }

    // 2. Add to Stream
    // "*" tells Redis to auto-generate a unique Message ID (e.g., 1672531200000-0)
    err := r.redisClient.XAdd(ctx, &redis.XAddArgs{
        Stream: "stream:orders:created",
        MaxLen: 1000, // Optional: Keep only the last 1000 messages to save RAM
        Approx: true,
        Values: msg,
    }).Err()

    return err
}