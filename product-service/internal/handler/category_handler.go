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

// Create godoc
// @Summary Create a new category
// @Description Create a new product category (Admin only)
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category body domain.Category true "Category data"
// @Success 201 {object} domain.CategorySuccessResponse "Category created successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid request body"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 403 {object} domain.ErrorResponse "Access denied: Admins only"
// @Failure 409 {object} domain.ErrorResponse "category already exists"
// @Failure 500 {object} domain.ErrorResponse "could not create category"
// @Router /categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var category domain.Category

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&category); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Call the service layer
	if err := h.productService.CreateCategory(&category); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, domain.ErrorResponse{Error: "category already exists"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not create category"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, domain.CategorySuccessResponse{Message: "Category created successfully", Category: category})
}
