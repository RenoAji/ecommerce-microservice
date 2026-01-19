// internal/domain/product_response.go
package domain

import (
	"time"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// SuccessResponse represents a success message
type SuccessResponse struct {
	Message string `json:"message"`
}

// ProductSuccessResponse represents a success response with product data
type ProductSuccessResponse struct {
	Message string          `json:"message"`
	Product ProductResponse `json:"product"`
}

// CategorySuccessResponse represents a success response with category data
type CategorySuccessResponse struct {
	Message  string   `json:"message"`
	Category Category `json:"category"`
}

// ProductDataResponse represents a single product response
type ProductDataResponse struct {
	Product ProductResponse `json:"product"`
}

type CategoryResponse struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type ProductResponse struct {
	ID          uint               `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Price       int64              `json:"price"`
	Stock       int                `json:"stock"`
	Categories  []CategoryResponse `json:"categories"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

type PaginatedProducts struct {
	Products    []ProductResponse `json:"products"`
	Total       int64                    `json:"total"`
	Page        int                      `json:"page"`
	Limit       int                      `json:"limit"`
	TotalPages  int                      `json:"total_pages"`
}