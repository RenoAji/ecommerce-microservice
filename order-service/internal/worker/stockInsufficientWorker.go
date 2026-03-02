package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type StockInsufficientWorker struct {
	s *service.OrderService
	w *infrastructure.EventConsumerWorker
}

func NewStockInsufficientWorker(brokerRedis *redis.Client, service *service.OrderService) *StockInsufficientWorker {
	return &StockInsufficientWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:stock:insufficient", "stream:stock:insufficient:dlq", "order-group", "stock-insufficient-worker"),
	}
}

func (d *StockInsufficientWorker) ListenForStockInsufficient(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			return nil
		}
		err := d.s.UpdateOrderStatus(ctx, orderIDStr, "CANCELLED")
		if err != nil {
			log.Printf("Failed to cancel order %s due to insufficient stock: %v", orderIDStr, err)
			return err
		}
		log.Printf("Order %s cancelled due to insufficient stock", orderIDStr)
		return nil
	})
}
