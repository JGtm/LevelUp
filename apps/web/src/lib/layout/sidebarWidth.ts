/**
 * Largeur réelle de la barre latérale de l'application de bureau : 240 px dépliée (`w-60`), 56 px
 * repliée en icônes (`w-14`). `SidebarNav` y publie son état ; ce qui se place à côté de la barre (le
 * volet des référentiels ancré) borne ses dimensions sur la largeur effective, pas sur la plus large.
 */
import { create } from 'zustand'

/** Barre dépliée (`w-60`). */
export const SIDEBAR_PANEL_WIDTH = 240
/** Barre repliée en icônes (`w-14`). */
export const SIDEBAR_RAIL_WIDTH = 56

interface SidebarWidthState {
  /** Largeur actuelle de la barre, en pixels ; celle de la barre dépliée tant qu'aucune n'est montée. */
  width: number
  setCollapsed: (collapsed: boolean) => void
}

export const useSidebarWidthStore = create<SidebarWidthState>()((set) => ({
  width: SIDEBAR_PANEL_WIDTH,
  setCollapsed: (collapsed) => set({ width: collapsed ? SIDEBAR_RAIL_WIDTH : SIDEBAR_PANEL_WIDTH }),
}))
