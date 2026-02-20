package handler

import (
	"delivery-service/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type DeliveryHandler struct {
	service *service.DeliveryService
}

func NewDeliveryHandler(s *service.DeliveryService) *DeliveryHandler {
	return &DeliveryHandler{service: s}
}

// ListDeliveries godoc
// @Summary List all deliveries
// @Description Get all deliveries, optionally filtered by status
// @Tags Delivery
// @Accept json
// @Produce json
// @Param status query string false "Filter by delivery status (PENDING, IN_TRANSIT, DELIVERED, FAILED)"
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /delivery [get]
func (h *DeliveryHandler) ListDeliveries(c *gin.Context) {
	status:= c.Query("status")

	// call service layer to get deliveries
	deliveries, err := h.service.ListDeliveries(status)
	if err != nil {
		c.JSON(500, gin.H{"error": "could not retrieve deliveries"})
		return
	}

	c.JSON(200, deliveries)
}

// GetDelivery godoc
// @Summary Get delivery by ID
// @Description Retrieve a specific delivery by its ID
// @Tags Delivery
// @Accept json
// @Produce json
// @Param id path int true "Delivery ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string "Invalid delivery ID"
// @Failure 404 {object} map[string]string "Delivery not found"
// @Security BearerAuth
// @Router /delivery/{id} [get]
func (h *DeliveryHandler) GetDelivery(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid delivery ID"})
		return
	}

	/// call service layer to get delivery by ID
	delivery, err := h.service.GetDeliveryByID(uint(id)) 
	if err != nil {
		c.JSON(404, gin.H{"error": "delivery not found"})
		return
	}

	c.JSON(200, delivery)
}

// UpdateDeliveryStatus godoc
// @Summary Update delivery status
// @Description Update the status of a delivery (PENDING, IN_TRANSIT, DELIVERED, FAILED)
// @Tags Delivery
// @Accept json
// @Produce json
// @Param id path int true "Delivery ID"
// @Param request body map[string]string true "Status update request" SchemaExample({"status": "IN_TRANSIT"})
// @Success 200 {object} map[string]string "Delivery status updated successfully"
// @Failure 400 {object} map[string]string "Invalid delivery ID or request body"
// @Failure 500 {object} map[string]string "Could not update delivery status"
// @Security BearerAuth
// @Router /delivery/{id}/status [put]
func (h *DeliveryHandler) UpdateDeliveryStatus(c *gin.Context) {
	// parse delivery ID and status from request
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid delivery ID"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body"})
		return
	}

	// call service layer to update delivery status
	if err := h.service.UpdateDeliveryStatus(c.Request.Context(), uint(id), req.Status); err != nil {
		c.JSON(500, gin.H{"error": "could not update delivery status"})
		return
	}

	c.JSON(200, gin.H{"message": "delivery status updated successfully"})

}