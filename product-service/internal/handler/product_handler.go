package handler

import (
	"fmt"
	"net/http"
	"product-service/internal/domain"
	"product-service/internal/service"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductHandler struct {
	productService *service.ProductService
}

func NewProductHandler(ps *service.ProductService) *ProductHandler {
	return &ProductHandler{productService: ps}
}

func (h *ProductHandler) Create(c *gin.Context) {
	var product domain.CreateProductRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Call the service layer
	if err := h.productService.CreateProduct(&product); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, gin.H{"error": "product already exists"})
			return
		}

		if strings.Contains(err.Error(), "violates not-null constraint") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
			return
		}

		if strings.Contains(err.Error(), "one or more category IDs do not exist") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category ID"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create product"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, gin.H{"message": "Product created successfully", "product": product})
}

func (h *ProductHandler) Get(c *gin.Context) {
	search := c.Query("search")
	categoryID := c.Query("category_id")
	minPrice := c.Query("min_price")
	maxPrice := c.Query("max_price")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	order := c.DefaultQuery("order", "desc")

	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.productService.GetProducts(search, categoryID, minPrice, maxPrice, order, sortBy, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve products"})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *ProductHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	var productID uint

	// Parse ID parameter
	_, err := fmt.Sscanf(idParam, "%d", &productID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	product, err := h.productService.GetProductByID(productID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not retrieve product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": domain.ToProductResponse(*product)})
}

func (h *ProductHandler) AddStock(c *gin.Context) {
	// 1. Get ID from URL
	idParam := c.Param("id")

	// Parse the ID to uint
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	// Bind JSON body
	var req struct {
		Add int `json:"add" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.productService.AddStock(uint(productID), req.Add); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Stock updated successfully"})
}

func (h *ProductHandler) Delete(c *gin.Context) {
	// 1. Get ID from URL
	idParam := c.Param("id")

	// Parse the ID to uint
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	if err := h.productService.DeleteProduct(uint(productID)); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}

func (h *ProductHandler) Update(c *gin.Context) {
	// 1. Get ID from URL
	idParam := c.Param("id")

	// Parse the ID to uint
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	var product domain.UpdateProductRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Call the service layer to update the product
	updatedProduct, err := h.productService.UpdateProduct(uint(productID), &product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update product"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully", "product": domain.ToProductResponse(*updatedProduct)})
}