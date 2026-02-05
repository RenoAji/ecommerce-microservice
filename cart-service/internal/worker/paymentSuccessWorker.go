package worker

import (
	"cart-service/internal/infrastructure"
	"cart-service/internal/service"
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type PaidSuccessWorker struct {
	brokerRedis *redis.Client
	service     *service.CartService
}

func NewPaidSuccessWorker(brokerRedis *redis.Client, service *service.CartService) *PaidSuccessWorker {
	return &PaidSuccessWorker{brokerRedis: brokerRedis, service: service}
}

func (w *PaidSuccessWorker) ListenForPaidSuccess(ctx context.Context){
	// We start by trying to read pending messages (ID "0")
	// Once we run out of pending messages, we switch to new ones (ID ">")
	const GROUP_NAME = "cart-group"
	const STREAM_NAME = "stream:payment:success"
	const CONSUMER_NAME = "cart-worker-1"

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

					orderID, ok := msg.Values["order_id"].(uint)
					if !ok {
						continue
					}
					// Process the payment success event
					err := w.service.ClearCart(ctx, orderID)
					if err != nil {
						continue
					}
					// Acknowledge the message after processing
					_, err = w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID).Result()
					if err != nil {
						continue
					}

					log.Printf("Info: Cart cleared for Order ID %d after payment success.", orderID)
				}
			}
		}
	}
}

func (w *PaidSuccessWorker) moveToDLQ(ctx context.Context, msg redis.XMessage) {
    infrastructure.MoveToDLQ(ctx, w.brokerRedis, msg, "stream:payment:success:dlq", "Exceeded max retries (5)")
}