package worker

import (
	"context"

	vkafka "github.com/th-election/backend/internal/kafka"
)

// ConstituencyConsumer consumes the votes.constituency topic.
type ConstituencyConsumer struct {
	base baseConsumer
}

// NewConstituencyConsumer creates a consumer for constituency votes.
func NewConstituencyConsumer(brokers []string, updater *AtomicUpdater) *ConstituencyConsumer {
	return &ConstituencyConsumer{
		base: baseConsumer{
			reader:  newReader(brokers, vkafka.TopicConstituency, "vote-workers"),
			updater: updater,
			name:    "constituency-consumer",
		},
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (c *ConstituencyConsumer) Run(ctx context.Context) { c.base.run(ctx) }
