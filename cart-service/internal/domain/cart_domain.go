package domain

type Cart struct {
	UserID   string     `json:"user_id"`
	Items    []CartItem `json:"items"`
	TotalQty int        `json:"total_qty"`
	TotalAmt int64    `json:"total_amt"`
}

type CartItem struct {
	ProductID string  `json:"product_id"`
	Name	  string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     int64 `json:"price"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}