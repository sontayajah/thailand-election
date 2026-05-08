// Generic skeleton placeholder — use while data is loading.

interface SkeletonCardProps {
  /** Number of skeleton rows to render inside the card */
  rows?: number;
  className?: string;
}

export function SkeletonCard({ rows = 4, className = '' }: SkeletonCardProps) {
  return (
    <div
      className={`rounded-lg border border-border bg-card p-4 animate-pulse ${className}`}
      role="status"
      aria-label="Loading…"
    >
      {/* Card header */}
      <div className="h-4 bg-muted rounded w-2/5 mb-4" />
      {/* Rows */}
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 mb-3 last:mb-0">
          <div className="h-3 bg-muted rounded-full w-3 shrink-0" />
          <div
            className="h-3 bg-muted rounded"
            style={{ width: `${55 + (i % 3) * 15}%` }}
          />
          <div className="ml-auto h-3 bg-muted rounded w-10 shrink-0" />
        </div>
      ))}
    </div>
  );
}

/** Single-row skeleton — for table rows, list items. */
export function SkeletonRow({ className = '' }: { className?: string }) {
  return (
    <div
      className={`flex items-center gap-3 animate-pulse ${className}`}
      role="status"
      aria-label="Loading…"
    >
      <div className="h-3 bg-muted rounded-full w-3 shrink-0" />
      <div className="h-3 bg-muted rounded w-3/5" />
      <div className="ml-auto h-3 bg-muted rounded w-12 shrink-0" />
    </div>
  );
}

/** Full-width bar skeleton — for chart areas. */
export function SkeletonBar({ className = '' }: { className?: string }) {
  return (
    <div
      className={`h-32 bg-muted rounded animate-pulse ${className}`}
      role="status"
      aria-label="Loading…"
    />
  );
}
