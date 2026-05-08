'use client';

// 6-digit OTP input component — wraps react-otp-input v3.

import OTPInputLib from 'react-otp-input';

interface Props {
  value: string;
  onChange: (value: string) => void;
  disabled?: boolean;
}

export function OtpInput({ value, onChange, disabled }: Props) {
  return (
    <OTPInputLib
      value={value}
      onChange={onChange}
      numInputs={6}
      shouldAutoFocus
      inputType="tel"
      renderSeparator={<span className="mx-1 text-muted-foreground select-none">–</span>}
      renderInput={(props) => (
        <input
          {...props}
          disabled={disabled}
          className={[
            'w-10 h-12 rounded-md border text-center text-xl font-mono font-bold',
            'bg-background text-foreground',
            'outline-none transition-colors',
            'border-input focus:border-primary focus:ring-2 focus:ring-primary/30',
            disabled ? 'opacity-50 cursor-not-allowed' : '',
          ].join(' ')}
        />
      )}
      containerStyle="flex items-center justify-center"
      skipDefaultStyles
    />
  );
}
