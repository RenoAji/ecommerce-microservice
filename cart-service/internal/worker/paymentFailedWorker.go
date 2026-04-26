package worker

import (
	"cart-service/internal/infrastructure"
	"cart-service/internal/service"
	"context"
	"libs/logger"
	"strconv"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PaymentFailedWorker struct {
	s *service.CartService
	w *infrastructure.EventConsumerWorker
}

func NewPaymentFailedWorker(brokerRedis *redis.Client, service *service.CartService) *PaymentFailedWorker {
	return &PaymentFailedWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:payment:failed", "stream:payment:failed:dlq", "cart-group", "payment-failed-worker"),
	}
}

func (d *PaymentFailedWorker) ListenForPaymentFailed(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid payment failed message: missing order_id",
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}
		orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
		if err != nil {
			logger.Log.Warn("dropping invalid payment failed message: invalid order_id",
				zap.String("orderID", orderIDStr),
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}
		return d.s.ClearCart(ctx, uint(orderID))
	})
}
