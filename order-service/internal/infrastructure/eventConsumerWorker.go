package infrastructure

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

type EventConsumerWorker struct {
	brokerRedis  *redis.Client
	consumerGroup string
	consumerName  string
	streamName	string
	dlqStreamName string
}

func NewEventConsumerWorker(brokerRedis *redis.Client, streamName string, dlqStreamName string, consumerGroup string, consumerName string) *EventConsumerWorker {
	return &EventConsumerWorker{brokerRedis: brokerRedis, streamName: streamName, dlqStreamName: dlqStreamName, consumerGroup: consumerGroup, consumerName: consumerName}
}

func (w *EventConsumerWorker) ListenForEvents(ctx context.Context, handler func(ctx context.Context, msg redis.XMessage) error){
	currentID := "0"
	for {
		select {
		case <-ctx.Done():
			return
		default:
			entries, err := w.brokerRedis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    w.consumerGroup,
				Consumer: w.consumerName,
				Streams:  []string{w.streamName, currentID},
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
						Stream: w.streamName,
						Group:  w.consumerGroup,
						Start:  msg.ID,
						End:    msg.ID,
						Count:  1,
					}).Result()
					if len(pendingInfo) > 0 && pendingInfo[0].RetryCount >= 5 {
						// If delivered more than 5 times, move to DLQ
						log.Printf("Critical: Message %s failed 5 times. Moving to Dead Letter Queue.", msg.ID)
						MoveToDLQ(ctx, w.brokerRedis, msg, w.dlqStreamName, "Exceeded max retries (5)")
						_, _ = w.brokerRedis.XAck(ctx, w.streamName, w.consumerGroup, msg.ID).Result()
						continue
					}

					// Process the message with the provided handler
					err = handler(ctx, msg)
					if err != nil {
						log.Printf("Error processing message %s: %v", msg.ID, err)
						continue
					}

					// Acknowledge the message after processing
					_, err = w.brokerRedis.XAck(ctx, w.streamName, w.consumerGroup, msg.ID).Result()
					if err != nil {
						continue
					}
				}
			}
		}
	}
}