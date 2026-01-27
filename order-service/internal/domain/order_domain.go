package domain

import "time"

type Order struct {
    ID          uint        `gorm:"primaryKey" json:"id"`
    UserID      uint        `gorm:"index" json:"user_id"`
    TotalAmount int64     `json:"total_amount"`
    Status      string      `gorm:"default:PENDING" json:"status" oneof:"PENDING PAID SHIPPED CANCELLED"` 
    Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
    CreatedAt   time.Time   `json:"created_at"`
}

type OrderItem struct {
    ID        uint    `gorm:"primaryKey" json:"id"`
    OrderID   uint    `gorm:"index" json:"order_id"`
    ProductID string  `json:"product_id"`
    Name      string  `json:"name"`     // Snapshot of name at time of order
    Quantity  int     `json:"quantity"`
    Price     int64 `json:"price"`    // Snapshot of price at time of order
}