/**
 * relationsPrefsStore — préférences d'affichage de la page Communauté > Relations,
 * mémorisées entre les visites (localStorage, portée GLOBALE — ce sont des choix
 * de présentation, pas des données liées à un joueur).
 *
 * Persiste les boutons de la page : chips de filtre, toggle « Amis inclus », et
 * le toggle de la heatmap (tranche horaire / jour de semaine). La barre de
 * segmentation (useLocalFilterBar) reste page-local/commit-based — non persistée.
 */
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Miroir de relationsFilter.RelationFilter (évite un import stores → features ;
// même union littérale → structurellement compatible).
export type RelationFilter = 'all' | 'core' | 'allies' | 'rivals' | 'recent'
export type RelationsHeatmapMode = 'daypart' | 'day'

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
      heatmapMode: 'daypart',
      setFilter: (filter) => set({ filter }),
      setIncludeFriends: (includeFriends) => set({ includeFriends }),
      setHeatmapMode: (heatmapMode) => set({ heatmapMode }),
    }),
    { name: 'levelup-relations-prefs' },
  ),
)
