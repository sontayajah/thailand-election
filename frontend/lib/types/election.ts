// TypeScript types mirroring the Go backend response schemas.
// Field names match the JSON keys emitted by the API.

export interface Province {
  id: number;
  name_th: string;
  name_en: string;
  region: string;
  constituency_count: number;
  eligible_voters: number;
  svg_path_id?: string;
}

export interface Party {
  id: string;
  name: string;
  short_name: string;
  color_hex: string;
  logo_url?: string;
}

// ── National Summary ──────────────────────────────────────────────────────────

export interface PartyNationalResult {
  party_id: string;
  party_name: string;
  party_short_name: string;
  party_color: string;
  constituency_seats: number;
  party_list_seats: number;
  total_seats: number;
  party_list_votes: number;
}

export interface ReferendumBreakdown {
  agree_votes: number;
  disagree_votes: number;
  abstain_votes: number;
  total_votes: number;
  agree_pct: number;
  disagree_pct: number;
}

export interface NationalSummary {
  parties: PartyNationalResult[];
  total_votes_cast: number;
  referendum: ReferendumBreakdown;
  updated_at: string;
}

// ── Province Summary ──────────────────────────────────────────────────────────

export interface ProvinceResultEntry {
  candidate_id?: string;
  candidate_name?: string;
  party_id: string;
  party_name: string;
  party_short_name: string;
  party_color: string;
  total_votes: number;
}

export interface ProvinceSummary {
  province_id: number;
  province_name: string;
  ballot_type: string;
  results: ProvinceResultEntry[];
}

// ── Party List Calculation ────────────────────────────────────────────────────

export interface SeatAllocation {
  party_id: string;
  party_name: string;
  party_short_name: string;
  party_color: string;
  total_votes: number;
  base_seats: number;
  remainder_seats: number;
  total_seats: number;
  remainder: number;
}

export interface PartyListCalculation {
  total_party_list_votes: number;
  votes_per_seat: number;
  allocations: SeatAllocation[];
  updated_at: string;
}

// ── Referendum ────────────────────────────────────────────────────────────────

export interface ProvinceReferendumResult {
  province_id: number;
  province_name: string;
  agree_votes: number;
  disagree_votes: number;
  abstain_votes: number;
  total_votes: number;
  agree_pct: number;
}

export interface ReferendumSummary {
  national: ReferendumBreakdown;
  by_province: ProvinceReferendumResult[];
}

// ── WebSocket events (published by AtomicUpdater) ─────────────────────────────

export type BallotType = 'CONSTITUENCY' | 'PARTY_LIST' | 'REFERENDUM';

export interface VoteUpdateEvent {
  ballot_type: BallotType;
  province_id?: number;
  candidate_id?: string;
  vote_count: number;
  referendum_vote?: string;
}
