package domain

type Delivery struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	OrderID       uint   `gorm:"not null;index" json:"order_id"`
	Status        string `gorm:"type:varchar(50);not null;default:'RECEIVED'" json:"status" oneof:"RECEIVED,IN_TRANSIT,DELIVERED,FAILED"`
	CorrelationID string `gorm:"type:varchar(100);index" json:"correlation_id,omitempty"`
}

type DeliveryOutboxMessage struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	Status        string `gorm:"type:varchar(255);not null" json:"status" oneof:"success,failed"`
	DeliveryID    uint   `gorm:"not null;index" json:"delivery_id"`
	OrderID       uint   `gorm:"not null;index" json:"order_id"`
	CorrelationID string `gorm:"type:varchar(100);index" json:"correlation_id,omitempty"`
	Published     bool   `gorm:"not null;default:false" json:"published"`
}
