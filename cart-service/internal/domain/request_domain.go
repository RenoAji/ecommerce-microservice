package domain

type AddCartItemRequest struct {
	ProductID uint  `json:"product_id" binding:"required"`
	Quantity  uint     `json:"quantity" binding:"required,min=1"`
}

type UpdateCartItemRequest struct {
	Quantity uint `json:"quantity" binding:"required,min=1"`
}