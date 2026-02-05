package domain

import "time"

type Order struct {
    ID          uint        `gorm:"primaryKey" json:"id"`
    UserID      uint        `gorm:"index" json:"user_id"`
    TotalAmount uint      `json:"total_amount"`
    Status      string      `gorm:"default:RECEIVED" json:"status" oneof:"RECEIVED AWAITING_PAYMENT PAID SHIPPED CANCELLED"` 
    Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
    CreatedAt   time.Time   `json:"created_at"`
    PaymentURL  string      `gorm:"column:payment_url" json:"payment_url,omitempty"`
    PaymentExpires time.Time `gorm:"column:payment_expires" json:"payment_expires,omitempty"`
}

type OrderItem struct {
    ID        uint    `gorm:"primaryKey" json:"id"`
    OrderID   uint    `gorm:"index" json:"order_id"`
    ProductID uint  `json:"product_id"`
    Name      string  `json:"name"`     // Snapshot of name at time of order
    Quantity  uint     `json:"quantity"`
    Price     uint `json:"price"`    // Snapshot of price at time of order
}