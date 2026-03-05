package service

import (
	"context"
	"testing"

	"delivery-service/internal/domain"
	"delivery-service/internal/repository"
)

type mockDeliveryRepo struct {
	updatedID              uint
	updatedStatus          string
	getDeliveryByIDResp    *domain.Delivery
	createdOutboxEventType string
	createdOutboxPayload   *domain.Delivery
	markedOutboxID         uint
}

func (m *mockDeliveryRepo) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error { return nil }
func (m *mockDeliveryRepo) GetDeliveryByID(id uint) (*domain.Delivery, error) { return m.getDeliveryByIDResp, nil }
func (m *mockDeliveryRepo) GetDeliveryByOrderID(orderID uint) (*domain.Delivery, error) { return nil, nil }
func (m *mockDeliveryRepo) GetAllDeliveries(status string) ([]*domain.Delivery, error) { return nil, nil }
func (m *mockDeliveryRepo) UpdateDeliveryStatus(id uint, status string) error {
	m.updatedID = id
	m.updatedStatus = status
	return nil
}
func (m *mockDeliveryRepo) WithTransaction(ctx context.Context, fn func(repo repository.DeliveryRepository) error) error {
	return fn(m)
}
func (m *mockDeliveryRepo) CreateOutboxMessage(ctx context.Context, eventType string, payload *domain.Delivery) error {
	m.createdOutboxEventType = eventType
	m.createdOutboxPayload = payload
	return nil
}
func (m *mockDeliveryRepo) GetPendingOutboxMessages(ctx context.Context) ([]*domain.DeliveryOutboxMessage, error) {
	return nil, nil
}
func (m *mockDeliveryRepo) MarkOutboxMessageAsPublished(ctx context.Context, id uint) error {
	m.markedOutboxID = id
	return nil
}

type mockDeliveryEventRepo struct {
	lastEventType string
	lastDelivery  *domain.Delivery
}

func (m *mockDeliveryEventRepo) PublishEvent(ctx context.Context, eventType string, event *domain.Delivery) error {
	m.lastEventType = eventType
	m.lastDelivery = event
	return nil
}

func TestUpdateDeliveryStatusDeliveredCreatesOutbox(t *testing.T) {
	repo := &mockDeliveryRepo{getDeliveryByIDResp: &domain.Delivery{ID: 10, OrderID: 20, Status: "DELIVERED"}}
	svc := NewDeliveryService(repo, &mockDeliveryEventRepo{})

	err := svc.UpdateDeliveryStatus(context.Background(), 10, "DELIVERED")
	if err != nil {
		t.Fatalf("UpdateDeliveryStatus() error = %v", err)
	}
	if repo.updatedID != 10 || repo.updatedStatus != "DELIVERED" {
		t.Fatalf("expected status update to DELIVERED for ID 10, got ID=%d status=%s", repo.updatedID, repo.updatedStatus)
	}
	if repo.createdOutboxEventType != "delivered" || repo.createdOutboxPayload == nil {
		t.Fatalf("expected delivered outbox message, got event=%s payload=%#v", repo.createdOutboxEventType, repo.createdOutboxPayload)
	}
}

func TestUpdateDeliveryStatusInTransitDoesNotCreateOutbox(t *testing.T) {
	repo := &mockDeliveryRepo{getDeliveryByIDResp: &domain.Delivery{ID: 10, OrderID: 20, Status: "IN_TRANSIT"}}
	svc := NewDeliveryService(repo, &mockDeliveryEventRepo{})

	err := svc.UpdateDeliveryStatus(context.Background(), 10, "IN_TRANSIT")
	if err != nil {
		t.Fatalf("UpdateDeliveryStatus() error = %v", err)
	}
	if repo.createdOutboxPayload != nil {
		t.Fatal("did not expect outbox message for non-terminal status")
	}
}

func TestPublishOutboxMessagePublishesAndMarksAsPublished(t *testing.T) {
	repo := &mockDeliveryRepo{}
	eventRepo := &mockDeliveryEventRepo{}
	svc := NewDeliveryService(repo, eventRepo)

	outbox := &domain.DeliveryOutboxMessage{ID: 8, DeliveryID: 2, OrderID: 3, Status: "DELIVERED"}
	if err := svc.PublishOutboxMessage(context.Background(), outbox); err != nil {
		t.Fatalf("PublishOutboxMessage() error = %v", err)
	}
	if eventRepo.lastEventType != "delivery" || eventRepo.lastDelivery == nil {
		t.Fatalf("unexpected publish call: event=%s payload=%#v", eventRepo.lastEventType, eventRepo.lastDelivery)
	}
	if repo.markedOutboxID != 8 {
		t.Fatalf("expected outbox ID 8 marked published, got %d", repo.markedOutboxID)
	}
}
