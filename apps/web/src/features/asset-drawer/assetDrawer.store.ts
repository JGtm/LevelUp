import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { useSidebarWidthStore } from '@/lib/layout/sidebarWidth'
import { clampPanelWidth, loadPanelWidth, savePanelWidth } from './panelWidth'

export type AssetTab = 'maps' | 'weapons' | 'medals'

/** Version du format persisté : la 1 ajoute le mode épinglé (un ancien état « ouvert » devient épinglé). */
const PERSIST_VERSION = 1

interface AssetDrawerState {
  isOpen: boolean
  /**
   * Mode du volet ancré (application de bureau) : épinglé, il pousse le contenu ; sinon il se superpose
   * à la page sous un voile et se ferme au clic dessus. Fermer le volet ne change pas le mode.
   */
  pinned: boolean
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
  setPinned: (pinned: boolean) => void
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
      pinned: false,
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
      setPinned: (pinned) => set({ pinned }),
      setTab: (tab) => set({ activeTab: tab, search: '' }),
      setSearch: (q) => set({ search: q }),
      setPanelWidth: (width) => {
        const next = clampPanelWidth(width, currentViewportWidth(), useSidebarWidthStore.getState().width)
        savePanelWidth(next)
        set({ panelWidth: next })
      },
    }),
    {
      name: 'levelup-asset-drawer',
      version: PERSIST_VERSION,
      // Ouvert n'est mémorisé que s'il est épinglé : un volet superposé se rouvre fermé.
      partialize: (s) => ({ isOpen: s.pinned && s.isOpen, pinned: s.pinned, activeTab: s.activeTab }),
      migrate: (persisted, version) => {
        const old = (persisted ?? {}) as { isOpen?: boolean }
        // Avant la version 1, ouvert voulait dire poussant le contenu : c'est le mode épinglé.
        return version < 1 ? { ...old, pinned: old.isOpen === true } : old
      },
    },
  ),
)
