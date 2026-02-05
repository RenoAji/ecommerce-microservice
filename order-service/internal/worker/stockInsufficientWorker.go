package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type StockInsufficientWorker struct {
	brokerRedis *redis.Client
	service     *service.OrderService
}

func NewStockInsufficientWorker(brokerRedis *redis.Client, service *service.OrderService) *StockInsufficientWorker {
	return &StockInsufficientWorker{brokerRedis: brokerRedis, service: service}
}

func (w *StockInsufficientWorker) ListenForStockInsufficient(ctx context.Context) {
	const GROUP_NAME = "order-group"
	const STREAM_NAME = "stream:stock:insufficient"
	const CONSUMER_NAME = "order-worker-1"
	// We start by trying to read pending messages (ID "0")
	// Once we run out of pending messages, we switch to new ones (ID ">")
	currentID := "0"
	for {
		select {
		case <-ctx.Done():
			return
		default:
			entries, err := w.brokerRedis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    GROUP_NAME,
				Consumer: CONSUMER_NAME,
				Streams:  []string{STREAM_NAME, currentID},
				Count:    1,
				Block:    5000,
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
				continue
			}

			for _, stream := range entries {
				// If we asked for pending and got 0 results, switch to new messages
				if currentID == "0" && len(stream.Messages) == 0 {
					currentID = ">"
					continue
				}
				
				for _, msg := range stream.Messages {
					// check how many times this message has been delivered
					pendingInfo , _ := w.brokerRedis.XPendingExt(ctx, &redis.XPendingExtArgs{
						Stream: STREAM_NAME,
						Group:  GROUP_NAME,
						Start:  msg.ID,
						End:    msg.ID,
						Count:  1,
					}).Result()
					if len(pendingInfo) > 0 && pendingInfo[0].RetryCount >= 5 {
						// If delivered more than 5 times, move to DLQ
						log.Printf("Critical: Message %s failed 5 times. Moving to Dead Letter Queue.", msg.ID)
						w.moveToDLQ(ctx, msg)
						_, _ = w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID).Result()
						continue
					}

					orderIDStr, ok := msg.Values["order_id"].(string)
					if !ok {
						continue
					}
					// Process the stock insufficient event - cancel the order
					err := w.service.UpdateOrderStatus(ctx, orderIDStr, "CANCELLED")
					if err != nil {
						log.Printf("Failed to cancel order %s due to insufficient stock: %v", orderIDStr, err)
						continue
					}
					log.Printf("Order %s cancelled due to insufficient stock", orderIDStr)
					// Acknowledge the message after processing
					_, err = w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID).Result()
					if err != nil {
						continue
					}
				}
			}
		}
	}
}

func (w *StockInsufficientWorker) moveToDLQ(ctx context.Context, msg redis.XMessage) {
    infrastructure.MoveToDLQ(ctx, w.brokerRedis, msg, "stream:stock:insufficient:dlq", "Exceeded max retries (5)")
}
