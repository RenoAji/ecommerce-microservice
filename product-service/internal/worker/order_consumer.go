package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"product-service/internal/infrastructure"
	"product-service/internal/service"

	"github.com/redis/go-redis/v9"
)

type OrderWorker struct {
	brokerRedis *redis.Client
	service     *service.ProductService
}

func NewOrderWorker(brokerRedis *redis.Client, service *service.ProductService) *OrderWorker {
	return &OrderWorker{brokerRedis: brokerRedis, service: service}
}

func (w *OrderWorker) ListenForOrders(ctx context.Context) {
    // We start by trying to read pending messages (ID "0")
    // Once we run out of pending messages, we switch to new ones (ID ">")
    const GROUP_NAME = "product-group"
    const STREAM_NAME = "stream:orders:created"
    const CONSUMER_NAME = "product-worker-1"


    currentID := "0" 

    for {
        select {
        case <-ctx.Done():
            log.Println("Gracefully stopping worker...")
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
                log.Printf("Error reading stream: %v", err)
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
                    pendingInfo, _ := w.brokerRedis.XPendingExt(ctx, &redis.XPendingExtArgs{
                        Stream: STREAM_NAME,
                        Group:  GROUP_NAME,
                        Start:  msg.ID,
                        End:    msg.ID,
                        Count:  1,
                    }).Result()

                    // If retried more than 5 times, move to Dead Letter Queue
                    if len(pendingInfo) > 0 && pendingInfo[0].RetryCount > 5 {
                        log.Printf("Critical: Message %s failed 5 times. Moving to Dead Letter Queue.", msg.ID)
                        w.moveToDLQ(ctx, msg)
                        w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                        continue
                    }

                    // Parse the items JSON string
                    itemsStr, ok := msg.Values["items"].(string)
                    if !ok || itemsStr == "" {
                        log.Printf("Invalid or missing items in message: %v", msg.Values)
                        w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                        continue
                    }

                    // Get order_id from message
                    orderIDStr, ok := msg.Values["order_id"].(string)
                    if !ok || orderIDStr == "" {
                        log.Printf("Invalid or missing order_id in message: %v", msg.Values)
                        w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                        continue
                    }

                    // Convert order_id string to uint
                    var orderID uint
                    if _, err := fmt.Sscanf(orderIDStr, "%d", &orderID); err != nil {
                        log.Printf("Error parsing order_id: %v", err)
                        w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                        continue
                    }

                    // Define the structure for order items
                    type OrderItem struct {
                        ProductID uint `json:"product_id"`
                        Quantity  int  `json:"quantity"`
                    }

                    var items []OrderItem
                    if err := json.Unmarshal([]byte(itemsStr), &items); err != nil {
                        log.Printf("Error unmarshaling items: %v", err)
                        w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                        continue
                    }

                    stockUpdates := make(map[uint]int)
                    for _, item := range items {
                        // For stock deduction, we use negative quantity
                        stockUpdates[item.ProductID] -= item.Quantity
                    }

                    err := w.service.ReserveStock(ctx, orderID, stockUpdates)
                    if err != nil {
                        log.Printf("Error processing order %s: %v", msg.ID, err)
                        continue // Do not ack the message, so it can be retried
                    }
                    
                    // If processing is successful, acknowledge the message
                    w.brokerRedis.XAck(ctx, STREAM_NAME, GROUP_NAME, msg.ID)
                    log.Printf("Successfully processed order %s", msg.ID)
                }
            }
        }
    }
}


func (w *OrderWorker) moveToDLQ(ctx context.Context, msg redis.XMessage) {
    infrastructure.MoveToDLQ(ctx, w.brokerRedis, msg, "stream:orders:created:dlq", "Exceeded max retries (5)")
}