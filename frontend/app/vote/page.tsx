'use client';

// Step 1: National ID verification → /vote/otp

import { useRouter } from 'next/navigation';
import { useVerifyId } from '@/lib/voting/api';
import { setSessionId } from '@/lib/voting/session';
import { NationalIdForm } from '@/components/voting/NationalIdForm';

export default function VoteEntryPage() {
  const router = useRouter();
  const { mutate, isPending, error } = useVerifyId();

  function handleSubmit(nationalId: string) {
    mutate(
      { national_id: nationalId },
      {
        onSuccess(data) {
          setSessionId(data.session_id);
          router.push('/vote/otp');
        },
      }
    );
  }

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Step indicator */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground text-[10px] font-bold">
            1
          </span>
          <span className="font-medium text-foreground">Identity Verification</span>
          <span className="mx-1">→</span>
          <span>OTP</span>
          <span className="mx-1">→</span>
          <span>Ballot</span>
        </div>

        <div>
          <h1 className="text-xl font-bold">Verify Your Identity</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Enter your Thai national ID to receive a one-time password via SMS.
          </p>
        </div>

        <NationalIdForm
          onSubmit={handleSubmit}
          isPending={isPending}
          error={error?.message}
        />

        {/* Privacy notice */}
        <div className="rounded-lg border border-border bg-muted/30 p-3 text-xs text-muted-foreground space-y-1">
          <p className="font-medium text-foreground">Your privacy is protected</p>
          <p>
            Your national ID is used only to confirm eligibility. It is hashed
            immediately and never stored in plaintext. Your vote choices are
            completely anonymous.
          </p>
        </div>
      </div>
    </div>
  );
}
