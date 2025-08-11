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
    OrderID       string      `json:"order_id"`
    UserID        string      `json:"user_id"`
    Items         []OrderItem `json:"items"`
    TotalAmount   float64     `json:"total_amount"`
    Status        OrderStatus `json:"status"`
    IdempotencyKey string     `json:"idempotency_key"`
    CreatedAt     time.Time   `json:"created_at"`
    UpdatedAt     time.Time   `json:"updated_at"`
}

type OrderItem struct {
    ProductID   string  `json:"product_id"`
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
    OrderID string      `json:"order_id"`
    Status  OrderStatus `json:"status"`
}

type GetOrderResponse struct {
    OrderID     string      `json:"order_id"`
    UserID      string      `json:"user_id"`
    Items       []OrderItem `json:"items"`
    TotalAmount float64     `json:"total_amount"`
    Status      OrderStatus `json:"status"`
    CreatedAt   time.Time   `json:"created_at"`
}