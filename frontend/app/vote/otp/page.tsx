'use client';

// Step 2: OTP entry → /vote/ballot
// OTP is requested automatically on mount if we have a session_id.
// Dev mode: backend returns dev_otp; we display it in a yellow banner.

import { useEffect, useState, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { useRequestOtp, useVerifyOtp } from '@/lib/voting/api';
import {
  getSessionId,
  setVoterJWT,
  setDevOTP,
  getDevOTP,
} from '@/lib/voting/session';
import { OtpInput } from '@/components/voting/OtpInput';

const OTP_TTL = 5 * 60; // 5 minutes in seconds

export default function OtpPage() {
  const router = useRouter();
  const [otp, setOtp] = useState('');
  const [secondsLeft, setSecondsLeft] = useState(0);
  const [devOtp, setDevOtpState] = useState<string | null>(null);

  const requestOtp = useRequestOtp();
  const verifyOtp = useVerifyOtp();

  const sessionId = getSessionId();

  // Redirect to /vote if session missing
  useEffect(() => {
    if (!sessionId) {
      router.replace('/vote');
    }
  }, [sessionId, router]);

  // Request OTP on mount
  const sendOtp = useCallback(() => {
    if (!sessionId) return;
    requestOtp.mutate(
      { session_id: sessionId },
      {
        onSuccess(data) {
          setSecondsLeft(data.expires_in_seconds ?? OTP_TTL);
          if (data.dev_otp) {
            setDevOTP(data.dev_otp);
            setDevOtpState(data.dev_otp);
          }
        },
      }
    );
  }, [sessionId, requestOtp]);

  useEffect(() => {
    sendOtp();
    setDevOtpState(getDevOTP());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Countdown timer
  useEffect(() => {
    if (secondsLeft <= 0) return;
    const id = setInterval(() => setSecondsLeft((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(id);
  }, [secondsLeft]);

  function handleVerify() {
    if (!sessionId || otp.length < 6) return;
    verifyOtp.mutate(
      { session_id: sessionId, otp },
      {
        onSuccess(data) {
          setVoterJWT(data.token);
          router.push('/vote/ballot');
        },
      }
    );
  }

  const minutes = Math.floor(secondsLeft / 60);
  const secs = secondsLeft % 60;
  const expired = secondsLeft === 0 && requestOtp.isSuccess;

  return (
    <div className="flex flex-1 items-center justify-center px-4 py-12">
      <div className="w-full max-w-md space-y-6">
        {/* Step indicator */}
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-muted text-muted-foreground text-[10px] font-bold">
            1
          </span>
          <span>Identity</span>
          <span className="mx-1">→</span>
          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground text-[10px] font-bold">
            2
          </span>
          <span className="font-medium text-foreground">OTP</span>
          <span className="mx-1">→</span>
          <span>Ballot</span>
        </div>

        <div>
          <h1 className="text-xl font-bold">Enter Your OTP</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            A 6-digit code was sent to your registered phone number.
          </p>
        </div>

        {/* DEV MODE banner */}
        {devOtp && (
          <div
            className="rounded-lg border border-amber-400 bg-amber-50 dark:bg-amber-950/30 px-4 py-3"
            role="status"
          >
            <p className="text-xs font-semibold text-amber-800 dark:text-amber-300 mb-1">
              ⚠ Development Mode — OTP is not sent via SMS
            </p>
            <p className="text-xs text-amber-700 dark:text-amber-400">
              Your OTP:{' '}
              <span className="font-mono font-bold tracking-widest text-amber-900 dark:text-amber-200">
                {devOtp}
              </span>
            </p>
          </div>
        )}

        {/* OTP input */}
        <div className="flex flex-col items-center gap-4">
          <OtpInput
            value={otp}
            onChange={setOtp}
            disabled={verifyOtp.isPending || expired}
          />

          {/* Countdown */}
          {requestOtp.isSuccess && (
            <p
              className={[
                'text-xs',
                expired
                  ? 'text-destructive font-semibold'
                  : secondsLeft <= 30
                  ? 'text-amber-600 dark:text-amber-400'
                  : 'text-muted-foreground',
              ].join(' ')}
            >
              {expired
                ? 'OTP expired — please request a new one'
                : `Expires in ${minutes}:${secs.toString().padStart(2, '0')}`}
            </p>
          )}

          {requestOtp.isPending && (
            <p className="text-xs text-muted-foreground animate-pulse">
              Sending OTP…
            </p>
          )}
        </div>

        {/* Verify error */}
        {verifyOtp.isError && (
          <div
            className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive"
            role="alert"
          >
            {verifyOtp.error?.message ?? 'Invalid OTP. Please try again.'}
          </div>
        )}

        {/* Actions */}
        <div className="flex flex-col gap-2">
          <button
            onClick={handleVerify}
            disabled={otp.length < 6 || verifyOtp.isPending || expired}
            className="rounded-full bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-60 hover:opacity-90 transition-opacity"
          >
            {verifyOtp.isPending ? 'Verifying…' : 'Verify OTP'}
          </button>

          <button
            onClick={sendOtp}
            disabled={requestOtp.isPending || (!expired && secondsLeft > 0)}
            className="rounded-full border border-border px-6 py-2 text-sm text-muted-foreground disabled:opacity-40 hover:text-foreground hover:bg-muted transition-colors"
          >
            {requestOtp.isPending ? 'Sending…' : 'Resend OTP'}
          </button>
        </div>
      </div>
    </div>
  );
}
