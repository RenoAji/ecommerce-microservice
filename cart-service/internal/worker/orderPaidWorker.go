package worker

import (
	"cart-service/internal/infrastructure"
	"cart-service/internal/service"
	"context"
	"encoding/json"
	"fmt"
	"libs/logger"
	"strconv"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type orderItemMessage struct {
	ProductID uint `json:"product_id"`
	Quantity  uint `json:"quantity"`
}

type OrderPaidWorker struct {
	s *service.CartService
	w *infrastructure.EventConsumerWorker
}

func NewOrderPaidWorker(brokerRedis *redis.Client, service *service.CartService) *OrderPaidWorker {
	return &OrderPaidWorker{
		s: service,
		w: infrastructure.NewEventConsumerWorker(brokerRedis, "stream:orders:paid", "stream:orders:paid:dlq", "cart-group", "order-paid-worker"),
	}
}

func (d *OrderPaidWorker) Listen(ctx context.Context) {
	d.w.ListenForEvents(ctx, func(ctx context.Context, msg redis.XMessage) error {
		userIDStr, ok := msg.Values["user_id"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid order paid message: missing user_id",
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}
		userID, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			logger.Log.Warn("dropping invalid order paid message: invalid user_id",
				zap.String("userID", userIDStr),
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}

		itemsStr, ok := msg.Values["items"].(string)
		if !ok {
			logger.Log.Warn("dropping invalid order paid message: missing items",
				zap.Any("raw_values", msg.Values),
			)
			return nil
		}
		var items []orderItemMessage
		if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
			return fmt.Errorf("failed to unmarshal order items: %w", err)
		}

		productIDs := make([]uint, 0, len(items))
		for _, item := range items {
			productIDs = append(productIDs, item.ProductID)
		}

		if err := d.s.RemoveCartItems(ctx, uint(userID), productIDs); err != nil {
			return err
		}
		return nil
	})
}
