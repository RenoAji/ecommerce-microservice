package infrastructure

import (
	"context"
	"libs/logger"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type EventConsumerWorker struct {
	brokerRedis   *redis.Client
	consumerGroup string
	consumerName  string
	streamName    string
	dlqStreamName string
}

func NewEventConsumerWorker(brokerRedis *redis.Client, streamName string, dlqStreamName string, consumerGroup string, consumerName string) *EventConsumerWorker {
	return &EventConsumerWorker{brokerRedis: brokerRedis, streamName: streamName, dlqStreamName: dlqStreamName, consumerGroup: consumerGroup, consumerName: consumerName}
}

func (w *EventConsumerWorker) ListenForEvents(ctx context.Context, handler func(ctx context.Context, msg redis.XMessage) error) {
	l := logger.ForContext(ctx)
	currentID := "0"
	for {
		select {
		case <-ctx.Done():
			l.Info("stopping event consumer worker",
				zap.String("stream", w.streamName),
				zap.String("consumerGroup", w.consumerGroup),
				zap.String("consumer", w.consumerName),
			)
			return
		default:
			entries, err := w.brokerRedis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    w.consumerGroup,
				Consumer: w.consumerName,
				Streams:  []string{w.streamName, currentID},
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
				l.Error("failed to read from redis stream",
					zap.String("stream", w.streamName),
					zap.String("consumerGroup", w.consumerGroup),
					zap.String("consumer", w.consumerName),
					zap.String("currentID", currentID),
					zap.Error(err),
				)
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
					pendingInfo, err := w.brokerRedis.XPendingExt(ctx, &redis.XPendingExtArgs{
						Stream: w.streamName,
						Group:  w.consumerGroup,
						Start:  msg.ID,
						End:    msg.ID,
						Count:  1,
					}).Result()
					if err != nil && err != redis.Nil {
						l.Error("failed to inspect pending message retry count",
							zap.String("stream", w.streamName),
							zap.String("consumerGroup", w.consumerGroup),
							zap.String("msgID", msg.ID),
							zap.Error(err),
						)
						continue
					}
					if len(pendingInfo) > 0 && pendingInfo[0].RetryCount >= 5 {
						// If delivered more than 5 times, move to DLQ
						l.Error("message exceeded max retries, moving to DLQ",
							zap.String("stream", w.streamName),
							zap.String("consumerGroup", w.consumerGroup),
							zap.String("msgID", msg.ID),
							zap.Int64("retryCount", pendingInfo[0].RetryCount),
							zap.String("reason", "Exceeded max retries (5)"),
						)
						if err := MoveToDLQ(ctx, w.brokerRedis, msg, w.dlqStreamName, "Exceeded max retries (5)"); err != nil {
							l.Error("failed to move message to DLQ",
								zap.String("stream", w.streamName),
								zap.String("dlqStream", w.dlqStreamName),
								zap.String("consumerGroup", w.consumerGroup),
								zap.String("msgID", msg.ID),
								zap.Error(err),
							)
							continue
						}
						if _, err := w.brokerRedis.XAck(ctx, w.streamName, w.consumerGroup, msg.ID).Result(); err != nil {
							l.Error("failed acknowledging message after DLQ move",
								zap.String("stream", w.streamName),
								zap.String("consumerGroup", w.consumerGroup),
								zap.String("msgID", msg.ID),
								zap.Error(err),
							)
						}
						continue
					}

					// Process the message with the provided handler
					msgCtx := withCorrelationIDFromMessage(ctx, msg)
					err = handler(msgCtx, msg)
					if err != nil {
						l.Error("handler failed to process message",
							zap.String("stream", w.streamName),
							zap.String("consumerGroup", w.consumerGroup),
							zap.String("msgID", msg.ID),
							zap.Error(err),
						)
						continue
					}

					// Acknowledge the message after processing
					_, err = w.brokerRedis.XAck(ctx, w.streamName, w.consumerGroup, msg.ID).Result()
					if err != nil {
						l.Error("failed to acknowledge processed message",
							zap.String("stream", w.streamName),
							zap.String("consumerGroup", w.consumerGroup),
							zap.String("msgID", msg.ID),
							zap.Error(err),
						)
						continue
					}
				}
			}
		}
	}
}

func withCorrelationIDFromMessage(ctx context.Context, msg redis.XMessage) context.Context {
	if correlationID, ok := msg.Values["correlation_id"].(string); ok && correlationID != "" {
		return context.WithValue(ctx, "correlation_id", correlationID)
	}

	return ctx
}
