package worker

import (
	"context"
	"delivery-service/internal/service"
)

type OutboxWorker struct {
	service     *service.DeliveryService
}

func NewOutboxWorker(service *service.DeliveryService) *OutboxWorker {
	return &OutboxWorker{service: service}
}

func (w *OutboxWorker) ListenForOutboxMessages(ctx context.Context){
	for{
		outbox, err := w.service.GetPendingOutboxMessages(ctx)
		if err != nil {
			return
		}
	
		for _, message := range outbox {
			select {
			case <-ctx.Done():
				return
			default:
				// fmt.Printf("Processing outbox message ID: %d, Event: %s\n", message.ID, message.EventType)
				w.service.PublishOutboxMessage(ctx, message)
			}
		}
	}
}