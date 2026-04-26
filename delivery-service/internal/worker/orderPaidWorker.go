package worker

import (
	"context"
	"delivery-service/internal/domain"
	"delivery-service/internal/infrastructure"
	"delivery-service/internal/service"
	"libs/logger"
	"strconv"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OrderPaidWorker struct {
	s *service.DeliveryService
	w *infrastructure.EventConsumerWorker
}

func NewOrderPaidWorker(brokerRedis *redis.Client, service *service.DeliveryService) *OrderPaidWorker {
	return &OrderPaidWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:orders:paid", "stream:orders:paid:dlq", "delivery-group", "order-paid-worker"),
	}
}

func (d *OrderPaidWorker) Listen(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		orderIDStr, ok := msg.Values["order_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid order paid message: missing order_id",
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}
		orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
		if err != nil {
			logger.Log.Warn("dropping invalid order paid message: invalid order_id",
				zap.String("orderID", orderIDStr),
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}

		delivery := domain.Delivery{
			OrderID: uint(orderID),
			Status:  "PENDING",
		}
		if err := d.s.CreateDelivery(ctx, &delivery); err != nil {
			return err
		}
		return nil
	})
}
