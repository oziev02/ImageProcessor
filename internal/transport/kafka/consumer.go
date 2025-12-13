package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oziev02/ImageProcessor/internal/domain"
	"github.com/segmentio/kafka-go"
)

type Processor interface {
	ProcessImage(ctx context.Context, task *domain.ProcessingTask) error
}

type Consumer interface {
	Start(ctx context.Context, processor Processor) error
	Close() error
}

type consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   topic,
		GroupID: groupID,
	})
	return &consumer{reader: reader}
}

func (c *consumer) Start(ctx context.Context, processor Processor) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				return fmt.Errorf("failed to fetch message: %w", err)
			}

			var task domain.ProcessingTask
			if err := json.Unmarshal(msg.Value, &task); err != nil {
				_ = c.reader.CommitMessages(ctx, msg)
				continue
			}

			if err := processor.ProcessImage(ctx, &task); err != nil {
				// Log error but continue processing
				_ = err
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				return fmt.Errorf("failed to commit message: %w", err)
			}
		}
	}
}

func (c *consumer) Close() error {
	return c.reader.Close()
}
