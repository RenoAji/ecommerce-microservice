package worker

import (
	"context"
	"log"
	"order-service/internal/infrastructure"
	"order-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type DeliverySuccessWorker struct {
	brokerRedis *redis.Client
	service     *service.OrderService
}

func NewDeliverySuccessWorker(brokerRedis *redis.Client, service *service.OrderService) *DeliverySuccessWorker {
	return &DeliverySuccessWorker{brokerRedis: brokerRedis, service: service}
}

func (w *DeliverySuccessWorker) ListenForDeliverySuccess(ctx context.Context){
		// We start by trying to read pending messages (ID "0")
	// Once we run out of pending messages, we switch to new ones (ID ">")
	currentID := "0"
	for {
		select {
		case <-ctx.Done():
			return
		default:
			entries, err := w.brokerRedis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    "order-group",
				Consumer: "order-worker-1",
				Streams:  []string{"stream:delivery:delivered", currentID},
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
						Stream: "stream:delivery:delivered",
						Group:  "order-group",
						Start:  msg.ID,
						End:    msg.ID,
						Count:  1,
					}).Result()
					if len(pendingInfo) > 0 && pendingInfo[0].RetryCount >= 5 {
						// If delivered more than 5 times, move to DLQ
						log.Printf("Critical: Message %s failed 5 times. Moving to Dead Letter Queue.", msg.ID)
						w.moveToDLQ(ctx, msg)
						_, _ = w.brokerRedis.XAck(ctx, "stream:delivery:delivered", "order-group", msg.ID).Result()
						continue
					}

					orderIDStr, ok := msg.Values["order_id"].(string)
					if !ok {
						continue
					}
					// Process the delivery success event
					err := w.service.UpdateOrderStatus(ctx, orderIDStr, "SHIPPED")
					if err != nil {
						continue
					}
					// Acknowledge the message after processing
					_, err = w.brokerRedis.XAck(ctx, "stream:delivery:delivered", "order-group", msg.ID).Result()
					if err != nil {
						continue
					}

					log.Printf("Info: Order %s marked as SHIPPED.", orderIDStr)
				}
			}
		}
	}
}

func (w *DeliverySuccessWorker) moveToDLQ(ctx context.Context, msg redis.XMessage) {
    infrastructure.MoveToDLQ(ctx, w.brokerRedis, msg, "stream:delivery:delivered:dlq", "Exceeded max retries (5)")
}