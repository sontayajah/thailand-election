'use client';

// Post-vote receipt page — shows all 3 receipt hashes after casting.
// This is the final step of the voting flow.

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { isAuthenticated, getReceipts, clearSession } from '@/lib/voting/session';
import { ReceiptCard } from '@/components/voting/ReceiptCard';
import Link from 'next/link';
import type { CastReceipt } from '@/lib/voting/session';

export default function ReceiptPage() {
  const router = useRouter();
  const [receipts, setReceipts] = useState<CastReceipt[]>([]);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace('/vote');
      return;
    }
    setReceipts(getReceipts());
  }, [router]);

  function handleFinish() {
    clearSession();
    router.push('/');
  }

  const allThree = receipts.length >= 3;

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Success header */}
        <div className="flex flex-col items-center text-center gap-3">
          <div className="flex h-16 w-16 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30 text-3xl">
            🗳
          </div>
          <div>
            <h1 className="text-xl font-bold">
              {allThree ? 'Voting Complete!' : 'Receipts So Far'}
            </h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {allThree
                ? 'Thank you for participating in the 2026 General Election.'
                : `${receipts.length} of 3 ballots cast. Continue voting to finish.`}
            </p>
          </div>
        </div>

        {/* Receipt cards */}
        {receipts.length === 0 ? (
          <div className="rounded-lg border border-border bg-muted/30 p-4 text-sm text-muted-foreground text-center">
            No votes cast yet.{' '}
            <Link href="/vote/ballot" className="underline">
              Cast your ballots
            </Link>
          </div>
        ) : (
          <div className="flex flex-col gap-3">
            {receipts.map((r) => (
              <ReceiptCard key={r.ballot_type} receipt={r} />
            ))}
          </div>
        )}

        {/* Explanation */}
        <div className="rounded-lg border border-border bg-muted/30 px-4 py-3 text-xs text-muted-foreground space-y-1">
          <p className="font-semibold text-foreground">What is this receipt?</p>
          <p>
            Each receipt hash is a cryptographic fingerprint of your anonymous vote.
            You can use it to verify your vote was counted without revealing
            your identity or your choice.
          </p>
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-2">
          {!allThree && (
            <Link
              href="/vote/ballot"
              className="block text-center rounded-full bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground hover:opacity-90 transition-opacity"
            >
              Continue Voting
            </Link>
          )}

          <button
            onClick={handleFinish}
            className={[
              'rounded-full px-6 py-2.5 text-sm font-semibold transition-opacity',
              allThree
                ? 'bg-primary text-primary-foreground hover:opacity-90'
                : 'border border-border text-muted-foreground hover:text-foreground hover:bg-muted',
            ].join(' ')}
          >
            {allThree ? 'Done — View Live Results' : 'Finish & Exit'}
          </button>
        </div>

        <p className="text-xs text-center text-muted-foreground">
          Save your receipt hashes before closing this tab — they will not be
          shown again.
        </p>
      </div>
    </div>
  );
}
