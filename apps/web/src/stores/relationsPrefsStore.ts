/**
 * relationsPrefsStore — préférences d'affichage de la page Communauté > Relations,
 * mémorisées entre les visites (localStorage, portée GLOBALE — ce sont des choix
 * de présentation, pas des données liées à un joueur).
 *
 * Persiste les boutons de la page : chips de filtre, toggle « Jamais affrontés »,
 * et le toggle de la heatmap (par heure / jour de semaine). La barre de
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
  // includeNeverFaced : afficher les relations jamais affrontées (enemy_matches === 0).
  // Défaut MASQUÉ (false) — la page se concentre par défaut sur les joueurs croisés
  // en adversaire. Anciennement `includeFriends` (libellé trompeur, défaut inclus).
  includeNeverFaced: boolean
  heatmapMode: RelationsHeatmapMode
  setFilter: (f: RelationFilter) => void
  setIncludeNeverFaced: (v: boolean) => void
  setHeatmapMode: (m: RelationsHeatmapMode) => void
}

/**
 * migrateRelationsPrefs — migration du state persisté (exportée pour test).
 *  - v0 → v1 : l'ancien mode heatmap 'daypart' (6 tranches) devient 'hour' (24 créneaux).
 *  - v1 → v2 [2026-07-17] : le toggle « amis » (`includeFriends`, défaut inclus)
 *    masquait en réalité les relations jamais affrontées. Renommé `includeNeverFaced`,
 *    défaut MASQUÉ. Changement de sémantique assumé → réinitialisation UNIQUE à
 *    `false` ; `filter` et `heatmapMode` préservés, ancienne clé supprimée.
 */
export function migrateRelationsPrefs(persisted: unknown, version: number): RelationsPrefsState {
  const state = { ...((persisted as Record<string, unknown>) ?? {}) }
  if (version < 1 && state.heatmapMode === 'daypart') {
    state.heatmapMode = 'hour'
  }
  if (version < 2) {
    delete state.includeFriends
    state.includeNeverFaced = false
  }
  return state as unknown as RelationsPrefsState
}

export const useRelationsPrefsStore = create<RelationsPrefsState>()(
  persist(
    (set) => ({
      filter: 'all',
      includeNeverFaced: false,
      heatmapMode: 'hour',
      setFilter: (filter) => set({ filter }),
      setIncludeNeverFaced: (includeNeverFaced) => set({ includeNeverFaced }),
      setHeatmapMode: (heatmapMode) => set({ heatmapMode }),
    }),
    {
      name: 'levelup-relations-prefs',
      version: 2,
      migrate: (persisted, version) => migrateRelationsPrefs(persisted, version),
    },
  ),
)
