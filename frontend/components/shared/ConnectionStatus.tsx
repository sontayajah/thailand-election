'use client';

// Real-time connection status banner (F-09).
// Shows a coloured strip at the top of the page when the WebSocket is not live.

import { useEffect, useState } from 'react';
import type { ConnectionStatus as WsStatus } from '@/lib/ws/centrifuge';
import { onStatusChange, Channels, subscribeChannel } from '@/lib/ws/centrifuge';

export function ConnectionStatus() {
  const [status, setStatus] = useState<WsStatus>('disconnected');

  useEffect(() => {
    // Register status listener
    const unsub = onStatusChange(setStatus);

    // Trigger connection by subscribing to the national channel
    let cleanup: (() => void) | undefined;
    subscribeChannel(Channels.national, () => {}).then((fn) => {
      cleanup = fn;
    });

    return () => {
      unsub();
      cleanup?.();
    };
  }, []);

  if (status === 'connected') return null; // fully live — show nothing

  const isReconnecting = status === 'connecting';

  return (
    <div
      role="status"
      aria-live="polite"
      className={[
        'fixed top-0 inset-x-0 z-50 flex items-center justify-center gap-2',
        'px-4 py-1.5 text-xs font-medium',
        isReconnecting
          ? 'bg-amber-500 text-amber-950'
          : 'bg-rose-600 text-white',
      ].join(' ')}
    >
      <span
        className={[
          'inline-block h-2 w-2 rounded-full',
          isReconnecting ? 'bg-amber-900 animate-pulse' : 'bg-rose-200',
        ].join(' ')}
      />
      {isReconnecting
        ? 'Reconnecting to live updates…'
        : 'Live updates unavailable — refreshing every 10 s'}
    </div>
  );
}
