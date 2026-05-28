package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	brokers []string
}

func NewPublisher(bootstrapServers string) *Publisher {
	parts := strings.Split(bootstrapServers, ",")
	brokers := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			brokers = append(brokers, p)
		}
	}
	return &Publisher{brokers: brokers}
}

func (p *Publisher) Publish(ctx context.Context, topic, key string, payload map[string]interface{}) error {
	if strings.TrimSpace(topic) == "" {
		return errors.New("topic is required")
	}
	if len(p.brokers) == 0 {
		return errors.New("kafka bootstrap servers are not configured")
	}

	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(p.brokers...),
		Topic:        topic,
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchTimeout: 250 * time.Millisecond,
	}
	defer writer.Close()

	msg := kafka.Message{Key: []byte(key), Value: value, Time: time.Now().UTC()}
	return writer.WriteMessages(ctx, msg)
}
