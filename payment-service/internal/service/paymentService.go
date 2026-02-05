package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"payment-service/internal/config"
	"payment-service/internal/domain"
	"payment-service/internal/infrastructure"
	"payment-service/internal/repository"
	"strconv"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

type PaymentService struct {
	repo repository.PaymentRepository
	eventRepo repository.EventRepository
	midtransClient *infrastructure.MidtransWrapper
}

func NewPaymentService(repo repository.PaymentRepository, eventRepo repository.EventRepository, midtransClient *infrastructure.MidtransWrapper) *PaymentService {
	return &PaymentService{repo: repo, eventRepo: eventRepo, midtransClient: midtransClient}
}

func (s *PaymentService) CreatePendingPayment(orderID uint, amount uint) (string, error) {
	// Initiate Snap request
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:   fmt.Sprintf("%d", orderID),
			GrossAmt:  int64(amount),
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		Expiry: &snap.ExpiryDetails{
			StartTime: "",
			Unit:      "minutes", 
			Duration:  1,  
		},
	}

	// 3. Request create Snap transaction to Midtrans
	snapResp, err := s.midtransClient.SnapClient.CreateTransaction(req)
	if err != nil {
		return "", err
	}

	fmt.Println("Response :", snapResp)


	// Log the payment URL
	log.Printf("Payment URL for order %d: %s", orderID, snapResp.RedirectURL)

	// Save payment with status PENDING and payment URL
	repoErr := s.repo.AddPayment(orderID, amount, snapResp.RedirectURL, "PENDING")
	if repoErr != nil {
		return "", repoErr
	}

	return snapResp.RedirectURL, nil
}

func (s *PaymentService) ProcessPaymentNotification(ctx context.Context, notificationPayload map[string]interface{}) error {
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
		// If transaction not found (404), log and return nil (don't retry)
		if e.StatusCode == 404 {
			log.Printf("Transaction %s not found in Midtrans (test notification?): %v", orderId, e)
			return nil
		}
		return fmt.Errorf("failed to check transaction: %w", e)
	}

	orderIDUint, err := strconv.ParseUint(orderId, 10, 64)
	if err != nil {
		return err
	}

	if transactionStatusResp != nil {
		switch transactionStatusResp.TransactionStatus {
		case "capture":
			switch transactionStatusResp.FraudStatus {
			case "challenge":
				log.Printf("Payment for order %s requires manual review", orderId)
				return s.repo.UpdatePaymentStatus(uint(orderIDUint), "CHALLENGE")
			case "accept":
				log.Printf("Payment captured and accepted for order %s", orderId)
				return s.handlePaidPayment(orderId)
			case "deny":
				log.Printf("Payment for order %s denied due to fraud", orderId)
				return s.handleFailedPayment(orderId)
			}
		case "settlement":
			log.Printf("Payment settled for order %s", orderId)
			return s.handlePaidPayment(orderId)
		case "pending":
			log.Printf("Payment pending for order %s", orderId)
			return s.repo.UpdatePaymentStatus(uint(orderIDUint), "PENDING")
		case "deny":
			log.Printf("Payment denied for order %s", orderId)
			return s.handleFailedPayment(orderId)
		case "cancel", "expire":
			log.Printf("Payment canceled or expired for order %s", orderId)
			return s.handleFailedPayment(orderId)
		}
	}
	return nil
}

func (s *PaymentService) handlePaidPayment(orderID string) error{
	orderIDUint, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return err
	}

	// Send event to order service to update order status to PAID
	s.eventRepo.PublishPaymentEvent(context.Background(), &domain.PaymentEvent{
		OrderID: uint(orderIDUint),
		Status: "success",
	})

	return s.repo.UpdatePaymentStatus(uint(orderIDUint), "SUCCESS")
}

func (s *PaymentService) handleFailedPayment(orderID string) error{
	orderIDUint, err := strconv.ParseUint(orderID, 10, 64)
	if err != nil {
		return err
	}

	// Send event to order service to update order status to PAYMENT_FAILED
	s.eventRepo.PublishPaymentEvent(context.Background(), &domain.PaymentEvent{
		OrderID: uint(orderIDUint),
		Status: "failed",
	})

	return s.repo.UpdatePaymentStatus(uint(orderIDUint), "FAILED")
}

// CleanupExpiredPayments finds and processes payments that are stuck in PENDING status
// This handles cases where webhooks were missed due to downtime or network issues
func (s *PaymentService) CleanupExpiredPayments(ctx context.Context) {
	// Find all PENDING payments older than 2 minutes (1min expiry + 1min buffer)
	// TODO: Change to 35 minutes for production (30min expiry + 5min buffer)
	expiredPayments, err := s.repo.FindExpiredPendingPayments(2)
	if err != nil {
		log.Printf("Error finding expired payments: %v", err)
		return
	}

	if len(expiredPayments) == 0 {
		log.Printf("No expired pending payments found (checking payments older than 2 minutes)")
		return
	}

	log.Printf("Found %d expired pending payments to cleanup", len(expiredPayments))

	for _, payment := range expiredPayments {
		orderIDStr := fmt.Sprintf("%d", payment.OrderID)
		log.Printf("Checking expired payment for order %s", orderIDStr)
		
		// Double-check with Midtrans API to get current status
		transactionStatusResp, midtransErr := s.midtransClient.CoreClient.CheckTransaction(orderIDStr)
		
		if midtransErr != nil {
			// Handle 404 - transaction doesn't exist in Midtrans (payment page never opened or test order)
			if midtransErr.GetStatusCode() == 404 {
				log.Printf("Transaction %s not found in Midtrans (404), treating as expired/cancelled", orderIDStr)
				handleErr := s.handleFailedPayment(orderIDStr)
				if handleErr != nil {
					log.Printf("Error handling failed payment for order %s: %v", orderIDStr, handleErr)
				}
				continue
			}
			log.Printf("Error checking transaction %s with Midtrans: %v", orderIDStr, midtransErr)
			continue
		}
		
		if transactionStatusResp != nil {
			switch transactionStatusResp.TransactionStatus {
			case "expire", "cancel", "deny":
				log.Printf("Cleaning up expired/cancelled payment for order %s (status: %s)", orderIDStr, transactionStatusResp.TransactionStatus)
				err := s.handleFailedPayment(orderIDStr)
				if err != nil {
					log.Printf("Error handling failed payment for order %s: %v", orderIDStr, err)
				}
			case "settlement", "capture":
				log.Printf("Payment was actually successful for order %s, updating status", orderIDStr)
				err := s.handlePaidPayment(orderIDStr)
				if err != nil {
					log.Printf("Error handling paid payment for order %s: %v", orderIDStr, err)
				}
			case "pending":
				log.Printf("Payment for order %s is still pending in Midtrans, will check again later", orderIDStr)
			default:
				log.Printf("Unknown transaction status %s for order %s", transactionStatusResp.TransactionStatus, orderIDStr)
			}
		}
	}
}