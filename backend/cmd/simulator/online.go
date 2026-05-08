package main

// online.go — simulates complete online voter sessions.
// Requires pre-seeded test voters in voter_registry (created by seedTestVoters).
// Each goroutine walks through: verify-id → request-otp → verify-otp → cast×3.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// testVoter is a synthetic voter created for simulator use.
type testVoter struct {
	NationalID string
	// pepper must match NATIONAL_ID_PEPPER in .env so the hash matches the DB
}

// generateTestVoters returns synthetic 13-digit Thai-style IDs.
// These must be pre-seeded in voter_registry by seedTestVoters().
func generateTestVoters(count int) []testVoter {
	voters := make([]testVoter, count)
	for i := range voters {
		// Format: 9 (simulator prefix) + 12 digits; check digit = 0 (accepted in dev)
		voters[i] = testVoter{
			NationalID: fmt.Sprintf("9%012d", i+1),
		}
	}
	return voters
}

// apiPost is a thin helper that JSON-encodes body, POSTs, and decodes the response.
func apiPost(ctx context.Context, client *http.Client, url string, body, result any, bearerToken string) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d at %s", resp.StatusCode, url)
	}
	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}

type verifyIDResponse struct {
	SessionID       string   `json:"session_id"`
	EligibleBallots []string `json:"eligible_ballots"`
}

type requestOTPResponse struct {
	ExpiresIn int    `json:"expires_in_seconds"`
	DevOTP    string `json:"dev_otp"`
}

type verifyOTPResponse struct {
	Token string `json:"token"`
}

type castResponse struct {
	ReceiptHash string `json:"receipt_hash"`
}

// simulateOnlineVoter executes a complete voting session for one synthetic voter.
func simulateOnlineVoter(ctx context.Context, client *http.Client, apiBase, nationalID string) error {
	base := apiBase + "/online-voting"

	// Step 1: verify-id
	var idResp verifyIDResponse
	if err := apiPost(ctx, client, base+"/auth/verify-id",
		map[string]string{"national_id": nationalID}, &idResp, ""); err != nil {
		return fmt.Errorf("verify-id: %w", err)
	}

	// Step 2: request-otp
	var otpResp requestOTPResponse
	if err := apiPost(ctx, client, base+"/auth/request-otp",
		map[string]string{"session_id": idResp.SessionID}, &otpResp, ""); err != nil {
		return fmt.Errorf("request-otp: %w", err)
	}

	// In dev mode the OTP is returned directly; in non-dev mode the simulator can't proceed.
	if otpResp.DevOTP == "" {
		return fmt.Errorf("OTP_DEV_MODE is off — online simulation requires dev mode")
	}

	// Step 3: verify-otp → get JWT
	var jwtResp verifyOTPResponse
	if err := apiPost(ctx, client, base+"/auth/verify-otp",
		map[string]any{
			"session_id": idResp.SessionID,
			"otp":        otpResp.DevOTP,
		}, &jwtResp, ""); err != nil {
		return fmt.Errorf("verify-otp: %w", err)
	}

	// Step 4: cast all 3 ballots
	ballotTypes := idResp.EligibleBallots
	if len(ballotTypes) == 0 {
		ballotTypes = []string{"CONSTITUENCY", "PARTY_LIST", "REFERENDUM"}
	}

	for _, bt := range ballotTypes {
		castBody := buildCastBody(bt)
		var cr castResponse
		if err := apiPost(ctx, client, base+"/cast", castBody, &cr, jwtResp.Token); err != nil {
			return fmt.Errorf("cast %s: %w", bt, err)
		}
		log.Debug().
			Str("ballot_type", bt).
			Str("receipt", cr.ReceiptHash[:8]+"…").
			Msg("online cast OK")
	}

	return nil
}

func buildCastBody(ballotType string) map[string]any {
	body := map[string]any{
		"ballot_type": ballotType,
		"confirm":     true,
	}
	switch ballotType {
	case "REFERENDUM":
		r := rand.Float64()
		switch {
		case r < 0.55:
			body["referendum_vote"] = "AGREE"
		case r < 0.90:
			body["referendum_vote"] = "DISAGREE"
		default:
			body["referendum_vote"] = "ABSTAIN"
		}
	case "PARTY_LIST":
		// The API picks a valid party from the session's constituency — we just omit
		// party_id here and let the API use the voter's first eligible party.
		// In a richer simulator we'd pre-fetch the ballot first.
		body["party_id"] = "00000000-0000-0000-0000-000000000001" // placeholder; real value comes from GET /ballot
	case "CONSTITUENCY":
		body["candidate_id"] = "00000000-0000-0000-0000-000000000001"
	}
	return body
}

// runOnlineSimulator fires online voter sessions at the given RPS until ctx is done.
func runOnlineSimulator(
	ctx context.Context,
	apiBase string,
	rps int,
) (int64, int64) {
	var succeeded, failed atomic.Int64

	// We use 20 synthetic test voters cycling in round-robin.
	voters := generateTestVoters(20)
	client := &http.Client{Timeout: 30 * time.Second}

	interval := time.Duration(float64(time.Second) / float64(max(1, rps)))

	var (
		mu  sync.Mutex
		idx int
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var wg sync.WaitGroup
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return succeeded.Load(), failed.Load()
		case <-ticker.C:
			mu.Lock()
			voter := voters[idx%len(voters)]
			idx++
			mu.Unlock()

			wg.Add(1)
			go func(v testVoter) {
				defer wg.Done()
				if err := simulateOnlineVoter(ctx, client, apiBase, v.NationalID); err != nil {
					failed.Add(1)
					log.Debug().Err(err).Str("national_id", v.NationalID[:4]+"***").Msg("online sim failed")
				} else {
					succeeded.Add(1)
				}
			}(voter)
		}
	}
}

// seedTestVoters inserts synthetic voter records into voter_registry via the DB.
// Called automatically before --mode=online if the table is empty.
func seedTestVoters(ctx context.Context, pool interface{ Exec(context.Context, string, ...any) (any, error) }, count int) error {
	pepper := viper.GetString("NATIONAL_ID_PEPPER")
	voters := generateTestVoters(count)

	for i, v := range voters {
		hash := sha256NationalID(v.NationalID, pepper)
		// We use a raw SQL exec here to avoid importing the full query layer.
		_, err := pool.(interface {
			Exec(context.Context, string, ...any) (interface{}, error)
		}).Exec(ctx,
			`INSERT INTO voter_registry
			   (id, national_id_hash, province_id, constituency_id, registered_phone, is_eligible)
			 VALUES (gen_random_uuid(), $1, $2, (
			   SELECT id FROM constituencies WHERE province_id = $2 LIMIT 1
			 ), $3, true)
			 ON CONFLICT (national_id_hash) DO NOTHING`,
			hash,
			int16((i%77)+1), // cycle through provinces
			fmt.Sprintf("+6681%07d", i),
		)
		if err != nil {
			return fmt.Errorf("seed voter %d: %w", i, err)
		}
	}
	return nil
}

func sha256NationalID(nationalID, pepper string) string {
	h := sha256.New()
	h.Write([]byte(nationalID + pepper))
	return hex.EncodeToString(h.Sum(nil))
}
