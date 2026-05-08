// Zustand v5 UI store.
// Tracks: dark mode toggle, selected province, ballot type toggle.

import { create } from 'zustand';
import type { BallotType } from '@/lib/types/election';

type MapBallotView = 'CONSTITUENCY' | 'PARTY_LIST';

interface UIState {
  // Dark mode
  darkMode: boolean;
  toggleDarkMode: () => void;
  setDarkMode: (v: boolean) => void;

  // Province selection (for drill-down panel)
  selectedProvinceId: number | null;
  setSelectedProvinceId: (id: number | null) => void;

  // Which ballot type to colour the map by
  mapBallotView: MapBallotView;
  setMapBallotView: (v: MapBallotView) => void;

  // Active tab on the national dashboard
  activeTab: 'seats' | 'votes' | 'referendum';
  setActiveTab: (v: 'seats' | 'votes' | 'referendum') => void;
}

export const useUIStore = create<UIState>((set) => ({
  darkMode: false,
  toggleDarkMode: () =>
    set((s) => {
      const next = !s.darkMode;
      // Persist to <html> so Tailwind dark variant works
      if (typeof document !== 'undefined') {
        document.documentElement.classList.toggle('dark', next);
      }
      return { darkMode: next };
    }),
  setDarkMode: (v) =>
    set(() => {
      if (typeof document !== 'undefined') {
        document.documentElement.classList.toggle('dark', v);
      }
      return { darkMode: v };
    }),

  selectedProvinceId: null,
  setSelectedProvinceId: (id) => set({ selectedProvinceId: id }),

  mapBallotView: 'CONSTITUENCY',
  setMapBallotView: (v) => set({ mapBallotView: v }),

  activeTab: 'seats',
  setActiveTab: (v) => set({ activeTab: v }),
}));

// Initialise dark mode from system preference on first import (client only).
if (typeof window !== 'undefined') {
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  useUIStore.getState().setDarkMode(prefersDark);

  window
    .matchMedia('(prefers-color-scheme: dark)')
    .addEventListener('change', (e) => {
      useUIStore.getState().setDarkMode(e.matches);
    });
}
