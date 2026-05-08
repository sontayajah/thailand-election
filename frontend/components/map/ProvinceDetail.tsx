'use client';

// Province detail panel — shown when a user clicks a province tile.
// Fetches the province summary (constituency or party-list results).

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useProvinceSummary, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { useUIStore } from '@/lib/store/ui';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import type { ProvinceResultEntry } from '@/lib/types/election';

function ResultRow({ r }: { r: ProvinceResultEntry }) {
  return (
    <li className="flex items-center gap-2 py-1.5 border-b border-border last:border-0">
      <span
        className="inline-block h-2.5 w-2.5 rounded-full shrink-0"
        style={{ background: `#${r.party_color}` }}
        aria-hidden
      />
      <span className="text-xs flex-1 truncate">
        {r.candidate_name ? `${r.candidate_name} (${r.party_short_name})` : r.party_name}
      </span>
      <span className="text-xs tabular-nums font-semibold">
        {r.total_votes.toLocaleString()}
      </span>
    </li>
  );
}

export function ProvinceDetail() {
  const { selectedProvinceId, setSelectedProvinceId } = useUIStore();
  const queryClient = useQueryClient();

  const { data, isPending, isError } = useProvinceSummary(selectedProvinceId ?? 0);

  useEffect(() => {
    if (!selectedProvinceId) return;
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.province(selectedProvinceId), () => {
      void queryClient.invalidateQueries({
        queryKey: queryKeys.provinceSummary(selectedProvinceId),
      });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [selectedProvinceId, queryClient]);

  if (!selectedProvinceId) return null;

  return (
    <aside className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="text-sm font-semibold">
            {data?.province_name ?? 'Province Detail'}
          </h3>
          {data && (
            <p className="text-xs text-muted-foreground">
              {data.ballot_type === 'CONSTITUENCY'
                ? 'Constituency ballot'
                : 'Party-list ballot'}
            </p>
          )}
        </div>
        <button
          onClick={() => setSelectedProvinceId(null)}
          className="text-muted-foreground hover:text-foreground text-lg leading-none"
          aria-label="Close"
        >
          ×
        </button>
      </div>

      {isPending && <SkeletonCard rows={5} />}

      {isError && (
        <p className="text-xs text-muted-foreground">
          Failed to load province data.
        </p>
      )}

      {data && (
        <ul>
          {data.results.map((r, i) => (
            <ResultRow key={`${r.party_id}-${i}`} r={r} />
          ))}
        </ul>
      )}
    </aside>
  );
}
