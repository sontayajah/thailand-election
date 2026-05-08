package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer wraps a map of kafka.Writer — one per topic — for type-safe publishing.
type Producer struct {
	writers map[string]*kafka.Writer
}

// NewProducer creates one kafka.Writer per topic using the supplied broker list.
// Writers are configured with LeastBytes balancer for even partition distribution.
func NewProducer(brokers []string) *Producer {
	topics := []string{TopicConstituency, TopicPartyList, TopicReferendum, TopicOnline}
	writers := make(map[string]*kafka.Writer, len(topics))

	for _, topic := range topics {
		writers[topic] = &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			MaxAttempts:  3,
			BatchTimeout: 10 * time.Millisecond,
			// RequiredAcks = WriterConfig.RequiredAcks default is All (strong durability)
			Async: false, // synchronous for guaranteed delivery
		}
	}

	return &Producer{writers: writers}
}

// Publish serialises msg as JSON and writes it to the correct topic.
// The Kafka message key is the IdempotencyKey — Kafka deduplication by key is not
// enabled here; the application-level idempotency check in the ingestion handler
// provides the guarantee instead.
func (p *Producer) Publish(ctx context.Context, msg *VoteMessage) error {
	topic := TopicForBallotType(msg.BallotType, msg.Source)

	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal vote message: %w", err)
	}

	return p.writers[topic].WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.IdempotencyKey),
		Value: payload,
	})
}

// Close gracefully shuts down all underlying Kafka writers.
func (p *Producer) Close() {
	for _, w := range p.writers {
		_ = w.Close()
	}
}
