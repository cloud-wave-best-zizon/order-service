package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// StockDeductionFailedEvent는 재고 차감 실패 시 발행하는 이벤트
type StockDeductionFailedEvent struct {
	EventID   string    `json:"event_id"`
	OrderID   int       `json:"order_id"`
	ProductID string    `json:"product_id"`
	Quantity  int       `json:"quantity"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

type CompensationProducer struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

func NewCompensationProducer(brokers string, logger *zap.Logger) (*CompensationProducer, error) {
	config := &kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
		"retries":           10,
		"enable.idempotence": true,
	}

	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create compensation producer: %w", err)
	}

	return &CompensationProducer{
		producer: p,
		logger:   logger,
	}, nil
}

func (cp *CompensationProducer) PublishStockDeductionFailed(orderID int, productID string, quantity int, reason string) error {
	event := StockDeductionFailedEvent{
		EventID:   uuid.New().String(),
		OrderID:   orderID,
		ProductID: productID,
		Quantity:  quantity,
		Reason:    reason,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal compensation event: %w", err)
	}

	topic := "order-compensation"
	key := fmt.Sprintf("ORDER#%d", orderID)

	cp.logger.Warn("Publishing stock deduction failed event",
		zap.Int("order_id", orderID),
		zap.String("product_id", productID),
		zap.String("reason", reason))

	return cp.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(key),
		Value: data,
	}, nil)
}

func (cp *CompensationProducer) Close() {
	cp.producer.Flush(5000)
	cp.producer.Close()
}