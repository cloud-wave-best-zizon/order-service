package service

import (
	"context"
	"time"

	"github.com/cloud-wave-best-zizon/order-service/internal/domain"
	"github.com/cloud-wave-best-zizon/order-service/internal/events"
	"github.com/cloud-wave-best-zizon/order-service/internal/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type OrderService struct {
	orderRepo *repository.OrderRepository
	producer  *events.KafkaProducer
	logger    *zap.Logger
}

func NewOrderService(orderRepo *repository.OrderRepository, producer *events.KafkaProducer, logger *zap.Logger) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		producer:  producer,
		logger:    logger,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req domain.CreateOrderRequest, requestID string) (*domain.Order, error) {
	// Order 생성
	order := &domain.Order{
		OrderID:   uuid.New().String(),
		UserID:    req.UserID,
		Items:     make([]domain.OrderItem, 0, len(req.Items)),
		Status:    domain.OrderStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Items 처리 및 총액 계산
	var totalAmount float64
	for _, item := range req.Items {
		subtotal := float64(item.Quantity) * item.Price
		orderItem := domain.OrderItem{
			ProductID:   item.ProductID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			Price:       item.Price,
			Subtotal:    subtotal,
		}
		order.Items = append(order.Items, orderItem)
		totalAmount += subtotal
	}
	order.TotalAmount = totalAmount

	// DynamoDB에 저장
	if err := s.orderRepo.CreateOrder(ctx, order); err != nil {
		s.logger.Error("Failed to save order",
			zap.String("order_id", order.OrderID),
			zap.Error(err))
		return nil, err
	}

	// Kafka 이벤트 발행
	event := events.OrderCreatedEvent{
		EventID:     uuid.New().String(),
		OrderID:     order.OrderID,
		UserID:      order.UserID,
		TotalAmount: order.TotalAmount,
		Items:       order.Items,
		Status:      string(order.Status),
		Timestamp:   time.Now(),
		RequestID:   requestID,
	}

	if err := s.producer.PublishOrderCreated(event); err != nil {
		// 이벤트 발행 실패 시 로그만 (Eventual Consistency)
		s.logger.Error("Failed to publish event",
			zap.String("order_id", order.OrderID),
			zap.Error(err))
		// TODO: Outbox Pattern 구현
	}

	s.logger.Info("Order created successfully",
		zap.String("order_id", order.OrderID),
		zap.String("user_id", order.UserID),
		zap.Float64("total_amount", order.TotalAmount))

	return order, nil
}
