package repository

import (
	"context"
	"encoding/json"
	"order-service/internal/domain"
	"time"

	"github.com/redis/go-redis/v9"
)

type OrderEventRepository interface {
	PublishOrderCreatedEvent(ctx context.Context, event *domain.OrderEvent) error
    PublishOrderPaidEvent(ctx context.Context, event *domain.OrderEvent) error
}

type RedisRepository struct {
	redisClient *redis.Client
}

func NewRedisRepository(redisClient *redis.Client) *RedisRepository {
	return &RedisRepository{redisClient: redisClient}
}

func (r *RedisRepository) PublishOrderCreatedEvent(ctx context.Context, event *domain.OrderEvent) error {
    // Serialize the items to JSON or a flat map
    itemsJson, _ := json.Marshal(event.Items)

    msg := map[string]interface{}{
        "order_id":     event.OrderID,
        "user_id":      event.UserID,
        "total_amount": event.TotalAmount,
        "items":        string(itemsJson),
        "created_at":   time.Now().Format(time.RFC3339),
    }

    // Add to Stream
    err := r.redisClient.XAdd(ctx, &redis.XAddArgs{
        Stream: "stream:orders:created",
        MaxLen: 1000, 
        Approx: true,
        Values: msg,
    }).Err()

    return err
}

func (r *RedisRepository) PublishOrderPaidEvent(ctx context.Context, event *domain.OrderEvent) error {
    // Serialize the items to JSON or a flat map
    itemsJson, _ := json.Marshal(event.Items)

    msg := map[string]interface{}{
        "order_id":     event.OrderID,
        "user_id":      event.UserID,
        "total_amount": event.TotalAmount,
        "items":        string(itemsJson),
        "created_at":   time.Now().Format(time.RFC3339),
    }

    // Add to Stream
    err := r.redisClient.XAdd(ctx, &redis.XAddArgs{
        Stream: "stream:orders:paid",
        MaxLen: 1000, 
        Approx: true,
        Values: msg,
    }).Err()

    return err
}