/**
 * relationsPrefsStore — préférences d'affichage de la page Communauté > Relations,
 * mémorisées entre les visites (localStorage, portée GLOBALE — ce sont des choix
 * de présentation, pas des données liées à un joueur).
 *
 * Persiste les boutons de la page : chips de filtre, toggle « Amis inclus », et
 * le toggle de la heatmap (par heure / jour de semaine). La barre de
 * segmentation (useLocalFilterBar) reste page-local/commit-based — non persistée.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Miroir de relationsFilter.RelationFilter (évite un import stores → features ;
// même union littérale → structurellement compatible). Valeur 'cross' additive :
// aucune migration nécessaire (les états persistés existants restent valides).
export type RelationFilter = 'all' | 'core' | 'allies' | 'rivals' | 'recent' | 'cross'
export type RelationsHeatmapMode = 'hour' | 'day'

interface RelationsPrefsState {
  filter: RelationFilter
  includeFriends: boolean
  heatmapMode: RelationsHeatmapMode
  setFilter: (f: RelationFilter) => void
  setIncludeFriends: (v: boolean) => void
  setHeatmapMode: (m: RelationsHeatmapMode) => void
}

export const useRelationsPrefsStore = create<RelationsPrefsState>()(
  persist(
    (set) => ({
      filter: 'all',
      includeFriends: true,
      heatmapMode: 'hour',
      setFilter: (filter) => set({ filter }),
      setIncludeFriends: (includeFriends) => set({ includeFriends }),
      setHeatmapMode: (heatmapMode) => set({ heatmapMode }),
    }),
    {
      name: 'levelup-relations-prefs',
      version: 1,
      // v0 → v1 : l'ancien mode 'daypart' (6 tranches) devient 'hour' (24 créneaux).
      migrate: (persisted, version) => {
        const state = persisted as Partial<RelationsPrefsState> | undefined
        if (version < 1 && state && (state.heatmapMode as string) === 'daypart') {
          return { ...state, heatmapMode: 'hour' } as RelationsPrefsState
        }
        return state as RelationsPrefsState
      },
    },
  ),
)
