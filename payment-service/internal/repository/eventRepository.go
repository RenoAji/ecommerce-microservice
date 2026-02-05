package repository

import (
	"context"
	"payment-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type EventRepository interface {
	PublishPaymentEvent(ctx context.Context, event *domain.PaymentEvent) error
}

type RedisRepository struct {
	redisClient *redis.Client
}

func NewRedisRepository(redisClient *redis.Client) *RedisRepository {
	return &RedisRepository{
		redisClient: redisClient,
	}
}

func (r *RedisRepository) PublishPaymentEvent(ctx context.Context, event *domain.PaymentEvent) error {
	msg := map[string]interface{}{
		"order_id":  event.OrderID,
	}

	err := r.redisClient.XAdd(
		ctx,
		&redis.XAddArgs{
			Stream: "stream:payment:" + event.Status,
			MaxLen: 1000,
			Approx: true,
			Values: msg,
		},
	).Err()

	return err
}