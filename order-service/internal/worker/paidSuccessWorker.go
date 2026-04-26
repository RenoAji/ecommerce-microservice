package worker

import (
	"context"
	"libs/logger"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PaidSuccessWorker struct {
	s *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewPaidSuccessWorker(brokerRedis *redis.Client, service *service.OrderService) *PaidSuccessWorker {
	return &PaidSuccessWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:payment:success", "stream:payment:success:dlq", "order-group", "paid-success-worker"),
	}
}

func (d *PaidSuccessWorker) ListenForPaidSuccess(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid payment success message: missing order_id",
				zap.Any("raw_values", msg.Values))
			return nil
		}
		return d.s.UpdateOrderToPaid(ctx, orderIDStr)
	})
}