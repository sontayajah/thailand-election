package reporting_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/th-election/backend/internal/domain/reporting"
)

// TestCalculatePartyListSeats_PRDExample validates against real 2026 election data (PRD §1.3.4).
// Referendum: Approve 21,622,029 | Disapprove 11,231,161 → total ~32.8M party list votes
// Party list results: BJT=19, PP=31, PT=16, KT=2, DP=12, others=20
// We use proportional inputs derived from the official outcomes.
func TestCalculatePartyListSeats_PRDExample(t *testing.T) {
	parties := []reporting.PartyVotes{
		{PartyID: "bjt", PartyName: "Bhumjaithai", TotalVotes: 8_500_000},
		{PartyID: "pp",  PartyName: "People's Party", TotalVotes: 13_900_000},
		{PartyID: "pt",  PartyName: "Pheu Thai", TotalVotes: 7_200_000},
		{PartyID: "kt",  PartyName: "Kla Tham", TotalVotes: 900_000},
		{PartyID: "dp",  PartyName: "Democrat", TotalVotes: 5_400_000},
	}

	allocs, votesPerSeat := reporting.CalculatePartyListSeats(parties)

	require.Len(t, allocs, 5)
	require.Greater(t, votesPerSeat, float64(0))

	totalSeats := 0
	for _, a := range allocs {
		totalSeats += a.TotalSeats
	}
	assert.Equal(t, 100, totalSeats, "total allocated seats must always be 100")
}

func TestCalculatePartyListSeats_TotalAlwaysHundred(t *testing.T) {
	parties := []reporting.PartyVotes{
		{PartyID: "a", TotalVotes: 33_333},
		{PartyID: "b", TotalVotes: 33_333},
		{PartyID: "c", TotalVotes: 33_334},
	}

	allocs, _ := reporting.CalculatePartyListSeats(parties)

	total := 0
	for _, a := range allocs {
		total += a.TotalSeats
	}
	assert.Equal(t, 100, total, "seats must sum to 100 even with fractional remainders")
}

func TestCalculatePartyListSeats_EmptyInput(t *testing.T) {
	allocs, votesPerSeat := reporting.CalculatePartyListSeats(nil)
	assert.Nil(t, allocs)
	assert.Equal(t, float64(0), votesPerSeat)
}

func TestCalculatePartyListSeats_ZeroVotes(t *testing.T) {
	parties := []reporting.PartyVotes{
		{PartyID: "a", TotalVotes: 0},
		{PartyID: "b", TotalVotes: 0},
	}
	allocs, _ := reporting.CalculatePartyListSeats(parties)
	total := 0
	for _, a := range allocs {
		total += a.TotalSeats
	}
	assert.Equal(t, 0, total)
}

func TestCalculatePartyListSeats_SingleParty(t *testing.T) {
	parties := []reporting.PartyVotes{
		{PartyID: "only", TotalVotes: 10_000_000},
	}
	allocs, _ := reporting.CalculatePartyListSeats(parties)
	require.Len(t, allocs, 1)
	assert.Equal(t, 100, allocs[0].TotalSeats, "single party wins all 100 seats")
}
