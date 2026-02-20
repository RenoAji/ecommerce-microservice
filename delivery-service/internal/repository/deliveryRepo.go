package repository

import (
	"context"
	"delivery-service/internal/domain"

	"gorm.io/gorm"
)

type DeliveryRepository interface {
	CreateDelivery(ctx context.Context, delivery *domain.Delivery) error
	GetDeliveryByID(id uint) (*domain.Delivery, error)
	GetDeliveryByOrderID(orderID uint) (*domain.Delivery, error)
	GetAllDeliveries(status string) ([]*domain.Delivery, error)
	UpdateDeliveryStatus(id uint, status string) error
	WithTransaction(ctx context.Context, fn func(repo DeliveryRepository) error) error
	CreateOutboxMessage(ctx context.Context, eventType string, payload *domain.Delivery) error
	GetPendingOutboxMessages(ctx context.Context) ([]*domain.DeliveryOutboxMessage, error)
	MarkOutboxMessageAsPublished(ctx context.Context, id uint) error
}

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error {
	return r.db.WithContext(ctx).Create(delivery).Error
}

func (r *PostgresRepository) GetDeliveryByID(id uint) (*domain.Delivery, error) {
	var delivery domain.Delivery
	if err := r.db.First(&delivery, id).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *PostgresRepository) GetDeliveryByOrderID(orderID uint) (*domain.Delivery, error) {
	var delivery domain.Delivery
	if err := r.db.Where("order_id = ?", orderID).First(&delivery).Error; err != nil {
		return nil, err
	}
	return &delivery, nil
}

func (r *PostgresRepository) GetAllDeliveries(status string) ([]*domain.Delivery, error) {
	var deliveries []*domain.Delivery
	query := r.db
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&deliveries).Error; err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *PostgresRepository) UpdateDeliveryStatus(id uint, status string) error {
	return r.db.Model(&domain.Delivery{}).Where("id = ?", id).Update("status", status).Error
}

// Transactional wrapper
func (r *PostgresRepository) WithTransaction(ctx context.Context, fn func(repo DeliveryRepository) error) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	repoWithTx := &PostgresRepository{db: tx}

	if err := fn(repoWithTx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (r *PostgresRepository) CreateOutboxMessage(ctx context.Context, eventType string, payload *domain.Delivery) error {
	outbox := &domain.DeliveryOutboxMessage{
		Status:    eventType,
		DeliveryID: payload.ID,
		OrderID: payload.OrderID,
	}
	return r.db.WithContext(ctx).Create(outbox).Error
}

func (r *PostgresRepository) GetPendingOutboxMessages(ctx context.Context) ([]*domain.DeliveryOutboxMessage, error) {
	var messages []*domain.DeliveryOutboxMessage
	if err := r.db.WithContext(ctx).Where("published = ?", false).Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *PostgresRepository) MarkOutboxMessageAsPublished(ctx context.Context, id uint) error {
	return r.db.Model(&domain.DeliveryOutboxMessage{}).Where("id = ?", id).Update("published", true).Error
}