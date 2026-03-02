package worker

import (
	"cart-service/internal/infrastructure"
	"cart-service/internal/service"
	"context"
	"log"

	"github.com/redis/go-redis/v9"
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
		orderID, ok := msg.Values["order_id"].(uint)
		if !ok {
			return nil
		}
		err := d.s.ClearCart(ctx, orderID)
		if err != nil {
			return err
		}
		log.Printf("Info: Cart cleared for Order ID %d due to payment failure.", orderID)
		return nil
	})
}