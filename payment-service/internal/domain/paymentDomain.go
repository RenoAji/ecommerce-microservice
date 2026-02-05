package domain

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	OrderID       uint           `gorm:"not null;index" json:"order_id"`
	Amount        uint        `gorm:"not null" json:"amount"`
	PaymentUrl    string         `gorm:"type:varchar(500)" json:"payment_url"`
	SnapToken    string         `gorm:"type:varchar(255)" json:"snap_token"`
	Status        string         `gorm:"type:varchar(50);not null;default:'PENDING'" json:"status" oneof:"PENDING,COMPLETED,FAILED"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}