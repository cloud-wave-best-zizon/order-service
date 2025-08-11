package events

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloud-wave-best-zizon/order-service/internal/domain"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type OrderCreatedEvent struct {
	EventID     string             `json:"event_id"`
	OrderID     int                `json:"order_id"`  // int로 변경
	UserID      string             `json:"user_id"`
	TotalAmount float64            `json:"total_amount"`
	Items       []domain.OrderItem `json:"items"`
	Status      string             `json:"status"`
	Timestamp   time.Time          `json:"timestamp"`
	RequestID   string             `json:"request_id"`
}

type KafkaProducer struct {
	producer *kafka.Producer
}

func NewKafkaProducer(brokers string) (*KafkaProducer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
		"acks":              "all",
		"retries":           10,
	})

	if err != nil {
		return nil, err
	}

	// 비동기 전송 결과 처리
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					// 로그 처리
				}
			}
		}
	}()

	return &KafkaProducer{producer: p}, nil
}

func (p *KafkaProducer) PublishOrderCreated(event OrderCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	topic := "order-events"
	orderIDKey := fmt.Sprintf("ORDER#%d", event.OrderID)
	
	return p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(orderIDKey),
		Value: data,
	}, nil)
}

func (p *KafkaProducer) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
}