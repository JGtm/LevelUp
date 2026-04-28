import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type AssetTab = 'maps' | 'weapons'

interface AssetDrawerState {
  isOpen: boolean
  activeTab: AssetTab
  search: string

  open: () => void
  close: () => void
  toggle: () => void
  setTab: (tab: AssetTab) => void
  setSearch: (q: string) => void
}

export const useAssetDrawerStore = create<AssetDrawerState>()(
  persist(
    (set) => ({
      isOpen: false,
      activeTab: 'maps',
      search: '',

      open: () => set({ isOpen: true }),
      close: () => set({ isOpen: false, search: '' }),
      toggle: () => set((s) => ({ isOpen: !s.isOpen, search: s.isOpen ? '' : s.search })),
      setTab: (tab) => set({ activeTab: tab, search: '' }),
      setSearch: (q) => set({ search: q }),
    }),
    {
      name: 'levelup-asset-drawer',
      partialize: (s) => ({ isOpen: s.isOpen, activeTab: s.activeTab }),
    },
  ),
)
