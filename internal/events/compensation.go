package events

import (
    "context"
    "encoding/json"
    "time"
    
    "github.com/segmentio/kafka-go"
    "go.uber.org/zap"
)

type CompensationProducer struct {
    writer *kafka.Writer
    logger *zap.Logger
}

func NewCompensationProducer(brokers string, logger *zap.Logger) (*CompensationProducer, error) {
    writer := &kafka.Writer{
        Addr:     kafka.TCP(brokers),
        Topic:    "compensation-events",
        Balancer: &kafka.LeastBytes{},
    }
    
    return &CompensationProducer{
        writer: writer,
        logger: logger,
    }, nil
}

func (p *CompensationProducer) PublishCompensation(event CompensationEvent) error {
    eventBytes, err := json.Marshal(event)
    if err != nil {
        p.logger.Error("Failed to marshal compensation event", zap.Error(err))
        return err
    }
    
    msg := kafka.Message{
        Key:   []byte(event.EventID),
        Value: eventBytes,
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    err = p.writer.WriteMessages(ctx, msg)
    if err != nil {
        p.logger.Error("Failed to publish compensation event", 
            zap.String("event_id", event.EventID),
            zap.Error(err))
        return err
    }
    
    p.logger.Info("Compensation event published",
        zap.String("event_id", event.EventID),
        zap.Int("order_id", event.OrderID))
    
    return nil
}

func (p *CompensationProducer) Close() error {
    if p.writer != nil {
        return p.writer.Close()
    }
    return nil
}