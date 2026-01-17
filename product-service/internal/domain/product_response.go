// internal/domain/product_response.go
package domain

import "time"

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