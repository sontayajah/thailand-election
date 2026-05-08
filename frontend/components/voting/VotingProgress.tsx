'use client';

// Shows which of the 3 ballots have been cast and which remain.

const BALLOT_LABELS: Record<string, string> = {
  CONSTITUENCY: 'Constituency',
  PARTY_LIST: 'Party List',
  REFERENDUM: 'Referendum',
};

const BALLOT_ORDER = ['CONSTITUENCY', 'PARTY_LIST', 'REFERENDUM'];

interface Props {
  ballotsCast: Record<string, boolean>;
  ballotsRemaining: string[];
  activeBallotType?: string;
}

export function VotingProgress({ ballotsCast, ballotsRemaining, activeBallotType }: Props) {
  return (
    <div className="flex flex-col gap-2">
      {BALLOT_ORDER.map((type) => {
        const cast = ballotsCast[type] === true;
        const remaining = ballotsRemaining.includes(type);
        const active = activeBallotType === type;

        return (
          <div
            key={type}
            className={[
              'flex items-center gap-3 rounded-md px-3 py-2 text-sm',
              active ? 'bg-primary/10 font-semibold' : 'bg-muted/50',
            ].join(' ')}
            aria-current={active ? 'step' : undefined}
          >
            {/* Status icon */}
            <span
              className={[
                'flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-xs font-bold',
                cast
                  ? 'bg-green-500 text-white'
                  : active
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-border text-muted-foreground',
              ].join(' ')}
              aria-hidden
            >
              {cast ? '✓' : remaining ? '○' : '–'}
            </span>

            <span className={cast ? 'line-through text-muted-foreground' : ''}>
              {BALLOT_LABELS[type] ?? type}
            </span>

            {cast && (
              <span className="ml-auto text-xs text-green-600 dark:text-green-400">
                Voted
              </span>
            )}
            {active && !cast && (
              <span className="ml-auto text-xs text-primary">Current</span>
            )}
          </div>
        );
      })}
    </div>
  );
}
