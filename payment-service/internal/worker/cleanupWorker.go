package worker

import (
	"context"
	"log"
	"payment-service/internal/service"
	"time"
)

type CleanupWorker struct {
	service *service.PaymentService
}

func NewCleanupWorker(service *service.PaymentService) *CleanupWorker {
	return &CleanupWorker{service: service}
}

// StartCleanupJob runs the cleanup job every 5 minutes
func (w *CleanupWorker) StartCleanupJob(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting payment cleanup worker (runs every 5 minutes)...")

	// Run immediately on startup
	w.service.CleanupExpiredPayments(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping payment cleanup worker...")
			return
		case <-ticker.C:
			log.Println("Running scheduled payment cleanup...")
			w.service.CleanupExpiredPayments(ctx)
		}
	}
}
