package worker

import (
	"context"

	vkafka "github.com/th-election/backend/internal/kafka"
)

// PartyListConsumer consumes the votes.party_list topic.
type PartyListConsumer struct {
	base baseConsumer
}

// NewPartyListConsumer creates a consumer for party-list votes.
func NewPartyListConsumer(brokers []string, updater *AtomicUpdater) *PartyListConsumer {
	return &PartyListConsumer{
		base: baseConsumer{
			reader:  newReader(brokers, vkafka.TopicPartyList, "vote-workers"),
			updater: updater,
			name:    "party-list-consumer",
		},
	}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (c *PartyListConsumer) Run(ctx context.Context) { c.base.run(ctx) }
