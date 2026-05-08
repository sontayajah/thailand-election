// Voting session storage — sessionStorage only (never localStorage).
// Cleared automatically when the tab is closed, per PRD security requirement.

const KEY_SESSION_ID = 'th_election_session_id';
const KEY_JWT = 'th_election_jwt';
const KEY_DEV_OTP = 'th_election_dev_otp';
const KEY_RECEIPTS = 'th_election_receipts';

export interface CastReceipt {
  ballot_type: string;
  receipt_hash: string;
}

function storage(): Storage | null {
  if (typeof window === 'undefined') return null;
  return window.sessionStorage;
}

// ── Session ID (set after verify-id, before OTP) ──────────────────────────────

export function setSessionId(id: string): void {
  storage()?.setItem(KEY_SESSION_ID, id);
}

export function getSessionId(): string | null {
  return storage()?.getItem(KEY_SESSION_ID) ?? null;
}

// ── Voter JWT (set after verify-otp, authorises ballot/cast/eligibility) ──────

export function setVoterJWT(token: string): void {
  storage()?.setItem(KEY_JWT, token);
}

export function getVoterJWT(): string | null {
  return storage()?.getItem(KEY_JWT) ?? null;
}

export function clearVoterJWT(): void {
  storage()?.removeItem(KEY_JWT);
}

// ── Dev OTP (set from request-otp response when OTP_DEV_MODE=true) ───────────

export function setDevOTP(otp: string | undefined): void {
  if (otp) {
    storage()?.setItem(KEY_DEV_OTP, otp);
  } else {
    storage()?.removeItem(KEY_DEV_OTP);
  }
}

export function getDevOTP(): string | null {
  return storage()?.getItem(KEY_DEV_OTP) ?? null;
}

// ── Cast receipts (accumulated across the 3 ballot steps) ────────────────────

export function addReceipt(r: CastReceipt): void {
  const existing = getReceipts();
  const updated = [...existing.filter((e) => e.ballot_type !== r.ballot_type), r];
  storage()?.setItem(KEY_RECEIPTS, JSON.stringify(updated));
}

export function getReceipts(): CastReceipt[] {
  try {
    const raw = storage()?.getItem(KEY_RECEIPTS);
    if (!raw) return [];
    return JSON.parse(raw) as CastReceipt[];
  } catch {
    return [];
  }
}

// ── Full session clear (logout / expiry) ──────────────────────────────────────

export function clearSession(): void {
  const s = storage();
  if (!s) return;
  s.removeItem(KEY_SESSION_ID);
  s.removeItem(KEY_JWT);
  s.removeItem(KEY_DEV_OTP);
  s.removeItem(KEY_RECEIPTS);
}

// ── Auth guard helper ─────────────────────────────────────────────────────────

/** Returns true if a voter JWT is present in sessionStorage. */
export function isAuthenticated(): boolean {
  return getVoterJWT() !== null;
}
