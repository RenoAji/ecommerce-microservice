package service

import (
	"context"
	"delivery-service/internal/domain"
	"delivery-service/internal/repository"
	"fmt"
	"libs/logger"
	"strings"

	"go.uber.org/zap"
)

type DeliveryService struct {
	repo      repository.DeliveryRepository
	eventRepo repository.EventRepository
}

func NewDeliveryService(repo repository.DeliveryRepository, eventRepo repository.EventRepository) *DeliveryService {
	return &DeliveryService{repo: repo, eventRepo: eventRepo}
}

func (s *DeliveryService) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error {
	l := logger.ForContext(ctx)
	err := s.repo.CreateDelivery(ctx, delivery)
	if err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}
	l.Info("Delivery created successfully", zap.Uint("orderID", delivery.OrderID), zap.String("delivery_status", delivery.Status))
	return nil
}

func (s *DeliveryService) GetDeliveryByID(ctx context.Context, id uint) (*domain.Delivery, error) {
	l := logger.ForContext(ctx)
	delivery, err := s.repo.GetDeliveryByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery by id: %w", err)
	}
	l.Info("Delivery retrieved successfully", zap.Uint("deliveryID", id))
	return delivery, nil
}

func (s *DeliveryService) GetDeliveryByOrderID(ctx context.Context, orderID uint) (*domain.Delivery, error) {
	l := logger.ForContext(ctx)
	delivery, err := s.repo.GetDeliveryByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery by order id: %w", err)
	}
	l.Info("Delivery retrieved by order id", zap.Uint("orderID", orderID))
	return delivery, nil
}

func (s *DeliveryService) ListDeliveries(ctx context.Context, status string) ([]*domain.Delivery, error) {
	l := logger.ForContext(ctx)
	deliveries, err := s.repo.GetAllDeliveries(status)
	if err != nil {
		return nil, fmt.Errorf("failed to list deliveries: %w", err)
	}
	l.Info("Deliveries listed successfully", zap.String("delivery_status", status), zap.Int("count", len(deliveries)))
	return deliveries, nil
}

func (s *DeliveryService) UpdateDeliveryStatus(ctx context.Context, id uint, status string) error {
	l := logger.ForContext(ctx)
	err := s.repo.WithTransaction(ctx, func(txRepo repository.DeliveryRepository) error {
		if err := txRepo.UpdateDeliveryStatus(id, status); err != nil {
			return fmt.Errorf("failed to update delivery status: %w", err)
		}

		if status == "FAILED" || status == "DELIVERED" {
			delivery, err := txRepo.GetDeliveryByID(id)
			if err != nil {
				return fmt.Errorf("failed to fetch delivery for outbox: %w", err)
			}
			delivery.CorrelationID = correlationIDFromContext(ctx)
			eventName := strings.ToLower(status)
			err = s.repo.CreateOutboxMessage(ctx, eventName, delivery)
			if err != nil {
				return fmt.Errorf("failed to create outbox message: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update delivery status in transaction: %w", err)
	}
	l.Info("Delivery status updated successfully", zap.Uint("deliveryID", id), zap.String("delivery_status", status))
	return nil
}

func (s *DeliveryService) PublishOutboxMessage(ctx context.Context, outbox *domain.DeliveryOutboxMessage) error {
	msgCtx := ctx
	if outbox.CorrelationID != "" {
		msgCtx = context.WithValue(msgCtx, "correlation_id", outbox.CorrelationID)
	}

	l := logger.ForContext(msgCtx)
	err := s.eventRepo.PublishEvent(msgCtx, "delivery", &domain.Delivery{
		ID:            outbox.DeliveryID,
		OrderID:       outbox.OrderID,
		Status:        outbox.Status,
		CorrelationID: outbox.CorrelationID,
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	err = s.repo.MarkOutboxMessageAsPublished(msgCtx, outbox.ID)
	if err != nil {
		return fmt.Errorf("failed to mark outbox message as published: %w", err)
	}
	l.Info("Delivery outbox message published successfully", zap.Uint("deliveryID", outbox.DeliveryID), zap.String("delivery_status", outbox.Status), zap.Uint("outboxID", outbox.ID))
	return nil
}

func (s *DeliveryService) GetPendingOutboxMessages(ctx context.Context) ([]*domain.DeliveryOutboxMessage, error) {
	l := logger.ForContext(ctx)
	outbox, err := s.repo.GetPendingOutboxMessages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox messages: %w", err)
	}

	count := len(outbox)
	if count > 0 {
		l.Info("Pending outbox messages found", zap.Int("count", count))
	}
	return outbox, nil
}

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	return ""
}
