package domain

import (
	"time"
)

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	OrderID        int         `json:"order_id"` // int로 변경
	UserID         string      `json:"user_id"`
	Items          []OrderItem `json:"items"`
	TotalAmount    float64     `json:"total_amount"`
	Status         OrderStatus `json:"status"`
	IdempotencyKey string      `json:"idempotency_key"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ProductID   int     `json:"product_id"` //int 변경
	ProductName string  `json:"product_name"`
	Quantity    int     `json:"quantity"`
	Price       float64 `json:"price"`
}

type CreateOrderRequest struct {
	UserID         string      `json:"user_id" binding:"required"`
	Items          []OrderItem `json:"items" binding:"required,min=1"`
	IdempotencyKey string      `json:"idempotency_key" binding:"required"`
}

type CreateOrderResponse struct {
	OrderID int         `json:"order_id"`
	Status  OrderStatus `json:"status"`
	Message string      `json:"message"`
}

type GetOrderResponse struct {
	OrderID     int         `json:"order_id"`
	UserID      string      `json:"user_id"`
	Items       []OrderItem `json:"items"`
	TotalAmount float64     `json:"total_amount"`
	Status      OrderStatus `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
}
