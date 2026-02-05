package infrastructure

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func InitProductConsumerGroup(ctx context.Context, client *redis.Client) error  {
    streams := map[string]string{
        "stream:orders:created": "product-group",
        "stream:payment:failed": "product-group",
    }

    for stream, group := range streams {
        // Create the group. If it already exists, Redis returns an error: "BUSYGROUP"
        err := client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
        
        if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
            log.Printf("Error creating consumer group for %s: %v", stream, err)
            return err
        } else {
            log.Printf("Consumer group %s for stream %s is ready", group, stream)
        }
    }
    
    return nil
}

// MoveToDLQ moves a failed message to a Dead Letter Queue
func MoveToDLQ(ctx context.Context, client *redis.Client, msg redis.XMessage, dlqStream string, reason string) error {
    dlqData := msg.Values
    dlqData["error_reason"] = reason
    dlqData["failed_at"] = time.Now().Format(time.RFC3339)
    dlqData["original_id"] = msg.ID

    err := client.XAdd(ctx, &redis.XAddArgs{
        Stream: dlqStream,
        Values: dlqData,
    }).Err()

    if err != nil {
        log.Printf("CRITICAL: Failed to move message %s to DLQ %s: %v", msg.ID, dlqStream, err)
        return err
    }

    log.Printf("Successfully moved message %s to DLQ %s", msg.ID, dlqStream)
    return nil
}

func NewRedisBroker(addr string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
}