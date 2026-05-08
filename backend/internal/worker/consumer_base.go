package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// baseConsumer encapsulates the Kafka fetch-decode-process loop shared by all
// four vote consumer goroutines.
type baseConsumer struct {
	reader  *kafka.Reader
	updater *AtomicUpdater
	name    string
}

// newReader creates a kafka.Reader for the given topic and consumer group.
func newReader(brokers []string, topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       1 << 20, // 1 MB
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
		RetentionTime:  7 * 24 * time.Hour,
	})
}

// run starts the fetch-decode-process loop. It blocks until ctx is cancelled.
func (b *baseConsumer) run(ctx context.Context) {
	log.Info().Str("consumer", b.name).Msg("consumer: starting")

	for {
		rawMsg, err := b.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break // context cancelled — normal shutdown
			}
			log.Error().Err(err).Str("consumer", b.name).Msg("consumer: fetch error")
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var msg vkafka.VoteMessage
		if err := json.Unmarshal(rawMsg.Value, &msg); err != nil {
			log.Error().Err(err).
				Str("consumer", b.name).
				Bytes("value", rawMsg.Value).
				Msg("consumer: unmarshal error — skipping message")
			_ = b.reader.CommitMessages(ctx, rawMsg)
			continue
		}

		b.updater.ProcessWithRetry(ctx, &msg, 3)

		if err := b.reader.CommitMessages(ctx, rawMsg); err != nil {
			log.Warn().Err(err).Str("consumer", b.name).Msg("consumer: commit offset failed")
		}
	}

	_ = b.reader.Close()
	log.Info().Str("consumer", b.name).Msg("consumer: stopped")
}
