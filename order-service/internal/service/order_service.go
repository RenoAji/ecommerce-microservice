package service

import (
	"context"
	"fmt"
	"libs/logger"
	"libs/pb"
	"order-service/internal/domain"
	"order-service/internal/repository"
	"strconv"

	"go.uber.org/zap"
)

type OrderService struct {
	repo          repository.OrderRepository
	eventRepo     repository.OrderEventRepository
	cartClient    pb.CartServiceClient
	productClient pb.ProductServiceClient
	paymentClient pb.PaymentServiceClient
}

func NewOrderService(repo repository.OrderRepository, eventRepo repository.OrderEventRepository, cartClient pb.CartServiceClient, productClient pb.ProductServiceClient, paymentClient pb.PaymentServiceClient) *OrderService {
	return &OrderService{repo: repo, eventRepo: eventRepo, cartClient: cartClient, productClient: productClient, paymentClient: paymentClient}
}

func (s *OrderService) CreateOrder(req *domain.CreateOrderRequest, ctx context.Context, userID uint) (uint, error) {
	l := logger.ForContext(ctx)
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
			l.Error("failed to fetch cart items", zap.Error(err))
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
			l.Error("failed to fetch cart", zap.Error(err))
			return 0, fmt.Errorf("failed to fetch cart: %w", err)
		}
		cartItems = cartResp.Items
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
		if err != nil {
			l.Error("failed to fetch product details", zap.Error(err))
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
		l.Error("failed to create order", zap.Error(err))
		return 0, fmt.Errorf("failed to create order: %w", err)
	}
	l.Info("Order created successfully",
		zap.Uint("orderID", order.ID), zap.Uint("userID", userID), zap.Uint("totalAmount", totalAmt))

	// Publish order created event
	if err := s.eventRepo.PublishOrderCreatedEvent(ctx, &domain.OrderEvent{
		OrderID:       strconv.FormatUint(uint64(order.ID), 10),
		UserID:        userIDStr,
		TotalAmount:   order.TotalAmount,
		Items:         domain.ConvertToOrderItemMessages(order.Items),
		CorrelationID: correlationIDFromContext(ctx),
	}); err != nil {
		l.Error("failed to publish order created event", zap.Error(err))
		return 0, fmt.Errorf("failed to publish order created event: %w", err)
	}
	l.Info("Order Event Created", zap.String("orderID", strconv.FormatUint(uint64(order.ID), 10)))

	return order.ID, nil
}

func (s *OrderService) GetOrders(ctx context.Context, userID uint, status string) ([]domain.Order, error) {
	l := logger.ForContext(ctx)
	if status != "" {
		// validate status
		validStatuses := map[string]bool{
			"RECEIVED":         true,
			"AWAITING_PAYMENT": true,
			"PAID":             true,
			"SHIPPED":          true,
			"DELIVERED":        true,
			"CANCELLED":        true,
			"FAILED":           true,
		}
		if !validStatuses[status] {
			return nil, fmt.Errorf("invalid order status: %s", status)
		}
	}

	orders, err := s.repo.GetOrders(ctx, userID, status)
	if err != nil {
		l.Error("failed to get orders", zap.Error(err))
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	l.Info("Orders retrieved successfully", zap.Uint("userID", userID), zap.String("order_status", status), zap.Int("count", len(orders)))
	return orders, nil
}

func (s *OrderService) GetOrderByID(ctx context.Context, orderID string) (*domain.Order, error) {
	l := logger.ForContext(ctx)
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		l.Error("failed to get order by id", zap.Error(err))
		return nil, fmt.Errorf("failed to get order by id: %w", err)
	}

	l.Info("Order retrieved successfully", zap.String("orderID", orderID))
	return order, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, status string) error {
	l := logger.ForContext(ctx)
	err := s.repo.UpdateOrderStatus(ctx, orderID, status)
	if err != nil {
		l.Error("failed to update order status", zap.Error(err))
		return fmt.Errorf("failed to update order status: %w", err)
	}

	l.Info("Order status updated", zap.String("orderID", orderID), zap.String("order_status", status))

	return nil
}

func (s *OrderService) UpdateOrderToPaid(ctx context.Context, orderID string) error {
	l := logger.ForContext(ctx)
	// Update order status to PAID
	err := s.repo.UpdateOrderStatus(ctx, orderID, "PAID")
	if err != nil {
		l.Error("failed to update order status", zap.Error(err))
		return fmt.Errorf("failed to update order status: %w", err)
	}
	l.Info("Order marked as PAID", zap.String("orderID", orderID))

	// Publish order paid event
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		l.Error("failed to get order for event publishing", zap.Error(err))
		return fmt.Errorf("failed to get order for event publishing: %w", err)
	}

	userIDStr := strconv.FormatUint(uint64(order.UserID), 10)
	if err := s.eventRepo.PublishOrderPaidEvent(ctx, &domain.OrderEvent{
		OrderID:       orderID,
		UserID:        userIDStr,
		TotalAmount:   order.TotalAmount,
		Items:         domain.ConvertToOrderItemMessages(order.Items),
		CorrelationID: correlationIDFromContext(ctx),
	}); err != nil {
		l.Error("failed to publish order paid event", zap.Error(err))
		return fmt.Errorf("failed to publish order paid event: %w", err)
	}
	l.Info("Order Paid Event Sent", zap.String("orderID", orderID))

	return nil
}

func (s *OrderService) ProcessAwaitingPaymentOrders(ctx context.Context, orderID string) error {
	l := logger.ForContext(ctx)
	// Get order details to fetch the total amount
	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		l.Error("failed to get order", zap.Error(err))
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Call payment service to get payment URL
	orderIDUint, err := strconv.ParseUint(orderID, 10, 32)
	if err != nil {
		l.Error("invalid order id format", zap.Error(err))
		return fmt.Errorf("invalid order id format: %w", err)
	}

	paymentResp, err := s.paymentClient.GetPaymentURL(ctx, &pb.GetPaymentRequest{
		OrderId: uint32(orderIDUint),
		Amount:  uint64(order.TotalAmount),
	})
	if err != nil {
		l.Error("failed to get payment URL", zap.Error(err))
		return fmt.Errorf("failed to get payment URL: %w", err)
	}

	l.Info("Payment URL generated for order", zap.String("orderID", orderID), zap.String("paymentUrl", paymentResp.PaymentUrl))

	// Update order status to awaiting payment
	err = s.repo.UpdateOrderStatus(ctx, orderID, "AWAITING_PAYMENT")
	if err != nil {
		l.Error("failed to update order status", zap.Error(err))
		return fmt.Errorf("failed to update order status: %w", err)
	}

	err = s.repo.UpdatePaymentUrl(ctx, orderID, paymentResp.PaymentUrl)
	if err != nil {
		l.Error("failed to update order payment info", zap.Error(err))
		return fmt.Errorf("failed to update order payment info: %w", err)
	}

	l.Info("Order payment info updated", zap.String("orderID", orderID))
	return nil
}

func correlationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		return correlationID
	}

	return ""
}
