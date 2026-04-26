package worker

import (
	"context"
	"libs/logger"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
			logger.Log.Warn("dropping invalid stock insufficient message: missing order_id",
				zap.Any("raw_values", msg.Values))
			return nil
		}
		return d.s.UpdateOrderStatus(ctx, orderIDStr, "CANCELLED")
	})
}
