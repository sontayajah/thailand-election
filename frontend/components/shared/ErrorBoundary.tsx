'use client';

// React error boundary for client components.
// Next.js error.tsx handles server-component errors; this handles runtime client errors.

import { Component, type ReactNode } from 'react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  override componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  override render() {
    if (!this.state.hasError) return this.props.children;

    if (this.props.fallback) return this.props.fallback;

    return (
      <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-4 text-sm text-destructive">
        <p className="font-semibold">Something went wrong</p>
        <p className="mt-1 text-xs opacity-70 font-mono break-all">
          {this.state.error?.message}
        </p>
        <button
          className="mt-3 text-xs underline underline-offset-2"
          onClick={() => this.setState({ hasError: false, error: null })}
        >
          Try again
        </button>
      </div>
    );
  }
}
