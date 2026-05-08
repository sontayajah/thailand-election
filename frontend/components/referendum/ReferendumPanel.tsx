'use client';

// Referendum results panel — national breakdown (Agree / Disagree / Abstain).
// Uses Recharts bar chart for the visual; also renders a tabular summary.

import { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import { useReferendumSummary, queryKeys } from '@/lib/api/client';
import { subscribeChannel, Channels } from '@/lib/ws/centrifuge';
import { SkeletonBar, SkeletonCard } from '@/components/shared/SkeletonCard';
import type { ReferendumSummary } from '@/lib/types/election';

const CHART_DATA = (s: ReferendumSummary['national']) => [
  {
    label: 'Agree',
    votes: s.agree_votes,
    pct: s.agree_pct,
    color: '#22c55e',
  },
  {
    label: 'Disagree',
    votes: s.disagree_votes,
    pct: s.disagree_pct,
    color: '#ef4444',
  },
  {
    label: 'Abstain',
    votes: s.abstain_votes,
    pct:
      s.total_votes > 0
        ? ((s.abstain_votes / s.total_votes) * 100)
        : 0,
    color: '#a3a3a3',
  },
];

function pct(n: number): string {
  return `${n.toFixed(2)}%`;
}

function fmt(n: number): string {
  return n.toLocaleString();
}

export function ReferendumPanel() {
  const queryClient = useQueryClient();

  const { data, isPending, isError } = useReferendumSummary();

  useEffect(() => {
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.referendum, () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.referendum() });
    }).then((fn) => {
      cleanup = fn;
    });
    return () => cleanup?.();
  }, [queryClient]);

  if (isPending) {
    return (
      <div className="rounded-lg border border-border bg-card p-4 space-y-3">
        <SkeletonCard rows={2} />
        <SkeletonBar />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="rounded-lg border border-border p-4 text-sm text-muted-foreground">
        Failed to load referendum results.
      </div>
    );
  }

  const national = data.national;
  const chartData = CHART_DATA(national);

  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-semibold mb-1">Referendum Results</h2>
      <p className="text-xs text-muted-foreground mb-4">
        Total votes: {fmt(national.total_votes)}
      </p>

      {/* Bar chart */}
      <ResponsiveContainer width="100%" height={160}>
        <BarChart data={chartData} margin={{ top: 4, right: 4, bottom: 4, left: -16 }}>
          <XAxis dataKey="label" tick={{ fontSize: 11 }} />
          <YAxis
            tickFormatter={(v: number) => `${(v / 1_000_000).toFixed(1)}M`}
            tick={{ fontSize: 10 }}
          />
          <Tooltip
            formatter={(value, _name, entry) => {
              const v = typeof value === 'number' ? value : 0;
              const p = (entry.payload as { pct?: number } | undefined)?.pct ?? 0;
              return [`${fmt(v)} (${pct(p)})`, String(_name)];
            }}
          />
          <Bar dataKey="votes" radius={[4, 4, 0, 0]}>
            {chartData.map((entry) => (
              <Cell key={entry.label} fill={entry.color} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>

      {/* Summary stats */}
      <dl className="grid grid-cols-3 gap-2 mt-4 text-center">
        {chartData.map((d) => (
          <div key={d.label} className="rounded-md bg-muted px-2 py-2">
            <dt className="text-[10px] text-muted-foreground">{d.label}</dt>
            <dd className="text-base font-bold tabular-nums" style={{ color: d.color }}>
              {pct(d.pct)}
            </dd>
            <dd className="text-[10px] text-muted-foreground tabular-nums">
              {fmt(d.votes)}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
