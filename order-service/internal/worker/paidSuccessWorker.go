package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
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
			return nil
		}
		err := d.s.UpdateOrderStatus(ctx, orderIDStr, "PAID")
		if err != nil {
			return err
		}
		log.Printf("Info: Order %s marked as PAID.", orderIDStr)
		return nil
	})
}