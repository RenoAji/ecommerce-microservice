package worker

import (
	"context"
	"libs/logger"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type DeliverySuccessWorker struct {
	s *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewDeliverySuccessWorker(brokerRedis *redis.Client, service *service.OrderService) *DeliverySuccessWorker {
	return &DeliverySuccessWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:delivery:delivered", "stream:delivery:delivered:dlq", "order-group", "delivery-success-worker"),
	}
}

func (d *DeliverySuccessWorker) ListenForDeliverySuccess(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid delivery message: missing order_id", 
                zap.Any("raw_values", msg.Values))
			return nil
		}
		return d.s.UpdateOrderStatus(ctx, orderIDStr, "SHIPPED")
	})
}