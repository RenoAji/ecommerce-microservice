package worker

import (
	"context"
	"delivery-service/internal/service"
	"libs/logger"
	"time"

	"go.uber.org/zap"
)

type OutboxWorker struct {
	service *service.DeliveryService
}

func NewOutboxWorker(service *service.DeliveryService) *OutboxWorker {
	return &OutboxWorker{service: service}
}

func (w *OutboxWorker) ListenForOutboxMessages(ctx context.Context) {
	// Check every 5 seconds (adjust based on your needs)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	logger.Log.Info("Starting outbox worker")
	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Stopping outbox worker")
			return
		case <-ticker.C:
			// Fetch pending outbox messages and publish them
			outbox, err := w.service.GetPendingOutboxMessages(ctx)
			if err != nil {
				logger.Log.Error("failed to fetch pending outbox messages", zap.Error(err))
				continue
			}

			for _, message := range outbox {
				msgCtx := ctx
				if message.CorrelationID != "" {
					msgCtx = context.WithValue(msgCtx, "correlation_id", message.CorrelationID)
				}

				l := logger.ForContext(msgCtx)
				select {
				case <-ctx.Done():
					l.Info("stopping outbox worker")
					return
				default:
					err := w.service.PublishOutboxMessage(msgCtx, message)
					if err != nil {
						l.Error("failed to publish outbox message",
							zap.Uint("outboxID", message.ID),
							zap.String("eventType", "delivery"),
							zap.Error(err),
						)
					}
				}
			}
		}
	}
}
