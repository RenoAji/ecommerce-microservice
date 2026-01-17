package domain

import (
	"time"

	"gorm.io/gorm"
)

type Product struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string `gorm:"type:varchar(255);unique;not null" json:"name" binding:"required"`
	Description string `gorm:"type:text" json:"description"`
	Price       int64  `gorm:"type:bigint;not null" json:"price" binding:"required,gt=0"`
	Stock       int    `gorm:"not null" json:"stock" binding:"required,gte=0"`
	// Many-to-Many association
	Categories []Category     `gorm:"many2many:product_categories;" json:"categories"`
	CreatedAt  time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

type CreateProductRequest struct {
    Name        string `json:"name" binding:"required"`
    Description string `json:"description"`
    Price       int64  `json:"price" binding:"required,gt=0"`
    Stock       int    `json:"stock" binding:"required,gte=0"`
    CategoryIDs []uint `json:"category_ids" binding:"required"` // User only sends [1, 2, 3]
}

type UpdateProductRequest struct {
    Name        *string `json:"name"`
    Description *string `json:"description"`
    Price       *int64  `json:"price" binding:"omitempty,gt=0"`
    Stock       *int    `json:"stock" binding:"omitempty,gte=0"`
    CategoryIDs []uint  `json:"category_ids"` // If provided, we replace all categories
}

type Category struct {
	ID   uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name string `gorm:"type:varchar(100);uniqueIndex;not null" json:"name" binding:"required"`
	// Inverse association
	Products  []Product `gorm:"many2many:product_categories;" json:"products"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func ToProductResponse(p Product) ProductResponse {
    cats := make([]CategoryResponse, len(p.Categories))
    for i, c := range p.Categories {
        cats[i] = CategoryResponse{
            ID:   c.ID,
            Name: c.Name,
        }
    }

    return ProductResponse{
        ID:          p.ID,
        Name:        p.Name,
        Description: p.Description,
        Price:       p.Price,
        Stock:       p.Stock,
        Categories:  cats,
        UpdatedAt:   p.UpdatedAt,
    }
}
