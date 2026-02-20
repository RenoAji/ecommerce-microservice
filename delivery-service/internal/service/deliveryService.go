package service

import (
	"context"
	"delivery-service/internal/domain"
	"delivery-service/internal/repository"
	"fmt"
	"log"
	"strings"
)

type DeliveryService struct {
	repo repository.DeliveryRepository
	eventRepo repository.EventRepository
}

func NewDeliveryService(repo repository.DeliveryRepository, eventRepo repository.EventRepository) *DeliveryService {
	return &DeliveryService{repo: repo, eventRepo: eventRepo}
}

func (s *DeliveryService) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error {
	return s.repo.CreateDelivery(ctx, delivery)
}

func (s *DeliveryService) GetDeliveryByID(id uint) (*domain.Delivery, error) {
	return s.repo.GetDeliveryByID(id)
}

func (s *DeliveryService) GetDeliveryByOrderID(orderID uint) (*domain.Delivery, error) {
	return s.repo.GetDeliveryByOrderID(orderID)
}

func (s *DeliveryService) ListDeliveries(status string) ([]*domain.Delivery, error) {
	return s.repo.GetAllDeliveries(status)
}

func (s *DeliveryService) UpdateDeliveryStatus(ctx context.Context, id uint, status string) error {
	return s.repo.WithTransaction(ctx, func (txRepo repository.DeliveryRepository) error {
		if err := txRepo.UpdateDeliveryStatus(id, status); err != nil {
			return err
		}

		if status == "FAILED" || status == "DELIVERED" {
			delivery, err := txRepo.GetDeliveryByID(id)
			if err != nil {
				return err
			}
			eventName := strings.ToLower(status)
			err = s.repo.CreateOutboxMessage(ctx, eventName, delivery)
			if err != nil {
				return fmt.Errorf("failed to create outbox message: %w", err)
			}
		}
		return nil
	})
}

func (s *DeliveryService) PublishOutboxMessage(ctx context.Context, outbox *domain.DeliveryOutboxMessage) error {
	err := s.eventRepo.PublishEvent(ctx, "delivery", &domain.Delivery{
		ID:      outbox.DeliveryID,
		OrderID: outbox.OrderID,
		Status:  outbox.Status,
	})
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	log.Printf("Published event for delivery ID %d, status: %s", outbox.DeliveryID, outbox.Status)
	return s.repo.MarkOutboxMessageAsPublished(ctx, outbox.ID)
}

func (s *DeliveryService) GetPendingOutboxMessages(ctx context.Context) ([]*domain.DeliveryOutboxMessage, error) {
	return s.repo.GetPendingOutboxMessages(ctx)
}