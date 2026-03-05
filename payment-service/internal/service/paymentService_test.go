package service

import (
	"context"
	"testing"

	"payment-service/internal/domain"
)

type mockPaymentRepository struct {
	updatedOrderID uint
	updatedStatus  string
}

func (m *mockPaymentRepository) AddPayment(orderID uint, amount uint, paymentURL string, status string) error {
	return nil
}
func (m *mockPaymentRepository) GetPaymentByOrderID(orderID uint) (*domain.Payment, error) { return nil, nil }
func (m *mockPaymentRepository) UpdatePaymentStatus(orderID uint, status string) error {
	m.updatedOrderID = orderID
	m.updatedStatus = status
	return nil
}
func (m *mockPaymentRepository) FindExpiredPendingPayments(expiryMinutes int) ([]*domain.Payment, error) {
	return nil, nil
}

type mockPaymentEventRepository struct {
	publishedEvent *domain.PaymentEvent
}

func (m *mockPaymentEventRepository) PublishPaymentEvent(ctx context.Context, event *domain.PaymentEvent) error {
	m.publishedEvent = event
	return nil
}

func TestHandlePaidPaymentPublishesSuccessAndUpdatesStatus(t *testing.T) {
	repo := &mockPaymentRepository{}
	eventRepo := &mockPaymentEventRepository{}
	svc := NewPaymentService(repo, eventRepo, nil)

	err := svc.handlePaidPayment("42")
	if err != nil {
		t.Fatalf("handlePaidPayment() error = %v", err)
	}
	if repo.updatedOrderID != 42 || repo.updatedStatus != "SUCCESS" {
		t.Fatalf("expected update order 42 SUCCESS, got order=%d status=%s", repo.updatedOrderID, repo.updatedStatus)
	}
	if eventRepo.publishedEvent == nil || eventRepo.publishedEvent.Status != "success" || eventRepo.publishedEvent.OrderID != 42 {
		t.Fatalf("unexpected published event: %#v", eventRepo.publishedEvent)
	}
}

func TestHandleFailedPaymentPublishesFailedAndUpdatesStatus(t *testing.T) {
	repo := &mockPaymentRepository{}
	eventRepo := &mockPaymentEventRepository{}
	svc := NewPaymentService(repo, eventRepo, nil)

	err := svc.handleFailedPayment("9")
	if err != nil {
		t.Fatalf("handleFailedPayment() error = %v", err)
	}
	if repo.updatedOrderID != 9 || repo.updatedStatus != "FAILED" {
		t.Fatalf("expected update order 9 FAILED, got order=%d status=%s", repo.updatedOrderID, repo.updatedStatus)
	}
	if eventRepo.publishedEvent == nil || eventRepo.publishedEvent.Status != "failed" || eventRepo.publishedEvent.OrderID != 9 {
		t.Fatalf("unexpected published event: %#v", eventRepo.publishedEvent)
	}
}
