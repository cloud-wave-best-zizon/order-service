package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloud-wave-best-zizon/order-service/internal/domain"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.uber.org/zap"
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

type KafkaProducer struct {
	producer *kafka.Producer
	logger   *zap.Logger
}

func NewKafkaProducer(brokers string, logger *zap.Logger) (*KafkaProducer, error) {
	config := &kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"acks":                     "all",
		"retries":                  10,
		"max.in.flight.requests.per.connection": 5,
		"enable.idempotence":       true,
		"compression.type":         "snappy",
		"batch.size":              16384,
		"linger.ms":               5,
		"request.timeout.ms":      30000,
		"delivery.timeout.ms":     120000,
		"go.delivery.reports":     true,
	}

	p, err := kafka.NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	producer := &KafkaProducer{
		producer: p,
		logger:   logger,
	}

	// 비동기 전송 결과 처리 고루틴
	go producer.handleDeliveryReports()

	return producer, nil
}

func (p *KafkaProducer) handleDeliveryReports() {
	for e := range p.producer.Events() {
		switch ev := e.(type) {
		case *kafka.Message:
			if ev.TopicPartition.Error != nil {
				p.logger.Error("Message delivery failed",
					zap.Error(ev.TopicPartition.Error),
					zap.String("topic", *ev.TopicPartition.Topic),
					zap.Int32("partition", ev.TopicPartition.Partition),
					zap.String("key", string(ev.Key)))
			} else {
				p.logger.Debug("Message delivered successfully",
					zap.String("topic", *ev.TopicPartition.Topic),
					zap.Int32("partition", ev.TopicPartition.Partition),
					zap.Int64("offset", int64(ev.TopicPartition.Offset)),
					zap.String("key", string(ev.Key)))
			}
		case kafka.Error:
			p.logger.Error("Kafka error", zap.Error(ev))
		default:
			p.logger.Debug("Ignored kafka event", zap.Any("event", ev))
		}
	}
}

func (p *KafkaProducer) PublishOrderCreated(event OrderCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	topic := "order-events"
	orderIDKey := fmt.Sprintf("ORDER#%d", event.OrderID)
	
	p.logger.Info("Publishing order created event",
		zap.Int("order_id", event.OrderID),
		zap.String("user_id", event.UserID),
		zap.String("request_id", event.RequestID))

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(orderIDKey),
		Value: data,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte("order_created")},
			{Key: "service", Value: []byte("order-service")},
			{Key: "version", Value: []byte("v1")},
		},
	}

	return p.producer.Produce(message, nil)
}

func (p *KafkaProducer) Close() {
	p.logger.Info("Closing Kafka producer")
	
	// 남은 메시지들이 전송될 때까지 대기 (최대 5초)
	remaining := p.producer.Flush(5000)
	if remaining > 0 {
		p.logger.Warn("Some messages were not delivered", zap.Int("remaining", remaining))
	}
	
	p.producer.Close()
	p.logger.Info("Kafka producer closed")
}

// HealthCheck는 Kafka producer의 상태를 확인합니다
func (p *KafkaProducer) HealthCheck() error {
	metadata, err := p.producer.GetMetadata(nil, false, 5000)
	if err != nil {
		return fmt.Errorf("kafka producer health check failed: %w", err)
	}

	if len(metadata.Brokers) == 0 {
		return fmt.Errorf("no kafka brokers available")
	}

	return nil
}