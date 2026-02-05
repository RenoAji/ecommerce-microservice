package domain

type Cart struct {
	UserID   string     `json:"user_id"`
	Items    []CartItem `json:"items"`
	TotalQty uint        `json:"total_qty"`
	TotalAmt uint        `json:"total_amt"`
}

type CartItem struct {
	ProductID uint  `json:"product_id"`
	Name	  string  `json:"name"`
	Quantity  uint    `json:"quantity"`
	Price     uint    `json:"price"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}