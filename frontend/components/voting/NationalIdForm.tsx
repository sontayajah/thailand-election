'use client';

// 13-digit Thai National ID entry form with Luhn-style check digit validation.

import { useForm } from 'react-hook-form';

interface FormValues {
  national_id: string;
}

/** Thai national ID check digit algorithm. */
function isValidThaiId(id: string): boolean {
  if (!/^\d{13}$/.test(id)) return false;
  const digits = id.split('').map(Number);
  const sum = digits
    .slice(0, 12)
    .reduce((acc, d, i) => acc + d * (13 - i), 0);
  const check = (11 - (sum % 11)) % 10;
  return check === digits[12];
}

interface Props {
  onSubmit: (nationalId: string) => void;
  isPending: boolean;
  error?: string;
}

export function NationalIdForm({ onSubmit, isPending, error }: Props) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>();

  return (
    <form
      onSubmit={handleSubmit((v) => onSubmit(v.national_id))}
      className="flex flex-col gap-4"
      noValidate
    >
      <div className="flex flex-col gap-1.5">
        <label
          htmlFor="national_id"
          className="text-sm font-medium"
        >
          Thai National ID Number
        </label>
        <input
          id="national_id"
          type="tel"
          inputMode="numeric"
          maxLength={13}
          placeholder="0-0000-00000-00-0"
          autoComplete="off"
          className={[
            'rounded-md border px-3 py-2.5 text-base font-mono tracking-widest',
            'bg-background text-foreground placeholder:text-muted-foreground',
            'outline-none transition-colors',
            errors.national_id
              ? 'border-destructive ring-1 ring-destructive'
              : 'border-input focus:border-primary focus:ring-1 focus:ring-primary',
          ].join(' ')}
          {...register('national_id', {
            required: 'National ID is required',
            validate: (v) =>
              isValidThaiId(v.replace(/\D/g, '')) ||
              'Invalid ID — please check the 13-digit number on your card',
            setValueAs: (v: string) => v.replace(/\D/g, ''),
          })}
        />
        {errors.national_id && (
          <p className="text-xs text-destructive" role="alert">
            {errors.national_id.message}
          </p>
        )}
        <p className="text-xs text-muted-foreground">
          Enter the 13-digit number on your national ID card.
          Your identity will not be linked to your vote.
        </p>
      </div>

      {/* API-level error */}
      {error && (
        <div
          className="rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive"
          role="alert"
        >
          {error}
        </div>
      )}

      <button
        type="submit"
        disabled={isPending}
        className="rounded-full bg-primary px-6 py-2.5 text-sm font-semibold text-primary-foreground disabled:opacity-60 hover:opacity-90 transition-opacity"
      >
        {isPending ? 'Verifying…' : 'Continue'}
      </button>
    </form>
  );
}
