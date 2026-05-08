// Voting portal API — TanStack Query mutations + fetchers for the online voting flow.
// All protected endpoints attach the voter JWT from sessionStorage.

import { useMutation, useQuery } from '@tanstack/react-query';
import { getVoterJWT } from '@/lib/voting/session';

// Voting portal is always browser-side (uses sessionStorage, JWT).
// It always goes through nginx → Kong, never server-side direct.
const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:80/api/v1';

// ── Types ──────────────────────────────────────────────────────────────────────

export interface VerifyIdRequest {
  national_id: string;
}

export interface VerifyIdResponse {
  session_id: string;
  eligible_ballots: string[];
  status: string;
}

export interface RequestOtpRequest {
  session_id: string;
}

export interface RequestOtpResponse {
  expires_in_seconds: number;
  dev_otp?: string;
}

export interface VerifyOtpRequest {
  session_id: string;
  otp: string;
}

export interface VerifyOtpResponse {
  token: string;
  expires_at: string;
  eligible_ballots: string[];
}

export interface EligibilityResponse {
  ballots_cast: Record<string, boolean>;
  ballots_remaining: string[];
}

export interface Candidate {
  id: string;
  name: string;
  candidate_number: number;
  party_id: string;
  party_name: string;
  party_color: string;
}

export interface PartyOption {
  id: string;
  name: string;
  short_name: string;
  color_hex: string;
  logo_url?: string;
}

export interface ReferendumOption {
  value: string;   // 'agree' | 'disagree' | 'abstain'
  label: string;
}

export interface BallotResponse {
  ballot_type: string;
  candidates?: Candidate[];
  parties?: PartyOption[];
  referendum_options?: ReferendumOption[];
}

export interface CastVoteRequest {
  ballot_type: string;
  candidate_id?: string;
  party_id?: string;
  referendum_vote?: string;
  confirm: true;
}

export interface CastVoteResponse {
  receipt_hash: string;
  receipt_id: string;
}

export interface ReceiptVerifyResponse {
  verified: boolean;
  ballot_type: string;
  timestamp: string;
  receipt_hash: string;
}

// ── Helpers ────────────────────────────────────────────────────────────────────

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

async function authGet<T>(path: string): Promise<T> {
  const jwt = getVoterJWT();
  const res = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(jwt ? { Authorization: `Bearer ${jwt}` } : {}),
    },
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

async function authPost<T>(path: string, body: unknown): Promise<T> {
  const jwt = getVoterJWT();
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(jwt ? { Authorization: `Bearer ${jwt}` } : {}),
    },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

// ── Auth mutations ─────────────────────────────────────────────────────────────

export function useVerifyId() {
  return useMutation({
    mutationFn: (req: VerifyIdRequest) =>
      post<VerifyIdResponse>('/online-voting/auth/verify-id', req),
  });
}

export function useRequestOtp() {
  return useMutation({
    mutationFn: (req: RequestOtpRequest) =>
      post<RequestOtpResponse>('/online-voting/auth/request-otp', req),
  });
}

export function useVerifyOtp() {
  return useMutation({
    mutationFn: (req: VerifyOtpRequest) =>
      post<VerifyOtpResponse>('/online-voting/auth/verify-otp', req),
  });
}

// ── Protected queries ──────────────────────────────────────────────────────────

export function useEligibility(enabled = true) {
  return useQuery({
    queryKey: ['voting', 'eligibility'],
    queryFn: () => authGet<EligibilityResponse>('/online-voting/eligibility'),
    enabled,
    staleTime: 0, // always refetch — changes after each cast
  });
}

export function useBallot(ballotType: string, enabled = true) {
  return useQuery({
    queryKey: ['voting', 'ballot', ballotType],
    queryFn: () => authGet<BallotResponse>(`/online-voting/ballot/${ballotType}`),
    enabled: enabled && ballotType.length > 0,
    staleTime: 5 * 60_000, // ballot options rarely change
  });
}

// ── Cast mutation ──────────────────────────────────────────────────────────────

export function useCastVote() {
  return useMutation({
    mutationFn: (req: CastVoteRequest) =>
      authPost<CastVoteResponse>('/online-voting/cast', req),
  });
}

// ── Public receipt verification ────────────────────────────────────────────────

export async function fetchReceiptVerify(hash: string): Promise<ReceiptVerifyResponse> {
  const res = await fetch(`${API_BASE}/online-voting/receipt/${encodeURIComponent(hash)}`);
  if (!res.ok) {
    const err = (await res.json().catch(() => ({}))) as { error?: string };
    throw new Error(err.error ?? `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<ReceiptVerifyResponse>;
}

export function useReceiptVerify(hash: string) {
  return useQuery({
    queryKey: ['voting', 'receipt', hash],
    queryFn: () => fetchReceiptVerify(hash),
    enabled: hash.length > 0,
    staleTime: Infinity, // receipt is immutable
    retry: 1,
  });
}
