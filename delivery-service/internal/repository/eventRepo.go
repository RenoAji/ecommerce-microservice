package repository

import (
	"context"

	"delivery-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type EventRepository interface {
	PublishEvent(ctx context.Context, eventType string, event *domain.Delivery) error
}

type RedisStreamRepository struct {
	redis *redis.Client
}

func NewRedisStreamRepository(redisClient *redis.Client) *RedisStreamRepository {
	return &RedisStreamRepository{redis: redisClient}
}

func (r *RedisStreamRepository) PublishEvent(ctx context.Context, eventType string, event *domain.Delivery) error {
	correlationID := event.CorrelationID
	if correlationID == "" {
		correlationID = correlationIDFromContext(ctx)
	}

	streamName := "stream:" + eventType + ":" + event.Status
	_, err := r.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: streamName,
		Values: map[string]interface{}{
			"delivery_id":    event.ID,
			"order_id":       event.OrderID,
			"correlation_id": correlationID,
		},
	}).Result()

	if err != nil {
		return err
	}
	return nil
}
