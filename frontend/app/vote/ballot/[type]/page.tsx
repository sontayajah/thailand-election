'use client';

// Individual ballot page — CONSTITUENCY / PARTY_LIST / REFERENDUM.
// Handles selection → confirmation dialog → cast → receipt.

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useBallot, useCastVote } from '@/lib/voting/api';
import { isAuthenticated, addReceipt } from '@/lib/voting/session';
import { BallotCard } from '@/components/voting/BallotCard';
import { ConfirmDialog } from '@/components/voting/ConfirmDialog';
import { VotingProgress } from '@/components/voting/VotingProgress';
import { useEligibility } from '@/lib/voting/api';
import { SkeletonCard } from '@/components/shared/SkeletonCard';
import Link from 'next/link';

// Next.js 16: params is a Promise
export default function BallotTypePage({
  params,
}: {
  params: Promise<{ type: string }>;
}) {
  const router = useRouter();
  const [ballotType, setBallotType] = useState('');
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedLabel, setSelectedLabel] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);

  // Resolve params async
  useEffect(() => {
    params.then((p) => setBallotType(p.type.toUpperCase()));
  }, [params]);

  // Auth guard
  useEffect(() => {
    if (!isAuthenticated()) {
      router.replace('/vote');
    }
  }, [router]);

  const { data: ballot, isPending: ballotPending } = useBallot(
    ballotType,
    isAuthenticated() && ballotType.length > 0
  );

  const { data: eligibility } = useEligibility(isAuthenticated());

  const castVote = useCastVote();

  function selectCandidate(id: string, label: string) {
    setSelectedId(id);
    setSelectedLabel(label);
  }

  function handleCast() {
    if (!selectedId || !ballotType) return;

    const req =
      ballotType === 'REFERENDUM'
        ? { ballot_type: ballotType, referendum_vote: selectedId, confirm: true as const }
        : ballotType === 'PARTY_LIST'
        ? { ballot_type: ballotType, party_id: selectedId, confirm: true as const }
        : { ballot_type: ballotType, candidate_id: selectedId, confirm: true as const };

    castVote.mutate(req, {
      onSuccess(data) {
        addReceipt({ ballot_type: ballotType, receipt_hash: data.receipt_hash });
        setConfirmOpen(false);
        router.push('/vote/ballot');
      },
    });
  }

  if (!ballotType || ballotPending) {
    return (
      <div className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <SkeletonCard rows={5} />
        </div>
      </div>
    );
  }

  if (!ballot) {
    return (
      <div className="flex flex-1 items-center justify-center px-4 py-12">
        <div className="w-full max-w-md text-sm text-muted-foreground">
          Unable to load ballot.{' '}
          <Link href="/vote/ballot" className="underline">
            Go back
          </Link>
        </div>
      </div>
    );
  }

  const REFERENDUM_OPTIONS = [
    { id: 'agree', label: 'Agree', sublabel: 'เห็นชอบ', color: '22c55e' },
    { id: 'disagree', label: 'Disagree', sublabel: 'ไม่เห็นชอบ', color: 'ef4444' },
    { id: 'abstain', label: 'Abstain', sublabel: 'งดออกเสียง', color: 'a3a3a3' },
  ];

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Back link */}
        <Link
          href="/vote/ballot"
          className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
        >
          ← Back to Ballots
        </Link>

        {/* Title */}
        <div>
          <h1 className="text-xl font-bold">
            {ballotType === 'CONSTITUENCY'
              ? 'Constituency Ballot'
              : ballotType === 'PARTY_LIST'
              ? 'Party-List Ballot'
              : 'Referendum'}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {ballotType === 'CONSTITUENCY'
              ? 'Select one candidate for your constituency.'
              : ballotType === 'PARTY_LIST'
              ? 'Select one party for the proportional seat allocation.'
              : 'Vote on the constitutional amendment.'}
          </p>
        </div>

        {/* Eligibility progress */}
        {eligibility && (
          <VotingProgress
            ballotsCast={eligibility.ballots_cast}
            ballotsRemaining={eligibility.ballots_remaining}
            activeBallotType={ballotType}
          />
        )}

        {/* Ballot options */}
        <div role="radiogroup" aria-label={`${ballotType} options`} className="flex flex-col gap-2">
          {/* CONSTITUENCY */}
          {ballotType === 'CONSTITUENCY' &&
            ballot.candidates?.map((c) => (
              <BallotCard
                key={c.id}
                isSelected={selectedId === c.id}
                onSelect={() =>
                  selectCandidate(c.id, `No.${c.candidate_number} ${c.name} (${c.party_name})`)
                }
                color={c.party_color}
                label={`No.${c.candidate_number} ${c.name}`}
                sublabel={c.party_name}
                badge={`#${c.candidate_number}`}
              />
            ))}

          {/* PARTY_LIST */}
          {ballotType === 'PARTY_LIST' &&
            ballot.parties?.map((p) => (
              <BallotCard
                key={p.id}
                isSelected={selectedId === p.id}
                onSelect={() => selectCandidate(p.id, p.name)}
                color={p.color_hex.replace('#', '')}
                label={p.name}
                sublabel={p.short_name}
              />
            ))}

          {/* REFERENDUM */}
          {ballotType === 'REFERENDUM' &&
            REFERENDUM_OPTIONS.map((opt) => (
              <BallotCard
                key={opt.id}
                isSelected={selectedId === opt.id}
                onSelect={() => selectCandidate(opt.id, `${opt.label} (${opt.sublabel})`)}
                color={opt.color}
                label={opt.label}
                sublabel={opt.sublabel}
              />
            ))}
        </div>

        {/* Cast error */}
        {castVote.isError && (
          <div
            className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {castVote.error?.message ?? 'Failed to cast vote. Please try again.'}
          </div>
        )}

        {/* Confirm button */}
        <button
          onClick={() => setConfirmOpen(true)}
          disabled={!selectedId}
          className="w-full rounded-full bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-40 hover:opacity-90 transition-opacity"
        >
          Review & Confirm
        </button>

        {/* Confirm dialog */}
        <ConfirmDialog
          open={confirmOpen}
          onOpenChange={setConfirmOpen}
          ballotType={ballotType}
          choiceLabel={selectedLabel}
          onConfirm={handleCast}
          isPending={castVote.isPending}
        />
      </div>
    </div>
  );
}
