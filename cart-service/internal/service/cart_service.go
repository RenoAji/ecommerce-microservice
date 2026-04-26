package service

import (
	"cart-service/internal/domain"
	"cart-service/internal/repository"
	"context"
	"fmt"
	"libs/logger"
	"libs/pb"
	"strconv"

	"go.uber.org/zap"
)

type CartService struct {
	repo          repository.CartRepository
	productClient pb.ProductServiceClient
}

func NewCartService(repo repository.CartRepository, productClient pb.ProductServiceClient) *CartService {
	return &CartService{repo: repo, productClient: productClient}
}

func (s *CartService) GetCart(ctx context.Context, userID uint) (*domain.Cart, error) {
	l := logger.ForContext(ctx)
	// get array of CartItem from repository
	items, err := s.repo.GetCart(ctx, strconv.FormatUint(uint64(userID), 10))
	if err != nil {
		l.Error("failed to get cart", zap.Error(err))
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	// construct Cart object
	var cart domain.Cart
	cart.UserID = strconv.FormatUint(uint64(userID), 10)
	for _, item := range items {
		cart.Items = append(cart.Items, *item)
		cart.TotalQty += item.Quantity
		cart.TotalAmt += item.Quantity * item.Price
	}
	l.Info("Cart retrieved successfully", zap.Uint("userID", userID), zap.Int("itemCount", len(cart.Items)))
	return &cart, nil
}

func (s *CartService) GetCartItems(ctx context.Context, userID uint, productIDs []uint) ([]*domain.CartItem, error) {
	l := logger.ForContext(ctx)
	items, err := s.repo.GetCartItems(ctx, strconv.FormatUint(uint64(userID), 10), productIDs)
	if err != nil {
		l.Error("failed to get cart items", zap.Error(err))
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}
	l.Info("Cart items retrieved successfully", zap.Uint("userID", userID), zap.Int("itemCount", len(items)))
	return items, nil
}

func (s *CartService) AddToCart(ctx context.Context, userID uint, item *domain.AddCartItemRequest) error {
	l := logger.ForContext(ctx)
	resp, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: uint32(item.ProductID)})
	if err != nil {
		l.Error("failed to fetch product details", zap.Error(err))
		return fmt.Errorf("failed to fetch product details: %w", err)
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)

	// Check if item already exists
	existingItems, err := s.repo.GetCart(ctx, userIDStr)
	if err != nil {
		l.Error("failed to get existing cart items", zap.Error(err))
		return fmt.Errorf("failed to get existing cart items: %w", err)
	}
	for _, existing := range existingItems {
		if existing.ProductID == item.ProductID {
			// Update quantity instead of replacing
			newQty := existing.Quantity + item.Quantity
			err := s.repo.UpdateCartItem(ctx, userIDStr, strconv.FormatUint(uint64(item.ProductID), 10), newQty)
			if err != nil {
				l.Error("failed to update cart item quantity", zap.Error(err))
				return fmt.Errorf("failed to update cart item quantity: %w", err)
			}
			l.Info("Cart item quantity updated", zap.Uint("userID", userID), zap.Uint("productID", item.ProductID), zap.Uint("quantity", newQty))
			return nil
		}
	}

	// If not exists, add new item
	cartItem := &domain.CartItem{
		ProductID: item.ProductID,
		Quantity:  item.Quantity,
		Name:      resp.Name,
		Price:     uint(resp.Price),
	}

	err = s.repo.SaveCart(ctx, userIDStr, cartItem)
	if err != nil {
		l.Error("failed to add item to cart", zap.Error(err))
		return fmt.Errorf("failed to add item to cart: %w", err)
	}
	l.Info("Cart item added successfully", zap.Uint("userID", userID), zap.Uint("productID", item.ProductID), zap.Uint("quantity", item.Quantity))
	return nil
}

func (s *CartService) ClearCart(ctx context.Context, userID uint) error {
	l := logger.ForContext(ctx)
	err := s.repo.ClearCart(ctx, strconv.FormatUint(uint64(userID), 10))
	if err != nil {
		l.Error("failed to clear cart", zap.Error(err))
		return fmt.Errorf("failed to clear cart: %w", err)
	}
	l.Info("Cart cleared successfully", zap.Uint("userID", userID))
	return nil
}

func (s *CartService) RemoveCartItems(ctx context.Context, userID uint, productIds []uint) error {
	l := logger.ForContext(ctx)
	err := s.repo.DeleteCartItems(ctx, strconv.FormatUint(uint64(userID), 10), productIds)
	if err != nil {
		l.Error("failed to remove cart items", zap.Error(err))
		return fmt.Errorf("failed to remove cart items: %w", err)
	}
	l.Info("Cart items removed successfully", zap.Uint("userID", userID), zap.Int("itemCount", len(productIds)))
	return nil
}

func (s *CartService) UpdateCartItem(ctx context.Context, userID uint, productId string, qty uint) error {
	l := logger.ForContext(ctx)
	err := s.repo.UpdateCartItem(ctx, strconv.FormatUint(uint64(userID), 10), productId, qty)
	if err != nil {
		l.Error("failed to update cart item", zap.Error(err))
		return fmt.Errorf("failed to update cart item: %w", err)
	}
	l.Info("Cart item updated successfully", zap.Uint("userID", userID), zap.String("productID", productId), zap.Uint("quantity", qty))
	return nil
}
