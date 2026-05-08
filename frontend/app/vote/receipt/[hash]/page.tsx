// Public receipt verification page — /vote/receipt/[hash]
// No authentication required. Server component with SSR verification attempt.

import Link from 'next/link';
import { fetchReceiptVerify } from '@/lib/voting/api';
import type { ReceiptVerifyResponse } from '@/lib/voting/api';

// Next.js 16: params is a Promise
export default async function ReceiptVerifyPage({
  params,
}: {
  params: Promise<{ hash: string }>;
}) {
  const { hash } = await params;

  let result: ReceiptVerifyResponse | null = null;
  let fetchError: string | null = null;

  try {
    result = await fetchReceiptVerify(hash);
  } catch (e) {
    fetchError = e instanceof Error ? e.message : 'Verification failed';
  }

  const BALLOT_LABELS: Record<string, string> = {
    CONSTITUENCY: 'Constituency Ballot',
    PARTY_LIST: 'Party-List Ballot',
    REFERENDUM: 'Referendum',
  };

  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-background px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        <Link
          href="/"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          ← Thailand Election 2026
        </Link>

        <h1 className="text-xl font-bold">Receipt Verification</h1>

        {/* Hash being verified */}
        <div className="rounded-lg border border-border bg-muted/30 px-4 py-3">
          <p className="text-xs text-muted-foreground mb-1">Receipt Hash</p>
          <code className="text-xs font-mono break-all">{hash}</code>
        </div>

        {/* Result */}
        {fetchError && (
          <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-4 text-sm text-destructive">
            <p className="font-semibold mb-1">Verification Failed</p>
            <p className="text-xs">{fetchError}</p>
          </div>
        )}

        {result && result.verified && (
          <div className="rounded-lg border border-green-500/30 bg-green-50 dark:bg-green-950/20 px-4 py-4 space-y-3">
            <div className="flex items-center gap-2">
              <span className="flex h-6 w-6 items-center justify-center rounded-full bg-green-500 text-white text-sm font-bold">
                ✓
              </span>
              <span className="text-sm font-semibold text-green-800 dark:text-green-300">
                Vote Verified
              </span>
            </div>
            <dl className="text-sm space-y-2">
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Ballot Type</dt>
                <dd className="font-medium">
                  {BALLOT_LABELS[result.ballot_type] ?? result.ballot_type}
                </dd>
              </div>
              <div className="flex justify-between">
                <dt className="text-muted-foreground">Timestamp</dt>
                <dd className="font-medium font-mono text-xs">
                  {new Date(result.timestamp).toLocaleString()}
                </dd>
              </div>
            </dl>
            <p className="text-xs text-muted-foreground">
              This receipt confirms your vote was recorded. It does not reveal
              your identity or your choice.
            </p>
          </div>
        )}

        {result && !result.verified && (
          <div className="rounded-lg border border-amber-400/30 bg-amber-50 dark:bg-amber-950/20 px-4 py-4 text-sm text-amber-800 dark:text-amber-300">
            <p className="font-semibold mb-1">Receipt Not Found</p>
            <p className="text-xs">
              This receipt hash could not be verified. It may be invalid or
              the vote may not yet be counted.
            </p>
          </div>
        )}

        <Link
          href="/"
          className="block text-center text-sm text-muted-foreground underline underline-offset-2 hover:text-foreground"
        >
          View live election results →
        </Link>
      </div>
    </div>
  );
}
