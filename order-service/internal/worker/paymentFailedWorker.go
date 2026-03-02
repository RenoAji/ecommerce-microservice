package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type PaymentFailedWorker struct {
	s *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewPaymentFailedWorker(brokerRedis *redis.Client, service *service.OrderService) *PaymentFailedWorker {
	return &PaymentFailedWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:payment:failed", "stream:payment:failed:dlq", "order-group", "payment-failed-worker"),
	}
}

func (d *PaymentFailedWorker) ListenForPaymentFailed(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			return nil
		}
		err := d.s.UpdateOrderStatus(ctx, orderIDStr, "CANCELLED")
		if err != nil {
			return err
		}
		log.Printf("Info: Order %s marked as CANCELLED due to payment failure.", orderIDStr)
		return nil
	})
}