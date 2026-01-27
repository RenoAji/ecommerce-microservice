package domain

type CreateOrderRequest struct {
	ProductIDs []string `json:"product_ids" binding:"omitempty,min=1"`
}