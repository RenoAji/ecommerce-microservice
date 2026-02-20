package handler

import (
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

// Create godoc
// @Summary Create a new product
// @Description Create a new product (Admin only)
// @Tags Products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product body domain.CreateProductRequest true "Product data"
// @Success 201 {object} domain.SuccessResponse "Product created successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid request body / missing required fields / invalid category ID"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 403 {object} domain.ErrorResponse "Access denied: Admins only"
// @Failure 409 {object} domain.ErrorResponse "product already exists"
// @Failure 500 {object} domain.ErrorResponse "could not create product"
// @Router /products [post]
func (h *ProductHandler) Create(c *gin.Context) {
	var product domain.CreateProductRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Call the service layer
	if err := h.productService.CreateProduct(&product); err != nil {
		// Check PostgreSQL unique constraint violation return 409
		if strings.Contains(err.Error(), "duplicate key value") {
			c.JSON(http.StatusConflict, domain.ErrorResponse{Error: "product already exists"})
			return
		}

		if strings.Contains(err.Error(), "violates not-null constraint") {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "missing required fields"})
			return
		}

		if strings.Contains(err.Error(), "one or more category IDs do not exist") {
			c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "invalid category ID"})
			return
		}

		// Server error return 500
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not create product"})
		return
	}

	// Success response
	c.JSON(http.StatusCreated, domain.SuccessResponse{Message: "Product created successfully"})
}

// Get godoc
// @Summary Get paginated products
// @Description Get products with pagination and filters
// @Tags Products
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param search query string false "Search by product name"
// @Param category_id query int false "Filter by category ID"
// @Param min_price query number false "Minimum price"
// @Param max_price query number false "Maximum price"
// @Param sort_by query string false "Sort field" default(created_at)
// @Param order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} domain.PaginatedProducts
// @Failure 500 {object} domain.ErrorResponse "could not retrieve products"
// @Router /products [get]
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
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not retrieve products"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get product by ID
// @Description Get a single product by its ID
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} domain.ProductDataResponse "product"
// @Failure 400 {object} domain.ErrorResponse "invalid product ID"
// @Failure 404 {object} domain.ErrorResponse "product not found"
// @Failure 500 {object} domain.ErrorResponse "could not retrieve product"
// @Router /products/{id} [get]
func (h *ProductHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "invalid product ID"})
		return
	}

	product, err := h.productService.GetProductByID(uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not retrieve product"})
		return
	}

	c.JSON(http.StatusOK, domain.ProductDataResponse{Product: domain.ToProductResponse(*product)})
}

// Delete godoc
// @Summary Delete product
// @Description Delete a product by ID (Admin only)
// @Tags Products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} domain.SuccessResponse "Product deleted successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid product ID"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 403 {object} domain.ErrorResponse "Access denied: Admins only"
// @Failure 404 {object} domain.ErrorResponse "product not found"
// @Failure 500 {object} domain.ErrorResponse "could not delete product"
// @Router /products/{id} [delete]
func (h *ProductHandler) Delete(c *gin.Context) {
	// 1. Get ID from URL
	idParam := c.Param("id")

	// Parse the ID to uint
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	if err := h.productService.DeleteProduct(uint(productID)); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, domain.ErrorResponse{Error: "product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not delete product"})
		return
	}

	c.JSON(http.StatusOK, domain.SuccessResponse{Message: "Product deleted successfully"})
}

// Update godoc
// @Summary Update product
// @Description Update product details (Admin only)
// @Tags Products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param product body domain.UpdateProductRequest true "Product update data"
// @Success 200 {object} domain.ProductSuccessResponse "Product updated successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid request body"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 403 {object} domain.ErrorResponse "Access denied: Admins only"
// @Failure 500 {object} domain.ErrorResponse "could not update product"
// @Router /products/{id} [put]
func (h *ProductHandler) Update(c *gin.Context) {
	// 1. Get ID from URL
	idParam := c.Param("id")

	// Parse the ID to uint
	productID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid product ID"})
		return
	}

	var product domain.UpdateProductRequest

	// Bind JSON to struct
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Call the service layer to update the product
	updatedProduct, err := h.productService.UpdateProduct(uint(productID), &product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, domain.ErrorResponse{Error: "could not update product"})
		return
	}

	// Success response
	c.JSON(http.StatusOK, domain.ProductSuccessResponse{Message: "Product updated successfully", Product: domain.ToProductResponse(*updatedProduct)})
}