package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oziev02/ImageProcessor/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Producer interface {
	SendTask(ctx context.Context, task *domain.ProcessingTask) error
	Close() error
}

type producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) Producer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
	return &producer{writer: writer}
}

func (p *producer) SendTask(ctx context.Context, task *domain.ProcessingTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	msg := kafka.Message{
		Key:   []byte(task.ImageID),
		Value: data,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (p *producer) Close() error {
	return p.writer.Close()
}
