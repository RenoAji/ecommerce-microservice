package repository

import (
	"payment-service/internal/domain"

	"gorm.io/gorm"
)

type PaymentRepository interface {
	AddPayment(orderID uint, amount uint, paymentURL string, status string) error
	GetPaymentByOrderID(orderID uint) (*domain.Payment, error)
	UpdatePaymentStatus(orderID uint, status string) error
	FindExpiredPendingPayments(expiryMinutes int) ([]*domain.Payment, error)
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) AddPayment(orderID uint, amount uint, paymentURL string, status string) error {
	payment := &domain.Payment{
		OrderID:    orderID,
		Amount:     amount,
		PaymentUrl: paymentURL,
		Status:     status,
	}
	return r.db.Create(payment).Error
}

func (r *PostgresRepository) GetPaymentByOrderID(orderID uint) (*domain.Payment, error) {
	var payment domain.Payment
	result := r.db.Where("order_id = ?", orderID).First(&payment)
	if result.Error != nil {
		return nil, result.Error
	}
	return &payment, nil
}

func (r *PostgresRepository) UpdatePaymentStatus(orderID uint, status string) error {
	return r.db.Model(&domain.Payment{}).Where("order_id = ?", orderID).Update("status", status).Error
}

func (r *PostgresRepository) FindExpiredPendingPayments(expiryMinutes int) ([]*domain.Payment, error) {
	var payments []*domain.Payment
	// Find PENDING payments created more than expiryMinutes ago
	result := r.db.Where("status = ? AND created_at < NOW() - INTERVAL '1 minute' * ?", "PENDING", expiryMinutes).Find(&payments)
	if result.Error != nil {
		return nil, result.Error
	}
	return payments, nil
}