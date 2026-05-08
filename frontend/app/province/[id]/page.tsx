// Province detail page — /province/[id]
// Server component: fetches province summary SSR, then hydrates with real-time updates.

import { notFound } from 'next/navigation';
import Link from 'next/link';
import { fetchProvinceSummary, fetchProvinces } from '@/lib/api/client';
import { ProvinceResultsList } from './ProvinceResultsList';
import type { ProvinceSummary, Province } from '@/lib/types/election';

// Next.js 16: params is a Promise
export default async function ProvinceDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const numId = Number(id);

  if (!Number.isInteger(numId) || numId < 1) notFound();

  let summary: ProvinceSummary | null = null;
  try {
    summary = await fetchProvinceSummary(numId);
  } catch {
    notFound();
  }

  return (
    <main className="max-w-2xl mx-auto px-4 py-8">
      <Link
        href="/"
        className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground mb-6"
      >
        ← Back to Dashboard
      </Link>

      <h1 className="text-2xl font-bold mb-1">{summary.province_name}</h1>
      <p className="text-sm text-muted-foreground mb-6">
        {summary.ballot_type === 'CONSTITUENCY'
          ? 'Constituency Ballot Results'
          : 'Party-List Ballot Results'}
      </p>

      {/* Client component handles real-time updates */}
      <ProvinceResultsList
        provinceId={numId}
        initialSummary={summary}
      />
    </main>
  );
}

// Pre-render the first 10 provinces at build time; the rest are dynamic
export async function generateStaticParams(): Promise<{ id: string }[]> {
  try {
    const provinces: Province[] = await fetchProvinces();
    return provinces.slice(0, 10).map((p) => ({ id: String(p.id) }));
  } catch {
    return [];
  }
}
