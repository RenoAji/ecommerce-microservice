package worker

import (
	"context"
	"libs/logger"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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
			logger.Log.Warn("dropping invalid stock reserved message: missing order_id",
				zap.Any("raw_values", msg.Values))
			return nil
		}
		return d.s.ProcessAwaitingPaymentOrders(ctx, orderIDStr)
	})
}
