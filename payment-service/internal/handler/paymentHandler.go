package handler

import (
	"payment-service/internal/service"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

// HandleWebhook godoc
// @Summary Handle Midtrans payment webhook
// @Description Receives payment notifications from Midtrans (success, failed, expired, pending)
// @Tags Payment
// @Accept json
// @Produce json
// @Param notification body map[string]interface{} true "Midtrans notification payload"
// @Success 200 {object} map[string]string "Webhook processed successfully"
// @Failure 400 {object} map[string]string "Invalid signature or payload"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /payment/webhook [post]
func (h *PaymentHandler) HandleWebhook(c *gin.Context) {
	var notificationPayload map[string]interface{}
	if err := c.BindJSON(&notificationPayload); err != nil {
		c.JSON(400, gin.H{"error": "Invalid payload"})
		return
	}

	if err := h.service.ProcessPaymentNotification(c.Request.Context(), notificationPayload); err != nil {
		c.JSON(500, gin.H{"error": "Failed to process notification"})
		return
	}

	c.JSON(200, gin.H{"status": "success"})
}