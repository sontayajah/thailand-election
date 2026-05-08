package worker

import (
	"context"

	vkafka "github.com/th-election/backend/internal/kafka"
)

// ReferendumConsumer consumes the votes.referendum topic.
type ReferendumConsumer struct {
	base baseConsumer
}

// NewReferendumConsumer creates a consumer for referendum votes.
func NewReferendumConsumer(brokers []string, updater *AtomicUpdater) *ReferendumConsumer {
	return &ReferendumConsumer{
		base: baseConsumer{
			reader:  newReader(brokers, vkafka.TopicReferendum, "vote-workers"),
			updater: updater,
			name:    "referendum-consumer",
		},
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (c *ReferendumConsumer) Run(ctx context.Context) { c.base.run(ctx) }
