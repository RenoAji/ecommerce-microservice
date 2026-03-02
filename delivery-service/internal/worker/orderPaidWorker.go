package worker

import (
	"context"
	"delivery-service/internal/domain"
	"delivery-service/internal/infrastructure"
	"delivery-service/internal/service"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
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
			log.Printf("Warning: order_id not found in message %s", msg.ID)
			return nil
		}
		orderID, err := strconv.ParseUint(orderIDStr, 10, 64)
		if err != nil {
			log.Printf("Warning: invalid order_id format in message %s", msg.ID)
			return nil
		}

		delivery := domain.Delivery{
			OrderID: uint(orderID),
			Status:  "PENDING",
		}
		if err := d.s.CreateDelivery(ctx, &delivery); err != nil {
			return err
		}

		log.Printf("Info: Delivery created for Order ID %d after order paid.", orderID)
		return nil
	})
}