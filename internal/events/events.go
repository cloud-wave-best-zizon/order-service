package events

import (
    "time"
    "github.com/cloud-wave-best-zizon/order-service/internal/domain"
)

type OrderCreatedEvent struct {
    EventID     string             `json:"event_id"`
    OrderID     int                `json:"order_id"`
    UserID      string             `json:"user_id"`
    TotalAmount float64            `json:"total_amount"`
    Items       []domain.OrderItem `json:"items"`
    Status      string             `json:"status"`
    Timestamp   time.Time          `json:"timestamp"`
    RequestID   string             `json:"request_id"`
}

type StockDeductionEvent struct {
    EventID   string    `json:"event_id"`
    OrderID   int       `json:"order_id"`
    ProductID string    `json:"product_id"`
    Quantity  int       `json:"quantity"`
    Timestamp time.Time `json:"timestamp"`
}

type CompensationEvent struct {
    EventID   string    `json:"event_id"`
    OrderID   int       `json:"order_id"`
    Reason    string    `json:"reason"`
    Timestamp time.Time `json:"timestamp"`
}