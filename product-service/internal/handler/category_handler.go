package handler

import (
	"net/http"
	"product-service/internal/domain"
	"product-service/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	productService *service.ProductService
}

func NewCategoryHandler(ps *service.ProductService) *CategoryHandler {
	return &CategoryHandler{productService: ps}
}

func (h *CategoryHandler) Create(c *gin.Context) {
	var category domain.Category

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Call the service layer
	if err := h.productService.CreateCategory(&category); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "category already exists"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create category"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, gin.H{"message": "Category created successfully", "category": category})
}
