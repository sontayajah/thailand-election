'use client';

// Party seat allocation summary (constituency + party-list combined).
// Renders the proportional seat calculator result from GET /election/party-list/calculate.

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { usePartyListCalc, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import type { SeatAllocation } from '@/lib/types/election';

const TOTAL_SEATS = 500;
const CONSTITUENCY_SEATS = 400;
const PARTY_LIST_SEATS = 100;

function SeatBar({ allocs }: { allocs: SeatAllocation[] }) {
  return (
    <div className="flex rounded-full overflow-hidden h-5 w-full" aria-hidden>
      {allocs.map((a) => {
        const pct = (a.total_seats / TOTAL_SEATS) * 100;
        if (pct < 0.5) return null;
        return (
          <div
            key={a.party_id}
            title={`${a.party_short_name}: ${a.total_seats} seats`}
            style={{ width: `${pct}%`, background: `#${a.party_color}` }}
            className="transition-all duration-500"
          />
        );
      })}
    </div>
  );
}

function AllocRow({ a }: { a: SeatAllocation }) {
  return (
    <tr className="border-b border-border last:border-0 text-xs">
      <td className="py-1.5 pr-2">
        <span className="flex items-center gap-1.5">
          <span
            className="inline-block h-2.5 w-2.5 rounded-full shrink-0"
            style={{ background: `#${a.party_color}` }}
          />
          {a.party_short_name}
        </span>
      </td>
      <td className="py-1.5 text-right tabular-nums">{a.base_seats}</td>
      <td className="py-1.5 text-right tabular-nums">{a.remainder_seats}</td>
      <td className="py-1.5 text-right tabular-nums font-semibold">{a.total_seats}</td>
    </tr>
  );
}

export function PartySeats() {
  const queryClient = useQueryClient();
  const { data, isPending, isError } = usePartyListCalc();

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.national, () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.partyListCalc() });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [queryClient]);

  if (isPending) return <SkeletonCard rows={5} />;
  if (isError || !data) {
    return (
      <div className="rounded-lg border border-border p-4 text-sm text-muted-foreground">
        Failed to load seat allocation.
      </div>
    );
  }

  const sorted = [...data.allocations].sort((a, b) => b.total_seats - a.total_seats);

  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-baseline justify-between mb-3">
        <h2 className="text-sm font-semibold">Party-List Seat Allocation</h2>
        <span className="text-xs text-muted-foreground">
          {PARTY_LIST_SEATS} seats · {data.votes_per_seat.toLocaleString(undefined, { maximumFractionDigits: 0 })} votes/seat
        </span>
      </div>

      {/* Proportional bar */}
      <SeatBar allocs={sorted} />

      {/* Seat boundary labels */}
      <div className="flex justify-between text-[10px] text-muted-foreground mt-1 mb-3">
        <span>0</span>
        <span>{TOTAL_SEATS / 2}</span>
        <span>{TOTAL_SEATS}</span>
      </div>

      {/* Detail table */}
      <table className="w-full">
        <thead>
          <tr className="text-[10px] text-muted-foreground border-b border-border">
            <th className="pb-1 text-left font-medium">Party</th>
            <th className="pb-1 text-right font-medium">Base</th>
            <th className="pb-1 text-right font-medium">Remainder</th>
            <th className="pb-1 text-right font-medium">Total</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((a) => (
            <AllocRow key={a.party_id} a={a} />
          ))}
        </tbody>
      </table>

      <p className="text-[10px] text-muted-foreground mt-2">
        Proportional formula (§1.3.3) · Updated {new Date(data.updated_at).toLocaleTimeString()}
      </p>
    </section>
  );
}
