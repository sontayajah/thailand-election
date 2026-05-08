// TanStack Query v5 API client for the Thailand Election reporting API.
// All hooks fetch from /api/v1/* via the NEXT_PUBLIC_API_URL env var.

import {
  useQuery,
  type UseQueryOptions,
} from '@tanstack/react-query';
import type {
  Province,
  Party,
  NationalSummary,
  ProvinceSummary,
  PartyListCalculation,
  ReferendumSummary,
} from '@/lib/types/election';

// ── Base fetch ─────────────────────────────────────────────────────────────────

// SSR (server components / route handlers) calls the Go API directly over the
// internal Docker network — no nginx/Kong hop, no CORS check.
// Browser (client components / TanStack Query hooks) goes through nginx → Kong.
const API_BASE =
  typeof window === 'undefined'
    ? (process.env.API_INTERNAL_URL ?? 'http://api:8080/api/v1')
    : (process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:80/api/v1');

async function apiFetch<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    next: { revalidate: 0 }, // always fresh (Centrifugo pushes updates)
  });
  if (!res.ok) {
    throw new Error(`API ${path} → ${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

// ── Query keys ─────────────────────────────────────────────────────────────────

export const queryKeys = {
  provinces: () => ['provinces'] as const,
  parties: () => ['parties'] as const,
  nationalSummary: () => ['election', 'national', 'summary'] as const,
  provinceSummary: (id: number) => ['election', 'province', id, 'summary'] as const,
  partyListCalc: () => ['election', 'party-list', 'calculate'] as const,
  referendum: () => ['election', 'referendum', 'summary'] as const,
} as const;

// ── Fetchers (usable server-side too) ─────────────────────────────────────────

export async function fetchProvinces(): Promise<Province[]> {
  return apiFetch<Province[]>('/provinces');
}

export async function fetchParties(): Promise<Party[]> {
  return apiFetch<Party[]>('/parties');
}

export async function fetchNationalSummary(): Promise<NationalSummary> {
  return apiFetch<NationalSummary>('/election/national/summary');
}

export async function fetchProvinceSummary(id: number): Promise<ProvinceSummary> {
  return apiFetch<ProvinceSummary>(`/election/provinces/${id}/summary`);
}

export async function fetchPartyListCalc(): Promise<PartyListCalculation> {
  return apiFetch<PartyListCalculation>('/election/party-list/calculate');
}

export async function fetchReferendumSummary(): Promise<ReferendumSummary> {
  return apiFetch<ReferendumSummary>('/election/referendum/summary');
}

// ── Hooks ──────────────────────────────────────────────────────────────────────

type Opts<T> = Omit<UseQueryOptions<T>, 'queryKey' | 'queryFn'>;

const REFETCH_INTERVAL = 10_000; // 10 s fallback when WebSocket is unavailable

export function useProvinces(opts?: Opts<Province[]>) {
  return useQuery({
    queryKey: queryKeys.provinces(),
    queryFn: fetchProvinces,
    staleTime: 5 * 60_000, // provinces rarely change mid-election
    ...opts,
  });
}

export function useParties(opts?: Opts<Party[]>) {
  return useQuery({
    queryKey: queryKeys.parties(),
    queryFn: fetchParties,
    staleTime: 5 * 60_000,
    ...opts,
  });
}

export function useNationalSummary(opts?: Opts<NationalSummary>) {
  return useQuery({
    queryKey: queryKeys.nationalSummary(),
    queryFn: fetchNationalSummary,
    refetchInterval: REFETCH_INTERVAL,
    ...opts,
  });
}

export function useProvinceSummary(id: number, opts?: Opts<ProvinceSummary>) {
  return useQuery({
    queryKey: queryKeys.provinceSummary(id),
    queryFn: () => fetchProvinceSummary(id),
    enabled: id > 0,
    refetchInterval: REFETCH_INTERVAL,
    ...opts,
  });
}

export function usePartyListCalc(opts?: Opts<PartyListCalculation>) {
  return useQuery({
    queryKey: queryKeys.partyListCalc(),
    queryFn: fetchPartyListCalc,
    refetchInterval: REFETCH_INTERVAL,
    ...opts,
  });
}

export function useReferendumSummary(opts?: Opts<ReferendumSummary>) {
  return useQuery({
    queryKey: queryKeys.referendum(),
    queryFn: fetchReferendumSummary,
    refetchInterval: REFETCH_INTERVAL,
    ...opts,
  });
}

// ── Centrifugo token fetcher ───────────────────────────────────────────────────

export async function fetchCentrifugoToken(): Promise<string> {
  const res = await fetch(`${API_BASE}/realtime/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  });
  if (!res.ok) throw new Error('Failed to obtain realtime token');
  const body = (await res.json()) as { token: string };
  return body.token;
}
