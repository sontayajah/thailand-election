package main

// loader.go — loads election master data from PostgreSQL into in-memory slices
// that the physical and online simulators can use for random selection.

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	db "github.com/th-election/backend/internal/db/sqlc"
)

// electionData holds all seed data the simulator needs.
type electionData struct {
	Provinces      []db.ListProvincesRow
	Constituencies []db.ListConstituenciesByProvinceRow // all provinces combined
	// ProvinceMap maps province_id → its constituencies
	ProvinceMap map[int16][]constituencyEntry
	Parties     []db.ListPartiesRow
}

type constituencyEntry struct {
	ID         uuid.UUID
	ProvinceID int16
	// CandidateIDs for this constituency
	CandidateIDs []uuid.UUID
}

// loadElectionData queries the DB once at startup and returns the full dataset.
func loadElectionData(ctx context.Context, pool *pgxpool.Pool) (*electionData, error) {
	q := db.New(pool)

	provinces, err := q.ListProvinces(ctx)
	if err != nil {
		return nil, fmt.Errorf("list provinces: %w", err)
	}
	if len(provinces) == 0 {
		return nil, fmt.Errorf("no provinces found — run 'make seed' first")
	}

	parties, err := q.ListParties(ctx)
	if err != nil {
		return nil, fmt.Errorf("list parties: %w", err)
	}
	if len(parties) == 0 {
		return nil, fmt.Errorf("no parties found — run 'make seed' first")
	}

	provinceMap := make(map[int16][]constituencyEntry, len(provinces))

	for _, prov := range provinces {
		constits, err := q.ListConstituenciesByProvince(ctx, prov.ID)
		if err != nil {
			return nil, fmt.Errorf("list constituencies for province %d: %w", prov.ID, err)
		}

		var entries []constituencyEntry
		for _, c := range constits {
			pgUUID := pgtype.UUID{Bytes: c.ID, Valid: true}
			candidates, err := q.ListCandidatesByConstituency(ctx, pgUUID)
			if err != nil {
				return nil, fmt.Errorf("list candidates for constituency %s: %w", c.ID, err)
			}

			var cids []uuid.UUID
			for _, cand := range candidates {
				cids = append(cids, cand.ID)
			}
			if len(cids) == 0 {
				continue // skip unseeded constituencies
			}

			entries = append(entries, constituencyEntry{
				ID:           c.ID,
				ProvinceID:   c.ProvinceID,
				CandidateIDs: cids,
			})
		}

		if len(entries) > 0 {
			provinceMap[prov.ID] = entries
		}
	}

	return &electionData{
		Provinces:   provinces,
		Parties:     parties,
		ProvinceMap: provinceMap,
	}, nil
}
