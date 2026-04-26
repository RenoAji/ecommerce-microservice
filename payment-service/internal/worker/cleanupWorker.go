package worker

import (
	"context"
	"libs/logger"
	"payment-service/internal/service"
	"time"

	"go.uber.org/zap"
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

	logger.Log.Info("Starting payment cleanup worker", zap.Duration("interval", 5*time.Minute))

	// Run immediately on startup
	if err := w.service.CleanupExpiredPayments(ctx); err != nil {
		logger.Log.Error("failed to run payment cleanup", zap.Error(err))
	}

	for {
		select {
		case <-ctx.Done():
			logger.Log.Info("Stopping payment cleanup worker")
			return
		case <-ticker.C:
			logger.Log.Info("Running scheduled payment cleanup")
			if err := w.service.CleanupExpiredPayments(ctx); err != nil {
				logger.Log.Error("scheduled payment cleanup failed", zap.Error(err))
			}
		}
	}
}
