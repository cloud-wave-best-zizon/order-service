package handler

import (
	"net/http"

	"github.com/cloud-wave-best-zizon/order-service/internal/domain"
	"github.com/cloud-wave-best-zizon/order-service/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OrderHandler struct {
	orderService *service.OrderService
	logger       *zap.Logger
}

func NewOrderHandler(orderService *service.OrderService, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req domain.CreateOrderRequest

	// Request binding
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Request ID from middleware
	requestID := c.GetString("request_id")

	// Create order
	order, err := h.orderService.CreateOrder(c.Request.Context(), req, requestID)
	if err != nil {
		h.logger.Error("Failed to create order",
			zap.String("request_id", requestID),
			zap.Error(err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to create order",
			"request_id": requestID,
		})
		return
	}

	// Response
	response := domain.CreateOrderResponse{
		OrderID: order.OrderID,
		Status:  order.Status,
		Message: "Order created successfully",
	}

	c.JSON(http.StatusCreated, response)
}
