// Package realtime provides a thin HTTP client for the Centrifugo server API.
// Centrifugo's HTTP API is a single JSON-over-HTTP endpoint; no extra library is needed.
package realtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/th-election/backend/internal/config"
)

// Client publishes real-time events to Centrifugo via its HTTP API.
type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

// NewClient returns a Client configured from cfg.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		endpoint: cfg.Centrifugo.APIEndpoint,
		apiKey:   cfg.Centrifugo.APIKey,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// centrifugoRequest is the JSON body for the /api endpoint.
type centrifugoRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type publishParams struct {
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

// Publish sends data to a Centrifugo channel.
// Non-critical: errors are logged but not returned to callers
// (a WebSocket delivery failure must never abort a vote transaction).
func (c *Client) Publish(ctx context.Context, channel string, data any) {
	params, err := json.Marshal(publishParams{Channel: channel, Data: data})
	if err != nil {
		log.Warn().Err(err).Str("channel", channel).Msg("centrifugo: marshal params")
		return
	}

	body, err := json.Marshal(centrifugoRequest{
		Method: "publish",
		Params: params,
	})
	if err != nil {
		log.Warn().Err(err).Str("channel", channel).Msg("centrifugo: marshal request")
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		log.Warn().Err(err).Str("channel", channel).Msg("centrifugo: create request")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Warn().Err(err).Str("channel", channel).Msg("centrifugo: publish failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Warn().
			Str("channel", channel).
			Int("status", resp.StatusCode).
			Msg("centrifugo: unexpected status")
	}
}

// ── Channel name helpers ──────────────────────────────────────────────────────

// ChannelThailand is the national election update channel.
const ChannelThailand = "election:thailand"

// ChannelReferendum is the national referendum channel.
const ChannelReferendum = "election:referendum"

// ChannelProvince returns the province-specific update channel.
func ChannelProvince(provinceID int16) string {
	return fmt.Sprintf("election:province:%d", provinceID)
}
