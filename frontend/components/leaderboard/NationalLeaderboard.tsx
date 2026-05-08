'use client';

// National party leaderboard — shows party seats and votes ranked by total seats.
// Updates in real-time via Centrifugo (national channel).

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useNationalSummary, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import type { NationalSummary, PartyNationalResult } from '@/lib/types/election';

function formatVotes(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toLocaleString();
}

function PartyRow({ p, maxSeats }: { p: PartyNationalResult; maxSeats: number }) {
  const pct = maxSeats > 0 ? (p.total_seats / maxSeats) * 100 : 0;

  return (
    <li className="flex flex-col gap-1 py-2 border-b border-border last:border-0">
      <div className="flex items-center gap-2">
        {/* Colour swatch */}
        <span
          className="inline-block h-3 w-3 rounded-full shrink-0"
          style={{ background: `#${p.party_color}` }}
          aria-hidden
        />
        <span className="text-sm font-medium truncate flex-1">{p.party_name}</span>
        <span className="text-sm font-bold tabular-nums">{p.total_seats}</span>
        <span className="text-xs text-muted-foreground w-12 text-right tabular-nums">
          seats
        </span>
      </div>
      {/* Seat bar */}
      <div className="ml-5 h-1.5 rounded-full bg-muted overflow-hidden">
        <div
          className="h-full rounded-full transition-all duration-500"
          style={{ width: `${pct}%`, background: `#${p.party_color}` }}
        />
      </div>
      <div className="ml-5 flex gap-4 text-xs text-muted-foreground">
        <span>Constituency: {p.constituency_seats}</span>
        <span>Party list: {p.party_list_seats}</span>
        <span className="ml-auto">{formatVotes(p.party_list_votes)} votes</span>
      </div>
    </li>
  );
}

interface Props {
  initialData?: NationalSummary;
}

export function NationalLeaderboard({ initialData }: Props) {
  const queryClient = useQueryClient();

  const { data, isPending, isError } = useNationalSummary({ initialData });

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.national, () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.nationalSummary() });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [queryClient]);

  if (isPending) return <SkeletonCard rows={8} />;
  if (isError || !data) {
    return (
      <div className="rounded-lg border border-border p-4 text-sm text-muted-foreground">
        Failed to load national results.
      </div>
    );
  }

  const sorted = [...data.parties].sort((a, b) => b.total_seats - a.total_seats);
  const maxSeats = sorted[0]?.total_seats ?? 1;

  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-baseline justify-between mb-3">
        <h2 className="text-sm font-semibold">National Results</h2>
        <span className="text-xs text-muted-foreground">
          {data.total_votes_cast.toLocaleString()} votes cast
        </span>
      </div>
      <ul className="divide-y divide-border">
        {sorted.map((p) => (
          <PartyRow key={p.party_id} p={p} maxSeats={maxSeats} />
        ))}
      </ul>
      <p className="text-[10px] text-muted-foreground mt-2">
        Updated {new Date(data.updated_at).toLocaleTimeString()}
      </p>
    </section>
  );
}
