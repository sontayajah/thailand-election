'use client';

// Client-side providers — wraps the whole app.
// Kept as a thin shell so the root layout stays a Server Component.

import { useState } from 'react';
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';

export function Providers({ children }: { children: React.ReactNode }) {
  // One QueryClient per browser session (not shared across requests).
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            // Keep cached data for 30 s before re-fetching
            staleTime: 30_000,
            // Retry failed requests twice before showing an error
            retry: 2,
            // Don't refetch on window focus — Centrifugo keeps data fresh
            refetchOnWindowFocus: false,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {process.env.NODE_ENV === 'development' && (
        <ReactQueryDevtools initialIsOpen={false} />
      )}
    </QueryClientProvider>
  );
}
