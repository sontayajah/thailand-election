package reporting

import (
	"math"
	"sort"
)

const totalPartyListSeats = 100

// PartyVotes is the input to the seat calculator.
type PartyVotes struct {
	PartyID        string
	PartyName      string
	PartyShortName string
	PartyColor     string
	TotalVotes     int64
}

// CalculatePartyListSeats implements the Thai party-list seat allocation formula (PRD §1.3.3):
//
//	Seats per party = Party votes ÷ (Total all-party votes ÷ 100)
//	Remainder seats → distributed to parties with highest fractional remainders.
//
// Returns a slice of SeatAllocation sorted by total seats descending.
func CalculatePartyListSeats(parties []PartyVotes) ([]SeatAllocation, float64) {
	if len(parties) == 0 {
		return nil, 0
	}

	// Sum all party votes
	var total int64
	for _, p := range parties {
		total += p.TotalVotes
	}
	if total == 0 {
		allocs := make([]SeatAllocation, len(parties))
		for i, p := range parties {
			allocs[i] = SeatAllocation{
				PartyID:        p.PartyID,
				PartyName:      p.PartyName,
				PartyShortName: p.PartyShortName,
				PartyColor:     p.PartyColor,
				TotalVotes:     p.TotalVotes,
			}
		}
		return allocs, 0
	}

	votesPerSeat := float64(total) / float64(totalPartyListSeats)

	// Phase 1: compute exact quota, floor for base seats
	type withRemainder struct {
		SeatAllocation
		exact float64
	}

	allocations := make([]withRemainder, len(parties))
	totalBaseSeats := 0

	for i, p := range parties {
		exact := float64(p.TotalVotes) / votesPerSeat
		base := int(math.Floor(exact))
		remainder := exact - float64(base)
		allocations[i] = withRemainder{
			SeatAllocation: SeatAllocation{
				PartyID:        p.PartyID,
				PartyName:      p.PartyName,
				PartyShortName: p.PartyShortName,
				PartyColor:     p.PartyColor,
				TotalVotes:     p.TotalVotes,
				BaseSeats:      base,
				Remainder:      remainder,
			},
			exact: exact,
		}
		totalBaseSeats += base
	}

	// Phase 2: distribute remainder seats to parties with highest fractional remainders
	remainderSeats := totalPartyListSeats - totalBaseSeats

	// Sort by remainder descending; tie-break by total votes descending
	sort.Slice(allocations, func(i, j int) bool {
		if allocations[i].Remainder != allocations[j].Remainder {
			return allocations[i].Remainder > allocations[j].Remainder
		}
		return allocations[i].TotalVotes > allocations[j].TotalVotes
	})

	for i := 0; i < remainderSeats && i < len(allocations); i++ {
		allocations[i].RemainderSeats = 1
	}

	// Compute totals and sort final output by total seats descending
	result := make([]SeatAllocation, len(allocations))
	for i, a := range allocations {
		a.TotalSeats = a.BaseSeats + a.RemainderSeats
		result[i] = a.SeatAllocation
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalSeats != result[j].TotalSeats {
			return result[i].TotalSeats > result[j].TotalSeats
		}
		return result[i].TotalVotes > result[j].TotalVotes
	})

	return result, votesPerSeat
}
