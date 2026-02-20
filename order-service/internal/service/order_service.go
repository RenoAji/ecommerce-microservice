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
    eventRepo repository.OrderEventRepository
	cartClient pb.CartServiceClient
	productClient pb.ProductServiceClient
	paymentClient pb.PaymentServiceClient
}

func NewOrderService(repo repository.OrderRepository, eventRepo repository.OrderEventRepository, cartClient pb.CartServiceClient, productClient pb.ProductServiceClient, paymentClient pb.PaymentServiceClient) *OrderService {
	return &OrderService{ repo: repo, eventRepo: eventRepo, cartClient: cartClient, productClient: productClient, paymentClient: paymentClient}
}

func (s *OrderService) CreateOrder(req *domain.CreateOrderRequest, ctx context.Context, userID uint) (uint, error) {
    md := metadata.Pairs("authorization", "Bearer "+os.Getenv("INTERNAL_SECRET"))
    ctx = metadata.NewOutgoingContext(ctx, md)

    // Fetch cart (entire or specific items)
    var cartItems []*pb.CartItem
    userIDStr := strconv.FormatUint(uint64(userID), 10)

    if len(req.ProductIDs) > 0 {

        // Fetch specific items
        productIDs := make([]uint32, len(req.ProductIDs))
        for i, id := range req.ProductIDs {
            productIDs[i] = uint32(id)
        }

        cartResp, err := s.cartClient.GetCartItems(ctx, &pb.GetCartItemRequest{
            UserId:     userIDStr,
            ProductIds: productIDs,
        })

        if err != nil {
            return 0, fmt.Errorf("failed to fetch cart items: %w", err)
        }

        if len(cartResp.Items) != len(req.ProductIDs) {
            return 0, fmt.Errorf("invalid product id")
        }

        cartItems = cartResp.Items
    } else {
        // Fetch entire cart
        cartResp, err := s.cartClient.GetUserCart(ctx, &pb.GetCartRequest{
            UserId: userIDStr,
        })
        if err != nil {
            return 0, fmt.Errorf("failed to fetch cart: %w", err)
        }
        cartItems = cartResp.Items
        
        // Extract all product IDs for clearing
        // productIDs = make([]uint32, len(cartItems))
        // for i, item := range cartItems {
        //     productIDs[i] = item.ProductId
        // }
    }

    // Validate cart not empty
    if len(cartItems) == 0 {
        return 0, fmt.Errorf("cart is empty or no valid items found") 
    }

	// Build order items
    orderItems := make([]domain.OrderItem, 0, len(cartItems))
    var totalAmt uint

    for _, cartItem := range cartItems {

        // fetch price from product service to ensure latest price
		productResp, err := s.productClient.GetProduct(ctx, &pb.GetProductRequest{Id: cartItem.ProductId})
		if  err != nil {
			return 0, fmt.Errorf("failed to fetch product details: %w", err)
		}

        orderItems = append(orderItems, domain.OrderItem{
            ProductID: uint(cartItem.ProductId),
            Quantity:  uint(cartItem.Quantity),
            Name:      productResp.Name,
            Price:     uint(productResp.Price),
        })
        totalAmt += uint(productResp.Price) * uint(cartItem.Quantity)
    }

    

    // Create order
    order := &domain.Order{
        UserID:      userID,
        Items:       orderItems,
        TotalAmount: totalAmt,
    }

        // Save order to database
    if err := s.repo.AddOrder(ctx, order); err != nil {
        return 0, fmt.Errorf("failed to create order: %w", err)
    }
    log.Printf("Order %d created successfully", order.ID)

    // Publish order created event
    if err := s.eventRepo.PublishOrderCreatedEvent(ctx, &domain.OrderCreatedEvent{
        OrderID:     strconv.FormatUint(uint64(order.ID), 10),
        UserID:      userIDStr,
        TotalAmount: order.TotalAmount,
        Items:       domain.ConvertToOrderItemMessages(order.Items),
    }); err != nil {
        return 0, fmt.Errorf("failed to publish order created event: %w", err)
    }
    log.Println("Order Event Created")

    return order.ID, nil
}

func (s *OrderService) GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error) {
	if status != "" {
		// validate status
		validStatuses := map[string]bool{
			"RECEIVED":   true,
            "AWAITING_PAYMENT": true,
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

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	return s.repo.GetOrderByID(ctx, orderID)
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
    return s.repo.UpdateOrderStatus(ctx, orderID, status)
}

func (s *OrderService) ProcessAwaitingPaymentOrders(ctx context.Context, orderID string) error {
    // Get order details to fetch the total amount
    // log.Printf("Processing payment for order %s", orderID)
    order, err := s.repo.GetOrderByID(ctx, orderID)
    if err != nil {
        return fmt.Errorf("failed to get order: %w", err)
    }

    // Create metadata for internal gRPC call
    md := metadata.Pairs("authorization", "Bearer "+os.Getenv("INTERNAL_SECRET"))
    ctx = metadata.NewOutgoingContext(ctx, md)

    // Call payment service to get payment URL
    orderIDUint, _ := strconv.ParseUint(orderID, 10, 32)
    paymentResp, err := s.paymentClient.GetPaymentURL(ctx, &pb.GetPaymentRequest{
        OrderId: uint32(orderIDUint),
        Amount:  uint64(order.TotalAmount),
    })
    if err != nil {
        return fmt.Errorf("failed to get payment URL: %w", err)
    }

    log.Printf("Payment URL generated for order %s: %s", orderID, paymentResp.PaymentUrl)

    // Update order status to awaiting payment
    err = s.repo.UpdateOrderStatus(ctx, orderID, "AWAITING_PAYMENT")
    if err != nil {
        return fmt.Errorf("failed to update order status: %w", err)
    }

    err = s.repo.UpdatePaymentUrl(ctx, orderID, paymentResp.PaymentUrl)
    if err != nil {
        return fmt.Errorf("failed to update order payment info: %w", err)
    }

    return nil
}