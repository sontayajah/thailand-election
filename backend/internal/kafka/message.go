package kafka

// VoteMessage is the canonical JSON payload published to every Kafka vote topic.
// All UUID fields are carried as lowercase hex strings with hyphens.
// Consumers decode this and hand it to the atomic updater.
type VoteMessage struct {
	// BallotType is one of: CONSTITUENCY, PARTY_LIST, REFERENDUM.
	BallotType string `json:"ballot_type"`

	// Source is one of: physical, online, simulator, admin_batch.
	Source string `json:"source"`

	// ProvinceID is the official two-digit Thai province code (1–77).
	ProvinceID int16 `json:"province_id"`

	// ConstituencyID is the UUID of the constituency (CONSTITUENCY ballots only).
	ConstituencyID string `json:"constituency_id,omitempty"`

	// CandidateID is the UUID of the candidate (CONSTITUENCY) or the party UUID
	// (PARTY_LIST — stored in vote_events.candidate_id as a design simplification).
	CandidateID string `json:"candidate_id,omitempty"`

	// ReferendumVote is AGREE | DISAGREE | ABSTAIN (REFERENDUM ballots only).
	ReferendumVote string `json:"referendum_vote,omitempty"`

	// VoteCount is the number of votes in this event (batch physical delivery).
	VoteCount int32 `json:"vote_count"`

	// AnonymousToken is a one-time UUID bridging the cast to the receipt.
	// It contains NO voter identity information.
	AnonymousToken string `json:"anonymous_token"`

	// IdempotencyKey prevents double-processing of duplicate deliveries.
	IdempotencyKey string `json:"idempotency_key"`

	// PayloadSignature is the base64-encoded Ed25519 signature from the polling station.
	PayloadSignature string `json:"payload_signature,omitempty"`
}

// Topic routing constants (must match Kafka/Redpanda topic names in docker-compose).
const (
	TopicConstituency = "votes.constituency"
	TopicPartyList    = "votes.party_list"
	TopicReferendum   = "votes.referendum"
	TopicOnline       = "votes.online"
)

// TopicForBallotType returns the correct Kafka topic for a given ballot type.
// Online votes always go to TopicOnline regardless of ballot type.
func TopicForBallotType(ballotType, source string) string {
	if source == "online" {
		return TopicOnline
	}
	switch ballotType {
	case "PARTY_LIST":
		return TopicPartyList
	case "REFERENDUM":
		return TopicReferendum
	default:
		return TopicConstituency
	}
}

// SignedBytes returns the canonical byte slice that the polling station signs
// with its Ed25519 private key.
// Format: ballot_type|province_id|constituency_id|candidate_id|referendum_vote|vote_count|idempotency_key
func (m *VoteMessage) SignedBytes() []byte {
	return []byte(m.BallotType + "|" +
		itoa16(m.ProvinceID) + "|" +
		m.ConstituencyID + "|" +
		m.CandidateID + "|" +
		m.ReferendumVote + "|" +
		itoa32(m.VoteCount) + "|" +
		m.IdempotencyKey)
}

func itoa16(n int16) string {
	if n == 0 {
		return "0"
	}
	buf := [8]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func itoa32(n int32) string {
	if n == 0 {
		return "0"
	}
	buf := [12]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n >= 10 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	pos--
	buf[pos] = byte('0' + n)
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
