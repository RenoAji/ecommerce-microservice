package domain

type CreateOrderRequest struct {
	ProductIDs []uint `json:"product_ids" binding:"omitempty,min=1"`
}