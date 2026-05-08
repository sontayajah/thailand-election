package main

// physical.go — simulates physical polling station vote submissions.
// Spawns one goroutine per province; each goroutine fires POST /api/v1/votes
// at the given RPS share (total RPS / province count).

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	vkafka "github.com/th-election/backend/internal/kafka"
)

// voteRequest mirrors the IngestionHandler's VoteRequest (avoids import cycle).
type voteRequest struct {
	BallotType       string `json:"ballot_type"`
	ProvinceID       int16  `json:"province_id"`
	ConstituencyID   string `json:"constituency_id,omitempty"`
	CandidateID      string `json:"candidate_id,omitempty"`
	ReferendumVote   string `json:"referendum_vote,omitempty"`
	VoteCount        int32  `json:"vote_count"`
	IdempotencyKey   string `json:"idempotency_key"`
	PayloadSignature string `json:"payload_signature"`
}

// signVote signs the canonical payload bytes with the simulator Ed25519 key.
func signVote(privKey ed25519.PrivateKey, msg *vkafka.VoteMessage) string {
	sig := ed25519.Sign(privKey, msg.SignedBytes())
	return base64.StdEncoding.EncodeToString(sig)
}

// runPhysicalSimulator drives physical vote posting until ctx is cancelled.
func runPhysicalSimulator(
	ctx context.Context,
	data *electionData,
	privKey ed25519.PrivateKey,
	apiBase string,
	rps int,
	ballotFilter string,
) (int64, int64) {
	var succeeded, failed atomic.Int64

	provinces := data.Provinces
	if len(provinces) == 0 {
		log.Error().Msg("no provinces loaded — cannot simulate")
		return 0, 0
	}

	// Distribute target RPS evenly across provinces.
	rpsPerProvince := max(1, rps/len(provinces))
	interval := time.Duration(float64(time.Second) / float64(rpsPerProvince))

	client := &http.Client{Timeout: 10 * time.Second}

	var wg sync.WaitGroup
	for _, prov := range provinces {
		provID := prov.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					ballotType := pickBallotType(ballotFilter)
					req := buildPhysicalVote(data, provID, ballotType, privKey)
					if req == nil {
						continue
					}
					if err := postVote(ctx, client, apiBase, req); err != nil {
						failed.Add(1)
						log.Debug().Err(err).
							Int16("province_id", provID).
							Str("ballot_type", ballotType).
							Msg("vote POST failed")
					} else {
						succeeded.Add(1)
					}
				}
			}
		}()
	}

	wg.Wait()
	return succeeded.Load(), failed.Load()
}

// pickBallotType returns a ballot type based on the filter flag.
func pickBallotType(filter string) string {
	if filter != "all" && filter != "" {
		return filter
	}
	types := []string{"CONSTITUENCY", "PARTY_LIST", "REFERENDUM"}
	return types[rand.IntN(len(types))]
}

// buildPhysicalVote constructs a signed VoteRequest for the given province.
func buildPhysicalVote(
	data *electionData,
	provinceID int16,
	ballotType string,
	privKey ed25519.PrivateKey,
) *voteRequest {
	constits := data.ProvinceMap[provinceID]

	msg := &vkafka.VoteMessage{
		BallotType:     ballotType,
		ProvinceID:     provinceID,
		VoteCount:      1,
		Source:         "simulator",
		AnonymousToken: uuid.New().String(),
		IdempotencyKey: fmt.Sprintf("sim:%s:%d:%s", ballotType, provinceID, uuid.New().String()),
	}

	switch ballotType {
	case "CONSTITUENCY":
		if len(constits) == 0 {
			return nil
		}
		c := constits[rand.IntN(len(constits))]
		msg.ConstituencyID = c.ID.String()
		if len(c.CandidateIDs) == 0 {
			return nil
		}
		msg.CandidateID = c.CandidateIDs[rand.IntN(len(c.CandidateIDs))].String()

	case "PARTY_LIST":
		if len(data.Parties) == 0 {
			return nil
		}
		msg.CandidateID = data.Parties[rand.IntN(len(data.Parties))].ID.String()

	case "REFERENDUM":
		votes := []string{"AGREE", "DISAGREE", "ABSTAIN"}
		// Weight: 55% agree, 35% disagree, 10% abstain — for realistic demo data
		r := rand.Float64()
		switch {
		case r < 0.55:
			msg.ReferendumVote = votes[0]
		case r < 0.90:
			msg.ReferendumVote = votes[1]
		default:
			msg.ReferendumVote = votes[2]
		}
	}

	sig := signVote(privKey, msg)

	req := &voteRequest{
		BallotType:       msg.BallotType,
		ProvinceID:       msg.ProvinceID,
		VoteCount:        msg.VoteCount,
		IdempotencyKey:   msg.IdempotencyKey,
		PayloadSignature: sig,
	}
	if msg.ConstituencyID != "" {
		req.ConstituencyID = msg.ConstituencyID
	}
	if msg.CandidateID != "" {
		req.CandidateID = msg.CandidateID
	}
	if msg.ReferendumVote != "" {
		req.ReferendumVote = msg.ReferendumVote
	}

	return req
}

// postVote sends a single vote request to the API.
func postVote(ctx context.Context, client *http.Client, apiBase string, req *voteRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/votes", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
