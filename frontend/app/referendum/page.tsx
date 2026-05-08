// Referendum dashboard page — /referendum
// SSR initial data + client-side real-time chart.

import Link from 'next/link';
import { fetchReferendumSummary } from '@/lib/api/client';
import { ReferendumPanel } from '@/components/referendum/ReferendumPanel';
import { ReferendumProvinceTable } from './ReferendumProvinceTable';
import type { ReferendumSummary } from '@/lib/types/election';

async function getInitialData(): Promise<ReferendumSummary | null> {
  try {
    return await fetchReferendumSummary();
  } catch {
    return null;
  }
}

export const metadata = {
  title: 'Referendum Results — Thailand Election 2026',
};

export default async function ReferendumPage() {
  const summary = await getInitialData();

  return (
    <main className="max-w-4xl mx-auto px-4 py-8">
      <Link
        href="/"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-6"
      >
        ← Back to Dashboard
      </Link>

      <h1 className="text-2xl font-bold mb-1">Referendum Results</h1>
      <p className="text-sm text-muted-foreground mb-6">
        Constitutional amendment referendum — Approve / Disagree / Abstain
      </p>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* National summary chart */}
        <ReferendumPanel />

        {/* Province-by-province agree % */}
        <ReferendumProvinceTable initialData={summary ?? undefined} />
      </div>
    </main>
  );
}
