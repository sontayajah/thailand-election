'use client';

// Post-vote receipt display — shows receipt hash with a copy button.

import { useState } from 'react';
import type { CastReceipt } from '@/lib/voting/session';

const BALLOT_LABELS: Record<string, string> = {
  CONSTITUENCY: 'Constituency Ballot',
  PARTY_LIST: 'Party-List Ballot',
  REFERENDUM: 'Referendum Ballot',
};

interface Props {
  receipt: CastReceipt;
}

export function ReceiptCard({ receipt }: Props) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(receipt.receipt_hash);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // clipboard access denied — silently ignore
    }
  }

  return (
    <div className="rounded-xl border border-border bg-card p-4 flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <span className="text-sm font-semibold">
          {BALLOT_LABELS[receipt.ballot_type] ?? receipt.ballot_type}
        </span>
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-green-500 text-[10px] text-white font-bold">
          ✓
        </span>
      </div>

      {/* Hash display */}
      <div className="rounded-md bg-muted px-3 py-2 flex items-center gap-2">
        <code className="flex-1 text-xs font-mono break-all text-muted-foreground select-all">
          {receipt.receipt_hash}
        </code>
        <button
          onClick={handleCopy}
          aria-label="Copy receipt hash"
          className="shrink-0 rounded-md px-2 py-1 text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-border transition-colors"
        >
          {copied ? '✓ Copied' : 'Copy'}
        </button>
      </div>

      {/* Verify link */}
      <a
        href={`/vote/receipt/${encodeURIComponent(receipt.receipt_hash)}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-xs text-primary underline underline-offset-2 hover:opacity-80 self-start"
      >
        Verify this receipt →
      </a>
    </div>
  );
}
