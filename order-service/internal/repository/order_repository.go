package repository

import (
	"context"
	"order-service/internal/domain"

	"gorm.io/gorm"
)

type OrderRepository interface {
	AddOrder(ctx context.Context, order *domain.Order) error
	GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error)
	GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, status string) error
	UpdatePaymentUrl(ctx context.Context, orderID string, paymentUrl string) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) AddOrder(ctx context.Context, order *domain.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *PostgresRepository) GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error) {
	var orders []domain.Order
	query := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Preload("Items").Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PostgresRepository) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	var order domain.Order
	if err := r.db.WithContext(ctx).Where("id = ?", orderID).Preload("Items").First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PostgresRepository) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	return r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", orderID).Update("status", status).Error
}

func (r *PostgresRepository) UpdatePaymentUrl(ctx context.Context, orderID string, paymentUrl string) error { 
	return r.db.WithContext(ctx).Model(&domain.Order{}).Where("id = ?", orderID).Update("payment_url", paymentUrl).Error
}