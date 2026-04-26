package domain

type StockEvent struct {
	OrderID       uint   `json:"order_id"`
	CorrelationID string `json:"correlation_id,omitempty"`
}
