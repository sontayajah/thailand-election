// Voting portal layout — minimal, focused design for the ballot flow.
// Separate from the main dashboard layout.

import Link from 'next/link';

export const metadata = {
  title: 'Online Voting — Thailand Election 2026',
};

export default function VoteLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col bg-background">
      {/* Minimal header */}
      <header className="border-b border-border">
        <div className="max-w-lg mx-auto px-4 py-3 flex items-center justify-between">
          <Link href="/" className="text-sm font-semibold">
            🗳 Thailand Election 2026
          </Link>
          <span className="text-xs text-muted-foreground">Secure Online Voting</span>
        </div>
      </header>

      <main className="flex-1 flex flex-col">
        {children}
      </main>

      <footer className="border-t border-border py-4 text-center text-xs text-muted-foreground">
        Your identity is never linked to your vote · End-to-end anonymous
      </footer>
    </div>
  );
}
