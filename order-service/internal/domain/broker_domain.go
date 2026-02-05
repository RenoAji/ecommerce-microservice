package domain

type OrderCreatedEvent struct {
    OrderID     string             `json:"order_id"`
    UserID      string             `json:"user_id"`
    TotalAmount uint            `json:"total_amount"`
    Items       []OrderItemMessage `json:"items"`
}

type OrderItemMessage struct {
    ProductID uint `json:"product_id"`
    Quantity  uint    `json:"quantity"`
}

func ConvertToOrderItemMessages(items []OrderItem) []OrderItemMessage {
	msgs := make([]OrderItemMessage, len(items))
	for i, item := range items {
		msgs[i] = OrderItemMessage{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	return msgs
}