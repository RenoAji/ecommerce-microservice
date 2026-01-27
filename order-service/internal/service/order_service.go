package service

import (
	"context"
	"fmt"
	"log"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"order-service/pb"
	"os"
	"strconv"

	"google.golang.org/grpc/metadata"
)

type OrderService struct {
	repo repository.OrderRepository
	cartClient pb.CartServiceClient
	productClient pb.ProductServiceClient
}

func NewOrderService(repo repository.OrderRepository, cartClient pb.CartServiceClient, productClient pb.ProductServiceClient) *OrderService {
	return &OrderService{ repo: repo, cartClient: cartClient, productClient: productClient}
}

func (s *OrderService) CreateOrder(req *domain.CreateOrderRequest, ctx context.Context, userID uint) error {
    md := metadata.Pairs("authorization", "Bearer "+os.Getenv("INTERNAL_SECRET"))
    ctx = metadata.NewOutgoingContext(ctx, md)

    // Fetch cart (entire or specific items)
    var cartItems []*pb.CartItem
    var productIDs []string
    var err error
    userIDStr := strconv.FormatUint(uint64(userID), 10)

    if len(req.ProductIDs) > 0 {
        // Use product IDs from request
        productIDs = req.ProductIDs

        // Fetch specific items
        cartResp, err := s.cartClient.GetCartItems(ctx, &pb.GetCartItemRequest{
            UserId:     userIDStr,
            ProductIds: productIDs,
        })
        if err != nil {
            return fmt.Errorf("failed to fetch cart items: %w", err)
        }

        if len(cartResp.Items) != len(productIDs) {
            return fmt.Errorf("invalid product id")
        }

        cartItems = cartResp.Items
    } else {
        // Fetch entire cart
        cartResp, err := s.cartClient.GetUserCart(ctx, &pb.GetCartRequest{
            UserId: userIDStr,
        })
        if err != nil {
            return fmt.Errorf("failed to fetch cart: %w", err)
        }
        cartItems = cartResp.Items
        
        // Extract all product IDs for clearing
        productIDs = make([]string, len(cartItems))
        for i, item := range cartItems {
            productIDs[i] = item.ProductId
        }
    }

    // Validate cart not empty
    if len(cartItems) == 0 {
        return fmt.Errorf("cart is empty or no valid items found") 
    }

	// Build order items
    orderItems := make([]domain.OrderItem, 0, len(cartItems))
    var totalAmt int64

    for _, cartItem := range cartItems {

		// fetch price from product service to ensure latest price
		productResp, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: cartItem.ProductId})
		if  err != nil {
			return fmt.Errorf("failed to fetch product details: %w", err)
		}

        orderItems = append(orderItems, domain.OrderItem{
            ProductID: cartItem.ProductId,
            Quantity:  int(cartItem.Quantity),
            Name:      productResp.Name,
            Price:     productResp.Price,
        })
        totalAmt += productResp.Price * int64(cartItem.Quantity)
    }

    // Create order
    userIDUint, err := strconv.ParseUint(userIDStr, 10, 64)
    if err != nil {
        return fmt.Errorf("invalid user ID: %w", err)
    }

    order := &domain.Order{
        UserID:      uint(userIDUint),
        Items:       orderItems,
        TotalAmount: totalAmt,
    }

    if err := s.repo.AddOrder(ctx, order); err != nil {
        return fmt.Errorf("failed to create order: %w", err)
    }

    // Clear/remove cart items after successful order
    if len(req.ProductIDs) > 0 {
        _, err = s.cartClient.RemoveCartItems(ctx, &pb.GetCartItemRequest{
            UserId:     userIDStr,
            ProductIds: productIDs,
        })
    } else {
        _, err = s.cartClient.ClearUserCart(ctx, &pb.GetCartRequest{
            UserId: userIDStr,
        })
    }

    // Log error but don't fail the order
    if err != nil {
        log.Printf("warning: failed to clear cart: %v", err)
    }

    return nil
}

func (s *OrderService) GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error) {
	if status != "" {
		// validate status
		validStatuses := map[string]bool{
			"PENDING":   true,
			"PAID":      true,
			"SHIPPED":   true,
			"CANCELLED": true,
		}
		if !validStatuses[status] {
			return nil, fmt.Errorf("invalid order status: %s", status)
		}
	}
	return s.repo.GetOrders(ctx, userID, status)
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string, userID uint) (*domain.Order, error) {
	return s.repo.GetOrderByID(ctx, orderID, userID)
}