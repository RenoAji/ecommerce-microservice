package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"libs/logger"
	"product-service/internal/infrastructure"
	"product-service/internal/service"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type PaymentFailedWorker struct {
	brokerRedis *redis.Client
	service     *service.ProductService
}

func NewPaymentFailedWorker(brokerRedis *redis.Client, service *service.ProductService) *PaymentFailedWorker {
	return &PaymentFailedWorker{brokerRedis: brokerRedis, service: service}
}

func (w *PaymentFailedWorker) ListenForPaymentFailures(ctx context.Context) {
	// We start by trying to read pending messages (ID "0")
	// Once we run out of pending messages, we switch to new ones (ID ">")
	const GROUP_NAME = "product-group"
	const STREAM_NAME = "stream:payment:failed"
	const CONSUMER_NAME = "product-worker-2"

	currentID := "0"

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Gracefully stopping payment failed worker")
			return
		default:
			entries, err := w.brokerRedis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    GROUP_NAME,
				Consumer: CONSUMER_NAME,
				Streams:  []string{STREAM_NAME, currentID},
				Count:    1,
				Block:    5 * time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					// If we were checking pending (0) and found none,
					// switch to reading new messages (>)
					if currentID == "0" {
						currentID = ">"
					}
					continue
				}
				logger.Log.Error("failed to read payment failed stream", zap.Error(err))
				continue
			}

			for _, stream := range entries {
				// If we asked for pending and got 0 results, switch to new messages
				if currentID == "0" && len(stream.Messages) == 0 {
					currentID = ">"
					continue
				}

				for _, msg := range stream.Messages {
					msgCtx := withCorrelationIDFromMessage(ctx, msg)
					// check how many times this message has been delivered
					pendingInfo, err := w.brokerRedis.XPendingExt(msgCtx, &redis.XPendingExtArgs{
						Stream: STREAM_NAME,
						Group:  GROUP_NAME,
						Start:  msg.ID,
						End:    msg.ID,
						Count:  1,
					}).Result()
					if err != nil && err != redis.Nil {
						logger.Log.Error("failed to inspect pending message retry count",
							zap.String("stream", STREAM_NAME),
							zap.String("consumerGroup", GROUP_NAME),
							zap.String("msgID", msg.ID),
							zap.Error(err),
						)
						continue
					}

					// If retried more than 5 times, move to Dead Letter Queue
					if len(pendingInfo) > 0 && pendingInfo[0].RetryCount >= 5 {
						logger.Log.Error("message exceeded max retries, moving to DLQ",
							zap.String("stream", STREAM_NAME),
							zap.String("consumerGroup", GROUP_NAME),
							zap.String("msgID", msg.ID),
							zap.Int64("retryCount", pendingInfo[0].RetryCount),
						)
						if err := w.moveToDLQ(msgCtx, msg); err != nil {
							logger.Log.Error("failed to move message to DLQ", zap.String("msgID", msg.ID), zap.Error(err))
							continue
						}
						if _, err := w.brokerRedis.XAck(msgCtx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
							logger.Log.Error("failed acknowledging message after DLQ move", zap.String("msgID", msg.ID), zap.Error(err))
						}
						continue
					}

					// Parse the items JSON string
					itemsStr, ok := msg.Values["items"].(string)
					if !ok || itemsStr == "" {
						logger.Log.Warn("dropping invalid payment failed message: missing items", zap.Any("raw_values", msg.Values))
						if _, err := w.brokerRedis.XAck(msgCtx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
							logger.Log.Error("failed to acknowledge invalid message", zap.String("msgID", msg.ID), zap.Error(err))
						}
						continue
					}

					// Get order_id from message
					orderIDStr, ok := msg.Values["order_id"].(string)
					if !ok || orderIDStr == "" {
						logger.Log.Warn("dropping invalid payment failed message: missing order_id", zap.Any("raw_values", msg.Values))
						if _, err := w.brokerRedis.XAck(msgCtx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
							logger.Log.Error("failed to acknowledge invalid message", zap.String("msgID", msg.ID), zap.Error(err))
						}
						continue
					}

					// Convert order_id string to uint
					var orderID uint
					if _, err := fmt.Sscanf(orderIDStr, "%d", &orderID); err != nil {
						logger.Log.Warn("dropping invalid payment failed message: invalid order_id",
							zap.String("orderID", orderIDStr),
							zap.Any("raw_values", msg.Values),
						)
						if _, err := w.brokerRedis.XAck(msgCtx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
							logger.Log.Error("failed to acknowledge invalid message", zap.String("msgID", msg.ID), zap.Error(err))
						}
						continue
					}

					// Define the structure for order items
					type OrderItem struct {
						ProductID uint `json:"product_id"`
						Quantity  int  `json:"quantity"`
					}

					var items []OrderItem
					if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
						logger.Log.Warn("dropping invalid payment failed message: malformed items",
							zap.String("msgID", msg.ID),
							zap.Error(err),
						)
						if _, err := w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
							logger.Log.Error("failed to acknowledge invalid message", zap.String("msgID", msg.ID), zap.Error(err))
						}
						continue
					}

					stockUpdates := make(map[uint]int)
					for _, item := range items {
						stockUpdates[item.ProductID] = item.Quantity
					}

					err = w.service.ReleaseStock(msgCtx, stockUpdates)
					if err != nil {
						logger.Log.Error("failed to process payment failed message", zap.String("msgID", msg.ID), zap.Error(err))
						continue // Do not ack the message, so it can be retried
					}

					// If processing is successful, acknowledge the message
					if _, err := w.brokerRedis.XAck(msgCtx, STREAM_NAME, GROUP_NAME, msg.ID).Result(); err != nil {
						logger.Log.Error("failed to acknowledge processed payment failed message", zap.String("msgID", msg.ID), zap.Error(err))
						continue
					}
					logger.Log.Info("Payment failed message processed successfully", zap.String("msgID", msg.ID))
				}
			}
		}
	}
}

func (w *PaymentFailedWorker) moveToDLQ(ctx context.Context, msg redis.XMessage) error {
	return infrastructure.MoveToDLQ(ctx, w.brokerRedis, msg, "stream:payment:failed:dlq", "Exceeded max retries (5)")
}
