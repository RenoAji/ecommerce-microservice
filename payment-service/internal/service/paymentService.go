package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"libs/logger"
	"payment-service/internal/config"
	"payment-service/internal/domain"
	"payment-service/internal/infrastructure"
	"payment-service/internal/repository"
	"strconv"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"go.uber.org/zap"
)

type PaymentService struct {
	repo           repository.PaymentRepository
	eventRepo      repository.EventRepository
	midtransClient *infrastructure.MidtransWrapper
}

func NewPaymentService(repo repository.PaymentRepository, eventRepo repository.EventRepository, midtransClient *infrastructure.MidtransWrapper) *PaymentService {
	return &PaymentService{repo: repo, eventRepo: eventRepo, midtransClient: midtransClient}
}

func (s *PaymentService) CreatePendingPayment(ctx context.Context, orderID uint, amount uint) (string, error) {
	l := logger.ForContext(ctx)
	// Initiate Snap request
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  fmt.Sprintf("%d", orderID),
			GrossAmt: int64(amount),
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
	}

	// Request create Snap transaction to Midtrans
	snapResp, midtransErr := s.midtransClient.SnapClient.CreateTransaction(req)
	if midtransErr != nil {
		return "", fmt.Errorf("failed to create midtrans transaction: %w", midtransErr)
	}

	// Save payment with status PENDING and payment URL
	err := s.repo.AddPayment(orderID, amount, snapResp.RedirectURL, "PENDING")
	if err != nil {
		return "", fmt.Errorf("failed to save pending payment: %w", err)
	}
	l.Info("Pending payment created", zap.Uint("orderID", orderID), zap.Uint("amount", amount))

	return snapResp.RedirectURL, nil
}

func (s *PaymentService) ProcessPaymentNotification(ctx context.Context, notificationPayload map[string]interface{}) error {
	l := logger.ForContext(ctx)
	// Verify the signature to ensure this request is actually from Midtrans
	cfg := config.LoadConfig()

	signatureRaw, ok := notificationPayload["signature_key"].(string)
	if !ok {
		return fmt.Errorf("missing signature_key")
	}

	orderId, ok := notificationPayload["order_id"].(string)
	if !ok {
		return fmt.Errorf("missing order_id")
	}

	statusCode, ok := notificationPayload["status_code"].(string)
	if !ok {
		return fmt.Errorf("missing status_code")
	}

	grossAmount, ok := notificationPayload["gross_amount"].(string)
	if !ok {
		return fmt.Errorf("missing gross_amount")
	}

	serverKey := cfg.MidtransServerKey

	// Signature Formula: SHA512(order_id + status_code + gross_amount + server_key)
	data := orderId + statusCode + grossAmount + serverKey
	hash := sha512.New()
	hash.Write([]byte(data))
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	if signatureRaw != expectedSignature {
		return fmt.Errorf("invalid signature")
	}

	transactionStatusResp, e := s.midtransClient.CoreClient.CheckTransaction(orderId)
	if e != nil {
		// Ignore test notifications that do not exist in Midtrans.
		if e.StatusCode == 404 {
			l.Info("Midtrans transaction not found, skipping notification", zap.String("orderID", orderId))
			return nil
		}
		return fmt.Errorf("failed to check transaction: %w", e)
	}

	orderIDUint, err := strconv.ParseUint(orderId, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order id in notification: %w", err)
	}

	if transactionStatusResp != nil {
		switch transactionStatusResp.TransactionStatus {
		case "capture":
			switch transactionStatusResp.FraudStatus {
			case "challenge":
				err := s.repo.UpdatePaymentStatus(uint(orderIDUint), "CHALLENGE")
				if err != nil {
					return fmt.Errorf("failed to mark payment as challenge: %w", err)
				}
				l.Info("Payment requires manual review", zap.String("orderID", orderId), zap.String("payment_status", "CHALLENGE"))
				return nil
			case "accept":
				return s.handlePaidPayment(ctx, orderId)
			case "deny":
				return s.handleFailedPayment(ctx, orderId)
			}
		case "settlement":
			return s.handlePaidPayment(ctx, orderId)
		case "pending":
			err := s.repo.UpdatePaymentStatus(uint(orderIDUint), "PENDING")
			if err != nil {
				return fmt.Errorf("failed to update payment status to pending: %w", err)
			}
			l.Info("Payment remains pending", zap.String("orderID", orderId), zap.String("payment_status", "PENDING"))
			return nil
		case "deny":
			return s.handleFailedPayment(ctx, orderId)
		case "cancel", "expire":
			return s.handleFailedPayment(ctx, orderId)
		}
	}
	return nil
}

