// Main election dashboard — server-side rendered initial data, client-side real-time updates.
// F-01: Thailand province map (tile layout)
// F-02: National party leaderboard
// F-03: Province detail on click
// F-04: Party-list seat allocation
// F-05: Referendum summary panel

import { Suspense } from 'react';
import { fetchNationalSummary, fetchProvinces } from '@/lib/api/client';
import { ThailandMap } from '@/components/map/ThailandMap';
import { ProvinceDetail } from '@/components/map/ProvinceDetail';
import { NationalLeaderboard } from '@/components/leaderboard/NationalLeaderboard';
import { PartySeats } from '@/components/leaderboard/PartySeats';
import { ReferendumPanel } from '@/components/referendum/ReferendumPanel';
import { ConnectionStatus } from '@/components/shared/ConnectionStatus';
import { DarkModeToggle } from '@/components/shared/DarkModeToggle';
import { SkeletonCard, SkeletonBar } from '@/components/shared/SkeletonCard';
import { ErrorBoundary } from '@/components/shared/ErrorBoundary';
import type { NationalSummary, Province } from '@/lib/types/election';

// SSR prefetch — run on the server for every request
async function getInitialData(): Promise<{
  summary: NationalSummary | null;
  provinces: Province[] | null;
}> {
  try {
    const [summary, provinces] = await Promise.all([
      fetchNationalSummary(),
      fetchProvinces(),
    ]);
    return { summary, provinces };
  } catch {
    return { summary: null, provinces: null };
  }
}

export default async function DashboardPage() {
  const { summary, provinces } = await getInitialData();

  return (
    <>
      {/* Real-time connection banner — renders nothing when connected */}
      <ConnectionStatus />

      {/* Site header */}
      <header className="sticky top-0 z-40 bg-background/80 backdrop-blur border-b border-border">
        <div className="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
          <div>
            <h1 className="text-base font-bold leading-tight">
              🗳 Thailand General Election 2026
            </h1>
            <p className="text-xs text-muted-foreground">Live Results Dashboard</p>
          </div>
          <div className="flex items-center gap-2">
            <a
              href="/vote"
              className="text-xs px-3 py-1.5 rounded-full bg-primary text-primary-foreground font-medium hover:opacity-90 transition-opacity"
            >
              Cast Your Vote
            </a>
            <DarkModeToggle />
          </div>
        </div>
      </header>

      {/* Main layout: map + sidebar */}
      <main className="flex-1 max-w-7xl mx-auto w-full px-4 py-6 grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_340px] gap-6">
        {/* Left column: map + referendum */}
        <div className="flex flex-col gap-6">
          <ErrorBoundary>
            <Suspense fallback={<SkeletonCard rows={6} className="h-64" />}>
              <ThailandMap
                initialSummary={summary ?? undefined}
                initialProvinces={provinces ?? undefined}
              />
            </Suspense>
          </ErrorBoundary>

          {/* Province detail panel (appears below map on mobile, inline on desktop) */}
          <ErrorBoundary>
            <ProvinceDetail />
          </ErrorBoundary>

          <ErrorBoundary>
            <Suspense fallback={<SkeletonBar />}>
              <ReferendumPanel />
            </Suspense>
          </ErrorBoundary>
        </div>

        {/* Right sidebar: leaderboard + seat allocation */}
        <div className="flex flex-col gap-6">
          <ErrorBoundary>
            <Suspense fallback={<SkeletonCard rows={8} />}>
              <NationalLeaderboard initialData={summary ?? undefined} />
            </Suspense>
          </ErrorBoundary>

          <ErrorBoundary>
            <Suspense fallback={<SkeletonCard rows={6} />}>
              <PartySeats />
            </Suspense>
          </ErrorBoundary>
        </div>
      </main>

      <footer className="border-t border-border py-4 text-center text-xs text-muted-foreground">
        Data updates in real-time via WebSocket · Counts are unofficial
      </footer>
    </>
  );
}
