'use client';

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { useReferendumSummary, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import type { ReferendumSummary } from '@/lib/types/election';

interface Props {
  initialData?: ReferendumSummary;
}

export function ReferendumProvinceTable({ initialData }: Props) {
  const queryClient = useQueryClient();

  const { data, isPending } = useReferendumSummary({ initialData });

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.referendum, () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.referendum() });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [queryClient]);

  if (isPending) return <SkeletonCard rows={10} />;
  if (!data) return null;

  const sorted = [...data.by_province].sort((a, b) => b.agree_pct - a.agree_pct);

  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold mb-3">By Province — Agree %</h2>
      <div className="overflow-y-auto max-h-[480px]">
        <table className="w-full text-xs">
          <thead className="sticky top-0 bg-card">
            <tr className="border-b border-border text-muted-foreground">
              <th className="pb-2 text-left font-medium">Province</th>
              <th className="pb-2 text-right font-medium">Agree %</th>
              <th className="pb-2 text-right font-medium">Votes</th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((p) => (
              <tr
                key={p.province_id}
                className="border-b border-border last:border-0"
              >
                <td className="py-1.5">{p.province_name}</td>
                <td className="py-1.5 text-right tabular-nums">
                  <span
                    className={
                      p.agree_pct > 50
                        ? 'text-green-600 dark:text-green-400 font-semibold'
                        : 'text-red-600 dark:text-red-400'
                    }
                  >
                    {p.agree_pct.toFixed(1)}%
                  </span>
                </td>
                <td className="py-1.5 text-right tabular-nums text-muted-foreground">
                  {p.total_votes.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
