/**
 * explorerPrefsStore — préférences d'affichage de la page Explorer, mémorisées
 * entre les visites (localStorage, portée GLOBALE — ce sont des choix de
 * présentation, pas des données liées à un joueur, cf. relationsPrefsStore).
 *
 * Persiste l'état réduit/déplié de la synthèse (bandeau au-dessus du tableau,
 * mode Matchs). Défaut DÉPLIÉ : la synthèse reste visible tant que l'utilisateur
 * ne la réduit pas explicitement.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface ExplorerPrefsState {
  // briefingCollapsed : synthèse repliée au-dessus du tableau (mode Matchs).
  briefingCollapsed: boolean
  setBriefingCollapsed: (v: boolean) => void
  toggleBriefingCollapsed: () => void
}

/**
 * migrateExplorerPrefs — migration du state persisté (exportée pour test).
 * v1 initiale : aucun renommage/changement de sémantique à ce jour.
 */
export function migrateExplorerPrefs(persisted: unknown): ExplorerPrefsState {
  const state = { ...((persisted as Record<string, unknown>) ?? {}) }
  return state as unknown as ExplorerPrefsState
}

export const useExplorerPrefsStore = create<ExplorerPrefsState>()(
  persist(
    (set) => ({
      briefingCollapsed: false,
      setBriefingCollapsed: (briefingCollapsed) => set({ briefingCollapsed }),
      toggleBriefingCollapsed: () =>
        set((s) => ({ briefingCollapsed: !s.briefingCollapsed })),
    }),
    {
      name: 'levelup-explorer-prefs',
      version: 1,
      migrate: (persisted) => migrateExplorerPrefs(persisted),
    },
  ),
)
