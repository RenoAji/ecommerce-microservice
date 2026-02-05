package domain

type PaymentEvent struct {
	OrderID uint `json:"order_id"`
	Status string `json:"status" oneof:"success,failed"`
}