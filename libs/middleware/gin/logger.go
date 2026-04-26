package middleware

import (
	"context"
	"time"

	"libs/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery


		if path == "/api/v1/health" {
			c.Next()
			return
		}
		id := c.GetHeader("X-Correlation-ID")
        if id == "" {
            id = uuid.New().String() // Generate new if missing
        }
		c.Header("X-Correlation-ID", id)
		c.Set("correlation_id", id)
		
		ctx := context.WithValue(c.Request.Context(), "correlation_id", id)
        c.Request = c.Request.WithContext(ctx)
		
		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		status := c.Writer.Status()
		
		l := logger.ForContext(c.Request.Context())

		fields := []zap.Field{
            zap.Int("status", c.Writer.Status()),
            zap.String("method", c.Request.Method),
            zap.String("path", path),
			zap.String("query", query),
            zap.Float64("latency", float64(latency.Milliseconds())),
        }


		if len(c.Errors) > 0 {
            fields = append(fields, zap.String("error_details", c.Errors.String()))
        }

		logEntry := l.With(fields...)

		if status >= 500 {
			logEntry.Error("server error")
		} else if status >= 400 {
			logEntry.Warn("client error")
		} else {
			logEntry.Info("request processed")
		}
	}
}
