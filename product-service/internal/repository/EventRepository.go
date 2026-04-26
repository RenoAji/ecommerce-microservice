package repository

import (
	"context"
	"product-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type EventRepository interface {
	PublishStockReservedEvent(ctx context.Context, events *domain.StockEvent) error
	PublishStockInsufficientEvent(ctx context.Context, events *domain.StockEvent) error
}

type RedisRepository struct {
	redisClient *redis.Client
}

func NewRedisRepository(redisClient *redis.Client) *RedisRepository {
	return &RedisRepository{redisClient: redisClient}
}

func (r *RedisRepository) PublishStockReservedEvent(ctx context.Context, event *domain.StockEvent) error {
	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = correlationIDFromContext(ctx)
	}

	msg := map[string]interface{}{
		"order_id":       event.OrderID,
		"correlation_id": correlationID,
	}

	err := r.redisClient.XAdd(
		ctx,
		&redis.XAddArgs{
			Stream: "stream:stock:reserved",
			MaxLen: 1000,
			Approx: true,
			Values: msg,
		},
	).Err()

	return err
}

func (r *RedisRepository) PublishStockInsufficientEvent(ctx context.Context, event *domain.StockEvent) error {
	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = correlationIDFromContext(ctx)
	}

	msg := map[string]interface{}{
		"order_id":       event.OrderID,
		"correlation_id": correlationID,
	}

	err := r.redisClient.XAdd(
		ctx,
		&redis.XAddArgs{
			Stream: "stream:stock:insufficient",
			MaxLen: 1000,
			Approx: true,
			Values: msg,
		},
	).Err()

	return err
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
