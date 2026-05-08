package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// OnlineConsumer consumes the votes.online topic.
//
// Online votes differ from physical votes in one important way: the API handler
// already wrote vote_events and voter_rights_used atomically before publishing to
// Kafka. This consumer therefore calls ApplyReadModelsWithRetry (skipping the
// InsertVoteEvent step) to update constituency/party/referendum summaries, Redis
// leaderboard ZSETs, and Centrifugo channels.
type OnlineConsumer struct {
	reader  *kafka.Reader
	updater *AtomicUpdater
}

// NewOnlineConsumer creates a consumer for online votes.
func NewOnlineConsumer(brokers []string, updater *AtomicUpdater) *OnlineConsumer {
	return &OnlineConsumer{
		reader:  newReader(brokers, vkafka.TopicOnline, "vote-workers"),
		updater: updater,
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (c *OnlineConsumer) Run(ctx context.Context) {
	log.Info().Str("consumer", "online-consumer").Msg("consumer: starting")

	for {
		rawMsg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Error().Err(err).Str("consumer", "online-consumer").Msg("consumer: fetch error")
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var msg vkafka.VoteMessage
		if err := json.Unmarshal(rawMsg.Value, &msg); err != nil {
			log.Error().Err(err).
				Str("consumer", "online-consumer").
				Bytes("value", rawMsg.Value).
				Msg("consumer: unmarshal error — skipping message")
			_ = c.reader.CommitMessages(ctx, rawMsg)
			continue
		}

		// Use ApplyReadModels (not Process) — vote_events already inserted by API.
		c.updater.ApplyReadModelsWithRetry(ctx, &msg, 3)

		if err := c.reader.CommitMessages(ctx, rawMsg); err != nil {
			log.Warn().Err(err).Str("consumer", "online-consumer").Msg("consumer: commit offset failed")
		}
	}

	_ = c.reader.Close()
	log.Info().Str("consumer", "online-consumer").Msg("consumer: stopped")
}
