package worker

import (
	"context"
	"libs/logger"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type DeliveryFailedWorker struct {
	s     *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewDeliveryFailedWorker(brokerRedis *redis.Client, service *service.OrderService) *DeliveryFailedWorker {
	return &DeliveryFailedWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:delivery:failed", "stream:delivery:failed:dlq", "order-group", "delivery-failed-worker"),
	}
}

func (d *DeliveryFailedWorker) ListenForDeliveryFailed(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid delivery failed message: missing order_id",
				zap.Any("raw_values", msg.Values))
			return nil
		}

		return d.s.UpdateOrderStatus(ctx, orderIDStr, "FAILED")
	})
}


