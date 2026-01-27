package service

import (
	"cart-service/internal/domain"
	"cart-service/internal/repository"
	"cart-service/pb"
	"context"
	"fmt"
	"os"
	"strconv"

	"google.golang.org/grpc/metadata"
)


type CartService struct {
	repo repository.CartRepository
	productClient pb.ProductServiceClient
}

func NewCartService(repo repository.CartRepository, productClient pb.ProductServiceClient) *CartService {
	return &CartService{repo: repo, productClient: productClient}
}

func (s *CartService) GetCart(ctx context.Context, userID uint) (*domain.Cart, error) {
	// get array of CartItem from repository
	items, err := s.repo.GetCart(ctx, strconv.FormatUint(uint64(userID), 10))
	if err != nil {
		return nil, err
	}

	// construct Cart object
	var cart domain.Cart
	cart.UserID = strconv.FormatUint(uint64(userID), 10)
	for _, item := range items {
		cart.Items = append(cart.Items, *item)
		cart.TotalQty += item.Quantity
		cart.TotalAmt += int64(item.Quantity) * item.Price
	}
	return &cart, nil
}

func (s *CartService) GetCartItems(ctx context.Context, userID uint, productIDs []string) ([]*domain.CartItem, error) {
	return s.repo.GetCartItems(ctx, strconv.FormatUint(uint64(userID), 10), productIDs)
}

func (s *CartService) AddToCart(ctx context.Context, userID uint, item *domain.AddCartItemRequest) error {
    md := metadata.Pairs("authorization", "Bearer "+os.Getenv("INTERNAL_SECRET"))
    ctx = metadata.NewOutgoingContext(ctx, md)

    resp, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: item.ProductID})
    if err != nil {
        return fmt.Errorf("failed to fetch product details: %w", err)
    }

    userIDStr := strconv.FormatUint(uint64(userID), 10)
    
    // Check if item already exists
    existingItems, _ := s.repo.GetCart(ctx, userIDStr)
    for _, existing := range existingItems {
        if existing.ProductID == item.ProductID {
            // Update quantity instead of replacing
            newQty := existing.Quantity + item.Quantity
            return s.repo.UpdateCartItem(ctx, userIDStr, item.ProductID, newQty)
        }
    }

    // If not exists, add new item
    cartItem := &domain.CartItem{
        ProductID: item.ProductID,
        Quantity:  item.Quantity,
		Name:      resp.Name,
        Price:     resp.Price,
    }

    return s.repo.SaveCart(ctx, userIDStr, cartItem)
}

func (s *CartService) ClearCart(ctx context.Context, userID uint) error {
	return s.repo.ClearCart(ctx, strconv.FormatUint(uint64(userID), 10))
}	

func (s *CartService) RemoveCartItems(ctx context.Context, userID uint, productIds []string) error {
	return s.repo.DeleteCartItems(ctx, strconv.FormatUint(uint64(userID), 10), productIds)
}

func (s *CartService) UpdateCartItem(ctx context.Context, userID uint, productId string, qty int) error {
	return s.repo.UpdateCartItem(ctx, strconv.FormatUint(uint64(userID), 10), productId, qty)
}