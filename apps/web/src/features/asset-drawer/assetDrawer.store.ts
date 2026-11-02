import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { clampPanelWidth, loadPanelWidth, savePanelWidth } from './panelWidth'

export type AssetTab = 'maps' | 'weapons' | 'medals'

interface AssetDrawerState {
  isOpen: boolean
  activeTab: AssetTab
  search: string
  /** Largeur du volet ancré (application de bureau), en pixels ; mémorisée par `panelWidth`. */
  panelWidth: number
  /**
   * Compte les ouvertures demandées par l'utilisateur (bouton, raccourci) : le volet ancré y prend le
   * focus dans son champ de recherche. Non persisté : un volet rouvert au démarrage ne vole pas le focus.
   */
  focusToken: number

  open: () => void
  close: () => void
  toggle: () => void
  setTab: (tab: AssetTab) => void
  setSearch: (q: string) => void
  setPanelWidth: (width: number) => void
}

function currentViewportWidth(): number {
  return typeof window === 'undefined' ? Number.POSITIVE_INFINITY : window.innerWidth
}

export const useAssetDrawerStore = create<AssetDrawerState>()(
  persist(
    (set) => ({
      isOpen: false,
      activeTab: 'maps',
      search: '',
      panelWidth: loadPanelWidth(),
      focusToken: 0,

      open: () => set((s) => ({ isOpen: true, focusToken: s.isOpen ? s.focusToken : s.focusToken + 1 })),
      close: () => set({ isOpen: false, search: '' }),
      toggle: () =>
        set((s) => ({
          isOpen: !s.isOpen,
          search: s.isOpen ? '' : s.search,
          focusToken: s.isOpen ? s.focusToken : s.focusToken + 1,
        })),
      setTab: (tab) => set({ activeTab: tab, search: '' }),
      setSearch: (q) => set({ search: q }),
      setPanelWidth: (width) => {
        const next = clampPanelWidth(width, currentViewportWidth())
        savePanelWidth(next)
        set({ panelWidth: next })
      },
    }),
    {
      name: 'levelup-asset-drawer',
      partialize: (s) => ({ isOpen: s.isOpen, activeTab: s.activeTab }),
    },
  ),
)
