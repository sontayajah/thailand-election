'use client';

// Client component — handles real-time updates to province results.

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useProvinceSummary, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import type { ProvinceSummary } from '@/lib/types/election';

interface Props {
  provinceId: number;
  initialSummary: ProvinceSummary;
}

export function ProvinceResultsList({ provinceId, initialSummary }: Props) {
  const queryClient = useQueryClient();

  const { data } = useProvinceSummary(provinceId, { initialData: initialSummary });

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.province(provinceId), () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.provinceSummary(provinceId),
      });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [provinceId, queryClient]);

  if (!data) return <SkeletonCard rows={5} />;

  const sorted = [...data.results].sort((a, b) => b.total_votes - a.total_votes);
  const maxVotes = sorted[0]?.total_votes ?? 1;

  return (
    <div className="rounded-lg border border-border bg-card overflow-hidden">
      <ul>
        {sorted.map((r, i) => {
          const pct = (r.total_votes / maxVotes) * 100;
          return (
            <li
              key={`${r.party_id}-${i}`}
              className="flex flex-col gap-1 px-4 py-3 border-b border-border last:border-0"
            >
              <div className="flex items-center gap-2">
                <span
                  className="inline-block h-3 w-3 rounded-full shrink-0"
                  style={{ background: `#${r.party_color}` }}
                  aria-hidden
                />
                <span className="text-sm font-medium flex-1 truncate">
                  {r.candidate_name
                    ? `${r.candidate_name}`
                    : r.party_name}
                </span>
                {r.candidate_name && (
                  <span className="text-xs text-muted-foreground">
                    {r.party_short_name}
                  </span>
                )}
                <span className="text-sm font-bold tabular-nums">
                  {r.total_votes.toLocaleString()}
                </span>
              </div>
              {/* Vote bar */}
              <div className="ml-5 h-1.5 rounded-full bg-muted overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-500"
                  style={{ width: `${pct}%`, background: `#${r.party_color}` }}
                />
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
