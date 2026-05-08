'use client';

// Global error boundary for the root layout (server component errors).

import { useEffect } from 'react';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error('[GlobalError]', error);
  }, [error]);

  return (
    <html lang="th">
      <body className="min-h-screen flex flex-col items-center justify-center bg-background text-foreground p-8">
        <div className="max-w-md w-full rounded-lg border border-destructive/30 bg-destructive/5 p-6 text-center">
          <h2 className="text-lg font-semibold text-destructive mb-2">
            Something went wrong
          </h2>
          <p className="text-sm text-muted-foreground mb-4">
            {error.message ?? 'An unexpected error occurred.'}
          </p>
          <button
            onClick={reset}
            className="text-sm px-4 py-2 rounded-md bg-primary text-primary-foreground hover:opacity-90"
          >
            Try again
          </button>
        </div>
      </body>
    </html>
  );
}
