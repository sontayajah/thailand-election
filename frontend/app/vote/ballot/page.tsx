'use client';

// Step 3: Ballot dashboard — shows the 3 ballot types and their cast status.
// Redirects to /vote if not authenticated.

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { isAuthenticated } from '@/lib/voting/session';
import { useEligibility } from '@/lib/voting/api';
import { VotingProgress } from '@/components/voting/VotingProgress';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import Link from 'next/link';

const BALLOT_LABELS: Record<string, string> = {
  CONSTITUENCY: 'Constituency Ballot',
  PARTY_LIST: 'Party-List Ballot',
  REFERENDUM: 'Referendum',
};

const BALLOT_DESCRIPTIONS: Record<string, string> = {
  CONSTITUENCY: 'Vote for your local candidate (MP)',
  PARTY_LIST: 'Vote for a political party (proportional seats)',
  REFERENDUM: 'Approve or reject the constitutional amendment',
};

export default function BallotDashboardPage() {
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace('/vote');
    }
  }, [router]);

  const { data, isPending, isError } = useEligibility(isAuthenticated());

  // All ballots cast → go to receipt
  useEffect(() => {
    if (data && data.ballots_remaining.length === 0) {
      router.replace('/vote/receipt');
    }
  }, [data, router]);

  if (isPending) return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md">
        <SkeletonCard rows={3} />
      </div>
    </div>
  );

  if (isError || !data) {
    return (
      <div className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-md rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
          Session expired or an error occurred.{' '}
          <Link href="/vote" className="underline">
            Start over
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Step indicator */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px] font-bold">1</span>
          <span>Identity</span>
          <span className="mx-1">→</span>
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-[10px] font-bold">2</span>
          <span>OTP</span>
          <span className="mx-1">→</span>
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground text-[10px] font-bold">3</span>
          <span className="font-medium text-foreground">Ballot</span>
        </div>

        <div>
          <h1 className="text-xl font-bold">Your Ballots</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            You have {data.ballots_remaining.length} ballot
            {data.ballots_remaining.length !== 1 ? 's' : ''} remaining.
            Complete each one below.
          </p>
        </div>

        {/* Progress tracker */}
        <VotingProgress
          ballotsCast={data.ballots_cast}
          ballotsRemaining={data.ballots_remaining}
        />

        {/* Ballot type cards */}
        <div className="flex flex-col gap-3">
          {['CONSTITUENCY', 'PARTY_LIST', 'REFERENDUM'].map((type) => {
            const isCast = data.ballots_cast[type] === true;
            const isRemaining = data.ballots_remaining.includes(type);
            return (
              <div
                key={type}
                className={[
                  'rounded-xl border p-4',
                  isCast
                    ? 'border-green-500/30 bg-green-50 dark:bg-green-950/20 opacity-60'
                    : 'border-border bg-card',
                ].join(' ')}
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-semibold">{BALLOT_LABELS[type]}</p>
                    <p className="text-xs text-muted-foreground mt-0.5">
                      {BALLOT_DESCRIPTIONS[type]}
                    </p>
                  </div>
                  {isCast ? (
                    <span className="shrink-0 rounded-full bg-green-500 px-2 py-0.5 text-xs text-white font-medium">
                      Voted ✓
                    </span>
                  ) : isRemaining ? (
                    <Link
                      href={`/vote/ballot/${type}`}
                      className="shrink-0 rounded-full bg-primary px-3 py-1.5 text-xs font-semibold text-primary-foreground hover:opacity-90 transition-opacity"
                    >
                      Vote Now
                    </Link>
                  ) : null}
                </div>
              </div>
            );
          })}
        </div>

        {/* View receipts early */}
        {Object.values(data.ballots_cast).some(Boolean) && (
          <Link
            href="/vote/receipt"
            className="block text-center text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
          >
            View my receipts so far →
          </Link>
        )}
      </div>
    </div>
  );
}
