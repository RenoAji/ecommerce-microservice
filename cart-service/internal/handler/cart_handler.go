package handler

import (
	"cart-service/internal/domain"
	"cart-service/internal/service"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CartHandler struct {
	cartService *service.CartService
}

func NewCartHandler(cs *service.CartService) *CartHandler {
	return &CartHandler{cartService: cs}
}

// GetCart godoc
// @Summary Get user's cart
// @Description Retrieve the current user's shopping cart with all items
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.Cart
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 500 {object} domain.ErrorResponse "Internal server error"
// @Router /cart [get]
func (h *CartHandler) GetCart(c *gin.Context) {
	ctx := c.Request.Context()
	userIDInterface, _ := c.Get("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(500, domain.ErrorResponse{Error: "Internal server error"})
		return
	}

	cart, err := h.cartService.GetCart(ctx, userID)
	if err != nil {
		c.JSON(500, domain.ErrorResponse{Error: "Failed to retrieve cart"})
		return
	}

	c.JSON(200, cart)
}

// AddToCart godoc
// @Summary Add item to cart
// @Description Add a product item to the user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param item body domain.AddCartItemRequest true "Cart item to add"
// @Success 200 {object} domain.SuccessResponse "Item added successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid request body"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 500 {object} domain.ErrorResponse "Failed to add item"
// @Router /cart/item [post]
func (h *CartHandler) AddToCart(c *gin.Context) {
	ctx:= c.Request.Context()
	userIDInterface, _ := c.Get("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(500, domain.ErrorResponse{Error: "Internal server error"})
		return
	}

	var addItemRequest domain.AddCartItemRequest
	if err := c.ShouldBindJSON(&addItemRequest); err != nil {
		c.JSON(400, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	err := h.cartService.AddToCart(ctx, userID, &addItemRequest)
	if err != nil {
		fmt.Print(err)
		c.JSON(500, domain.ErrorResponse{Error: "Failed to add item to cart"})
		return
	}

	c.JSON(200, domain.SuccessResponse{Message: "Item added to cart successfully"})
}

// RemoveFromCart godoc
// @Summary Remove item from cart
// @Description Remove a specific product from the user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product_id path string true "Product ID"
// @Success 200 {object} domain.SuccessResponse "Item removed successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid product ID format"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 500 {object} domain.ErrorResponse "Failed to remove item"
// @Router /cart/item/{product_id} [delete]
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	ctx:= c.Request.Context()

	// Get user ID from context
	userIDInterface, _ := c.Get("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(500, domain.ErrorResponse{Error: "Internal server error"})
		return
	}

	// Get product ID from URL parameter
	productID := c.Param("product_id")
	if _, err := strconv.Atoi(productID); err != nil {
		c.JSON(400, domain.ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	// Call service to remove cart item
	err := h.cartService.RemoveCartItem(ctx, userID, productID)
	if err != nil {
		c.JSON(500, domain.ErrorResponse{Error: "Failed to remove cart item"})
		return
	}

	c.JSON(200, domain.SuccessResponse{Message: "Cart item removed successfully"})
}

// ClearCart godoc
// @Summary Clear cart
// @Description Remove all items from the user's cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} domain.SuccessResponse "Cart cleared successfully"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 500 {object} domain.ErrorResponse "Failed to clear cart"
// @Router /cart [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	ctx:= c.Request.Context()
	
	// Get user ID from context
	userIDInterface, _ := c.Get("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(500, domain.ErrorResponse{Error: "Internal server error"})
		return
	}

	// Call service to clear cart
	err := h.cartService.ClearCart(ctx, userID)
	if err != nil {
		c.JSON(500, domain.ErrorResponse{Error: "Failed to clear cart"})
		return
	}

	c.JSON(200, domain.SuccessResponse{Message: "Cart cleared successfully"})
}

// UpdateCartItem godoc
// @Summary Update cart item quantity
// @Description Update the quantity of a specific product in the cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param product_id path string true "Product ID"
// @Param item body domain.UpdateCartItemRequest true "Updated quantity"
// @Success 200 {object} domain.SuccessResponse "Item updated successfully"
// @Failure 400 {object} domain.ErrorResponse "Invalid request"
// @Failure 401 {object} domain.ErrorResponse "Unauthorized"
// @Failure 500 {object} domain.ErrorResponse "Failed to update item"
// @Router /cart/item/{product_id} [put]
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	ctx:= c.Request.Context()

	// Get user ID from context
	userIDInterface, _ := c.Get("userID")
	userID, ok := userIDInterface.(uint)
	if !ok {
		c.JSON(500, domain.ErrorResponse{Error: "Internal server error"})
		return
	}

	// Get product ID from URL parameter
	productID := c.Param("product_id")
	if _, err := strconv.Atoi(productID); err != nil {
		c.JSON(400, domain.ErrorResponse{Error: "Invalid product ID format"})
		return
	}

	// Bind request body
	var updateRequest domain.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(400, domain.ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Call service to update cart item
	err := h.cartService.UpdateCartItem(ctx, userID, productID, updateRequest.Quantity)
	if err != nil {
		c.JSON(500, domain.ErrorResponse{Error: "Failed to update cart item"})
		return
	}

	c.JSON(200, domain.SuccessResponse{Message: "Cart item updated successfully"})
}