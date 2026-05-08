'use client';

// Final cast confirmation dialog — uses shadcn/ui AlertDialog.

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ballotType: string;
  choiceLabel: string;
  onConfirm: () => void;
  isPending: boolean;
}

const BALLOT_NAMES: Record<string, string> = {
  CONSTITUENCY: 'Constituency',
  PARTY_LIST: 'Party List',
  REFERENDUM: 'Referendum',
};

export function ConfirmDialog({
  open,
  onOpenChange,
  ballotType,
  choiceLabel,
  onConfirm,
  isPending,
}: Props) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Confirm Your Vote</AlertDialogTitle>
          <AlertDialogDescription className="sr-only">
            Confirm your vote for {BALLOT_NAMES[ballotType] ?? ballotType}
          </AlertDialogDescription>
          <div className="space-y-3 text-sm text-muted-foreground">
            <p>
              You are about to cast your{' '}
              <span className="font-semibold text-foreground">
                {BALLOT_NAMES[ballotType] ?? ballotType}
              </span>{' '}
              ballot for:
            </p>
            <div className="rounded-lg border border-border bg-muted/30 px-4 py-3 text-center font-semibold text-foreground">
              {choiceLabel}
            </div>
            <p className="text-xs">
              <strong>This action is final and cannot be undone.</strong> Your vote
              is anonymous — it cannot be traced back to you after submission.
            </p>
          </div>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Go Back</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            disabled={isPending}
            className="bg-primary text-primary-foreground"
          >
            {isPending ? 'Submitting…' : 'Cast Vote'}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
