import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface FeedbackDrawerState {
  isOpen: boolean
  open: () => void
  close: () => void
  toggle: () => void
}

export const useFeedbackDrawerStore = create<FeedbackDrawerState>()(
  persist(
    (set) => ({
      isOpen: false,
      open: () => set({ isOpen: true }),
      close: () => set({ isOpen: false }),
      toggle: () => set((s) => ({ isOpen: !s.isOpen })),
    }),
    {
      name: 'levelup-feedback-drawer',
      partialize: (s) => ({ isOpen: s.isOpen }),
    },
  ),
)
