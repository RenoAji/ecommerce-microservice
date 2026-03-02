package worker

import (
	"context"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
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
				return nil
			}
			// Process the delivery failed event
			err := d.s.UpdateOrderStatus(ctx, orderIDStr, "FAILED")
			if err != nil {
				return nil
			}
			return nil
	})
}


