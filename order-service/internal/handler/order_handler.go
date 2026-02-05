package handler

import (
	"log"
	"order-service/internal/domain"
	"order-service/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OrderHandler struct {
	orderService *service.OrderService
}

func NewOrderHandler(orderService *service.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// PostOrder godoc
// @Summary Create a new order
// @Description Create a new order from cart items. Can order all cart items or specific products by providing product IDs.
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param order body domain.CreateOrderRequest false "Order details with optional product IDs. Leave empty to checkout entire cart."
// @Success 201 {object} map[string]string "Order received"
// @Failure 400 {object} map[string]string "Invalid request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to create order"
// @Router /order [post]
func (h *OrderHandler) PostOrder(c *gin.Context) {
	ctx := c.Request.Context()
	var req domain.CreateOrderRequest
	
	// Bind JSON but don't fail if body is empty
	_ = c.ShouldBindJSON(&req)

	userID := c.GetUint("userID")

	// Call the service layer to create the order
	err := h.orderService.CreateOrder(&req, ctx, userID)

	

	if err != nil {
		log.Printf("ERROR creating order: %v", err)
		c.JSON(500, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(201, gin.H{"message": "Order received"})
}

// GetOrders godoc
// @Summary Get user's orders
// @Description Retrieve all orders for the authenticated user with optional status filter
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by order status" default(PENDING)
// @Success 200 {array} domain.Order
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Failed to retrieve orders"
// @Router /order [get]
func (h *OrderHandler) GetOrders(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.GetUint("userID")
	status := c.DefaultQuery("status","PENDING")

	orders, err := h.orderService.GetOrders(ctx, userID, status)
	if err != nil {
		log.Printf("ERROR retrieving orders: %v", err)
		c.JSON(500, gin.H{"error": "Failed to retrieve orders"})
		return
	}

	c.JSON(200, orders)
}

// GetOrderByID godoc
// @Summary Get order by ID
// @Description Retrieve a specific order by its ID for the authenticated user
// @Tags Orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} domain.Order
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Order not found"
// @Failure 500 {object} map[string]string "Failed to retrieve order"
// @Router /order/{id} [get]
func (h *OrderHandler) GetOrderByID(c *gin.Context) {
	ctx := c.Request.Context()
	orderID := c.Param("id")

	order, err := h.orderService.GetOrderByID(ctx, orderID)
	if err != nil {
		log.Println("Error retrieving order:", err)
		
		// Check if it's a record not found error
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Order not found"})
			return
		}
		
		c.JSON(500, gin.H{"error": "Failed to retrieve order"})
		return
	}

	c.JSON(200, order)
}