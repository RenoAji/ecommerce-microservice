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
	// Serialize the items payload for stream transport.
	itemsJSON, err := json.Marshal(event.Items)
	if err != nil {
		return err
	}

	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = correlationIDFromContext(ctx)
	}

	msg := map[string]interface{}{
		"order_id":       event.OrderID,
		"user_id":        event.UserID,
		"total_amount":   event.TotalAmount,
		"items":          string(itemsJSON),
		"created_at":     time.Now().Format(time.RFC3339),
		"correlation_id": correlationID,
	}

	// Add to Stream
	return r.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "stream:orders:created",
		MaxLen: 1000,
		Approx: true,
		Values: msg,
	}).Err()
}

func (r *RedisRepository) PublishOrderPaidEvent(ctx context.Context, event *domain.OrderEvent) error {
	// Serialize the items payload for stream transport.
	itemsJSON, err := json.Marshal(event.Items)
	if err != nil {
		return err
	}

	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = correlationIDFromContext(ctx)
	}

	msg := map[string]interface{}{
		"order_id":       event.OrderID,
		"user_id":        event.UserID,
		"total_amount":   event.TotalAmount,
		"items":          string(itemsJSON),
		"created_at":     time.Now().Format(time.RFC3339),
		"correlation_id": correlationID,
	}

	// Add to Stream
	return r.redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "stream:orders:paid",
		MaxLen: 1000,
		Approx: true,
		Values: msg,
	}).Err()
}

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	return ""
}