func (s *PaymentService) handlePaidPayment(ctx context.Context, orderID string) error {
	l := logger.ForContext(ctx)
	orderIDUint, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order id for paid payment handling: %w", err)
	}

	// Send event to order service to update order status to PAID
	err = s.eventRepo.PublishPaymentEvent(ctx, &domain.PaymentEvent{
		OrderID:       uint(orderIDUint),
		Status:        "success",
		CorrelationID: correlationIDFromContext(ctx),
	})
	if err != nil {
		return fmt.Errorf("failed to publish payment success event: %w", err)
	}

	err = s.repo.UpdatePaymentStatus(uint(orderIDUint), "SUCCESS")
	if err != nil {
		return fmt.Errorf("failed to update payment status to success: %w", err)
	}

	l.Info("Payment marked as successful", zap.String("orderID", orderID), zap.String("payment_status", "SUCCESS"))
	return nil
}

func (s *PaymentService) handleFailedPayment(ctx context.Context, orderID string) error {
	l := logger.ForContext(ctx)
	orderIDUint, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid order id for failed payment handling: %w", err)
	}

	// Send event to order service to update order status to PAYMENT_FAILED
	err = s.eventRepo.PublishPaymentEvent(ctx, &domain.PaymentEvent{
		OrderID:       uint(orderIDUint),
		Status:        "failed",
		CorrelationID: correlationIDFromContext(ctx),
	})
	if err != nil {
		return fmt.Errorf("failed to publish payment failed event: %w", err)
	}

	err = s.repo.UpdatePaymentStatus(uint(orderIDUint), "FAILED")
	if err != nil {
		return fmt.Errorf("failed to update payment status to failed: %w", err)
	}

	l.Info("Payment marked as failed", zap.String("orderID", orderID), zap.String("payment_status", "FAILED"))
	return nil
}

// CleanupExpiredPayments finds and processes payments that are stuck in PENDING status
// This handles cases where webhooks were missed due to downtime or network issues
func (s *PaymentService) CleanupExpiredPayments(ctx context.Context) error {
	l := logger.ForContext(ctx)
	// Find all PENDING payments older than 2 minutes (1min expiry + 1min buffer)
	// TODO: Change to 35 minutes for production (30min expiry + 5min buffer)
	expiredPayments, err := s.repo.FindExpiredPendingPayments(60)
	if err != nil {
		return fmt.Errorf("failed to find expired pending payments: %w", err)
	}

	if len(expiredPayments) == 0 {
		l.Info("No expired pending payments found", zap.Int("expiryMinutes", 60))
		return nil
	}

	l.Info("Expired pending payments found", zap.Int("count", len(expiredPayments)))

	for _, payment := range expiredPayments {
		orderIDStr := fmt.Sprintf("%d", payment.OrderID)
		l.Info("Checking expired payment", zap.String("orderID", orderIDStr))

		// Double-check with Midtrans API to get current status
		transactionStatusResp, midtransErr := s.midtransClient.CoreClient.CheckTransaction(orderIDStr)

		if midtransErr != nil {
			// Handle 404 - transaction doesn't exist in Midtrans (payment page never opened or test order)
			if midtransErr.GetStatusCode() == 404 {
				err := s.handleFailedPayment(ctx, orderIDStr)
				if err != nil {
					return fmt.Errorf("failed to handle missing midtrans transaction %s as failed payment: %w", orderIDStr, err)
				}
				continue
			}
			return fmt.Errorf("failed to check transaction %s with midtrans: %w", orderIDStr, midtransErr)
		}

		if transactionStatusResp != nil {
			switch transactionStatusResp.TransactionStatus {
			case "expire", "cancel", "deny":
				err := s.handleFailedPayment(ctx, orderIDStr)
				if err != nil {
					return fmt.Errorf("failed to handle expired payment for order %s: %w", orderIDStr, err)
				}
			case "settlement", "capture":
				err := s.handlePaidPayment(ctx, orderIDStr)
				if err != nil {
					return fmt.Errorf("failed to handle successful payment for order %s: %w", orderIDStr, err)
				}
			case "pending":
				l.Info("Payment still pending", zap.String("orderID", orderIDStr))
			default:
				l.Info("Payment has unknown transaction status", zap.String("orderID", orderIDStr), zap.String("payment_status", transactionStatusResp.TransactionStatus))
			}
		}
	}

	l.Info("Expired payment cleanup completed", zap.Int("processedCount", len(expiredPayments)))
	return nil
}

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	return ""
}
