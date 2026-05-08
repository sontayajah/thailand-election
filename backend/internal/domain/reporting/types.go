package reporting

// NationalSummaryResponse is returned by GET /election/national/summary.
type NationalSummaryResponse struct {
	Parties         []PartyNationalResult `json:"parties"`
	TotalVotesCast  int64                 `json:"total_votes_cast"`
	ReferendumBreak ReferendumBreakdown   `json:"referendum"`
	UpdatedAt       string                `json:"updated_at"`
}

type PartyNationalResult struct {
	PartyID           string `json:"party_id"`
	PartyName         string `json:"party_name"`
	PartyShortName    string `json:"party_short_name"`
	PartyColor        string `json:"party_color"`
	ConstituencySeats int    `json:"constituency_seats"`
	PartyListSeats    int    `json:"party_list_seats"`
	TotalSeats        int    `json:"total_seats"`
	PartyListVotes    int64  `json:"party_list_votes"`
}

type ReferendumBreakdown struct {
	AgreeVotes    int64   `json:"agree_votes"`
	DisagreeVotes int64   `json:"disagree_votes"`
	AbstainVotes  int64   `json:"abstain_votes"`
	TotalVotes    int64   `json:"total_votes"`
	AgreePct      float64 `json:"agree_pct"`
	DisagreePct   float64 `json:"disagree_pct"`
}

// ProvinceSummaryResponse is returned by GET /election/provinces/:id/summary.
type ProvinceSummaryResponse struct {
	ProvinceID   int16                  `json:"province_id"`
	ProvinceName string                 `json:"province_name"`
	BallotType   string                 `json:"ballot_type"`
	Results      []ProvinceResultEntry  `json:"results"`
}

type ProvinceResultEntry struct {
	CandidateID    string `json:"candidate_id,omitempty"`
	CandidateName  string `json:"candidate_name,omitempty"`
	PartyID        string `json:"party_id"`
	PartyName      string `json:"party_name"`
	PartyShortName string `json:"party_short_name"`
	PartyColor     string `json:"party_color"`
	TotalVotes     int64  `json:"total_votes"`
}

// PartyListCalculationResponse is returned by GET /election/party-list/calculate.
type PartyListCalculationResponse struct {
	TotalPartyListVotes int64              `json:"total_party_list_votes"`
	VotesPerSeat        float64            `json:"votes_per_seat"`
	Allocations         []SeatAllocation   `json:"allocations"`
	UpdatedAt           string             `json:"updated_at"`
}

type SeatAllocation struct {
	PartyID        string  `json:"party_id"`
	PartyName      string  `json:"party_name"`
	PartyShortName string  `json:"party_short_name"`
	PartyColor     string  `json:"party_color"`
	TotalVotes     int64   `json:"total_votes"`
	BaseSeats      int     `json:"base_seats"`
	RemainderSeats int     `json:"remainder_seats"`
	TotalSeats     int     `json:"total_seats"`
	Remainder      float64 `json:"remainder"`
}

// ReferendumSummaryResponse is returned by GET /election/referendum/summary.
type ReferendumSummaryResponse struct {
	National   ReferendumBreakdown            `json:"national"`
	ByProvince []ProvinceReferendumResult     `json:"by_province"`
}

type ProvinceReferendumResult struct {
	ProvinceID   int16   `json:"province_id"`
	ProvinceName string  `json:"province_name"`
	AgreeVotes   int64   `json:"agree_votes"`
	DisagreeVotes int64  `json:"disagree_votes"`
	AbstainVotes int64   `json:"abstain_votes"`
	TotalVotes   int64   `json:"total_votes"`
	AgreePct     float64 `json:"agree_pct"`
}
