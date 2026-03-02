package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type StockreservedWorker struct {
	s *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewStockreservedWorker(brokerRedis *redis.Client, service *service.OrderService) *StockreservedWorker {
	return &StockreservedWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:stock:reserved", "stream:stock:reserved:dlq", "order-group", "stock-reserved-worker"),
	}
}

func (d *StockreservedWorker) ListenForStockReserved(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			return nil
		}
		err := d.s.ProcessAwaitingPaymentOrders(ctx, orderIDStr)
		if err != nil {
			return err
		}
		log.Printf("Info: Order %s marked as AWAITING_PAYMENT after stock reservation.", orderIDStr)
		return nil
	})
}
