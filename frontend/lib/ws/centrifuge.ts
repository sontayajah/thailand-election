// Centrifuge v5 WebSocket singleton.
// Subscribes to national, province, and referendum channels and delivers
// VoteUpdateEvent payloads to registered listeners.
//
// Usage (client component):
//   const { subscribe, unsubscribe } = useCentrifuge();
//   useEffect(() => {
//     const unsub = subscribe('election:thailand', (evt) => …);
//     return unsub;
//   }, [subscribe]);

import { Centrifuge, type PublicationContext } from 'centrifuge';
import type { VoteUpdateEvent } from '@/lib/types/election';

export type VoteEventHandler = (event: VoteUpdateEvent) => void;
export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected';

const WS_URL =
  process.env.NEXT_PUBLIC_CENTRIFUGO_URL ?? 'ws://localhost:8001/connection/websocket';

// ── Singleton state ────────────────────────────────────────────────────────────

let instance: Centrifuge | null = null;
let statusListeners: Array<(s: ConnectionStatus) => void> = [];
let currentStatus: ConnectionStatus = 'disconnected';

// channel name → set of listeners
const channelListeners = new Map<string, Set<VoteEventHandler>>();

function setStatus(s: ConnectionStatus) {
  currentStatus = s;
  statusListeners.forEach((fn) => fn(s));
}

// ── Build / reuse the singleton ────────────────────────────────────────────────

async function getOrCreate(): Promise<Centrifuge> {
  if (instance) return instance;

  // Lazily import token fetcher to avoid a hard dep at module load time
  const { fetchCentrifugoToken } = await import('@/lib/api/client');

  const cf = new Centrifuge(WS_URL, {
    getToken: async () => {
      const tok = await fetchCentrifugoToken();
      return tok;
    },
  });

  cf.on('connecting', () => setStatus('connecting'));
  cf.on('connected', () => setStatus('connected'));
  cf.on('disconnected', () => setStatus('disconnected'));

  instance = cf;
  return cf;
}

// ── Internal: ensure a subscription exists for a channel ──────────────────────

const subscriptions = new Map<string, ReturnType<Centrifuge['newSubscription']>>();

async function ensureSubscription(channel: string): Promise<void> {
  if (subscriptions.has(channel)) return;

  const cf = await getOrCreate();
  const sub = cf.newSubscription(channel);

  sub.on('publication', (ctx: PublicationContext) => {
    const listeners = channelListeners.get(channel);
    if (!listeners) return;
    const event = ctx.data as VoteUpdateEvent;
    listeners.forEach((fn) => fn(event));
  });

  sub.subscribe();
  subscriptions.set(channel, sub);

  // Connect the client the first time a subscription is created
  if (!['connecting', 'connected'].includes(currentStatus)) {
    cf.connect();
  }
}

// ── Public API ─────────────────────────────────────────────────────────────────

/**
 * Subscribe to a Centrifugo channel.
 * Returns an unsubscribe function — call it in useEffect cleanup.
 */
export async function subscribeChannel(
  channel: string,
  handler: VoteEventHandler
): Promise<() => void> {
  await ensureSubscription(channel);

  if (!channelListeners.has(channel)) {
    channelListeners.set(channel, new Set());
  }
  channelListeners.get(channel)!.add(handler);

  return () => {
    channelListeners.get(channel)?.delete(handler);
  };
}

/** Subscribe to connection status changes. Returns an unsubscribe fn. */
export function onStatusChange(fn: (s: ConnectionStatus) => void): () => void {
  statusListeners.push(fn);
  fn(currentStatus); // fire immediately with current state
  return () => {
    statusListeners = statusListeners.filter((f) => f !== fn);
  };
}

/** Disconnect and destroy the singleton (e.g. during hot-reload in dev). */
export function destroyCentrifuge(): void {
  instance?.disconnect();
  instance = null;
  subscriptions.clear();
  channelListeners.clear();
  currentStatus = 'disconnected';
}

// ── Pre-defined channel names ──────────────────────────────────────────────────

export const Channels = {
  national: 'election:thailand',
  province: (id: number) => `election:province:${id}`,
  referendum: 'election:referendum',
} as const;
