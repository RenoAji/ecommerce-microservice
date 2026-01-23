package repository

import (
	"cart-service/internal/domain"
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type CartRepository interface {
	GetCart(ctx context.Context, userID string) ([]*domain.CartItem, error)
	SaveCart(ctx context.Context, userID string, item *domain.CartItem) error
	ClearCart(ctx context.Context, userID string) error
	DeleteCartItem(ctx context.Context, userID string, productID string) error
	UpdateCartItem(ctx context.Context, userID string, productID string, qty int) error
}

type RedisCartRepository struct {
	redisClient *redis.Client
}

func NewRedisCartRepository(redisClient *redis.Client) *RedisCartRepository {
	return &RedisCartRepository{redisClient: redisClient}
}

// Implement CartRepository methods here (GetCart, SaveCart, ClearCart)
func (r *RedisCartRepository) GetCart(ctx context.Context, userID string) ([]*domain.CartItem, error) {
	key := "cart:" + userID
	result, err := r.redisClient.HGetAll(ctx, key).Result()

	if err != nil {
		return nil, err
	}

	var items []*domain.CartItem
	for _, v := range result {
		var item domain.CartItem
		if err := json.Unmarshal([]byte(v), &item); err != nil {
			return nil, err
		}
		items = append(items, &item)
	}

	return items, nil
}

func (r *RedisCartRepository) SaveCart(ctx context.Context, userID string, item *domain.CartItem) error {
		key := "cart:" + userID
    data, _ := json.Marshal(item)

    // Use a pipeline to set the field and update the expiration in one go
    pipe := r.redisClient.Pipeline()
    pipe.HSet(ctx, key, item.ProductID, data)
    pipe.Expire(ctx, key, 7 * 24 * time.Hour) // 7-day TTL
    
    _, err := pipe.Exec(ctx)
    return err
}

func (r *RedisCartRepository) ClearCart(ctx context.Context, userID string) error {
	key := "cart:" + userID
	return r.redisClient.Del(ctx, key).Err()
}

func (r *RedisCartRepository) DeleteCartItem(ctx context.Context, userID string, productID string) error {
	key := "cart:" + userID
	return r.redisClient.HDel(ctx, key, productID).Err()
}

func (r *RedisCartRepository) UpdateCartItem(ctx context.Context, userID string, productID string, qty int) error {
    key := "cart:" + userID

    if qty <= 0 {
        return r.redisClient.HDel(ctx, key, productID).Err()
    }

    // 1. Get existing item
    result, err := r.redisClient.HGet(ctx, key, productID).Result()
    if err != nil {
        return err
    }

    var item domain.CartItem
    if err := json.Unmarshal([]byte(result), &item); err != nil {
        return err
    }

    // 2. Update logic
    item.Quantity = qty
    data, _ := json.Marshal(item)

    // 3. Use Pipeline to save and extend life of the cart
    pipe := r.redisClient.Pipeline()
    pipe.HSet(ctx, key, productID, data)
    pipe.Expire(ctx, key, 7 * 24 * time.Hour) // Extend cart for another 7 days
    
    _, err = pipe.Exec(ctx)
    return err
}