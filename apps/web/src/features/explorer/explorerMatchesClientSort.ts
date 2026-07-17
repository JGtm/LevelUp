/**
 * Tri CLIENT du tableau Explorer (mode Matchs) — helpers purs.
 *
 * Le tableau charge TOUTES les lignes du scope (page_size 10000) et pagine cote
 * client (getPaginationRowModel), donc TanStack trie localement
 * (getSortedRowModel) sur TOUTES les colonnes — instantane, aucune requete. Ce
 * module porte les fonctions de tri par valeur SOUS-JACENTE (jamais le libelle
 * formate) et le mapping id de colonne → libelle pour l'aria-label.
 *
 * Remplace l'ancien tri SERVEUR (mapping sortKey ⇄ SortingState, manualSorting)
 * limite a 5 colonnes — cf. thought_log 2026-07-17.
 */
import type { SortingFn } from '@tanstack/react-table'

import type { ExplorerMatchRow } from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

/**
 * Options de tri des colonnes numeriques nullables. `sortUndefined: 'last'`
 * range les valeurs `undefined` en bas dans LES DEUX sens (TanStack 8.21 renvoie
 * le placement undefined AVANT l'inversion desc — cf. getSortedRowModel). Les
 * accessors des colonnes coalescent `null` → `undefined` (sortUndefined ne
 * capture pas `null`) pour que nuls ET absents se rangent proprement en bas.
 */
export const NUMERIC_SORT = { sortingFn: 'basic', sortUndefined: 'last' } as const

/** Tri chronologique sur un timestamp ISO brut (pas le libelle de date formate). */
export const dateTimeSortingFn: SortingFn<ExplorerMatchRow> = (a, b, id) => {
  const ta = Date.parse(a.getValue<string>(id) || '')
  const tb = Date.parse(b.getValue<string>(id) || '')
  const na = Number.isNaN(ta)
  const nb = Number.isNaN(tb)
  if (na && nb) return 0
  if (na) return 1
  if (nb) return -1
  return ta - tb
}

/** Tri alphabetique locale-aware + numerique naturel ("Map2" < "Map10") pour les
 *  colonnes texte (valeur brute map_ui / mode_ui / playlist_label / rating_type). */
export const localeTextSortingFn: SortingFn<ExplorerMatchRow> = (a, b, id) => {
  const sa = String(a.getValue(id) ?? '')
  const sb = String(b.getValue(id) ?? '')
  return sa.localeCompare(sb, undefined, { numeric: true, sensitivity: 'base' })
}

/**
 * id de colonne TanStack → cle i18n du libelle texte-plein, pour l'aria-label
 * « Trier par {col} » (le header rendu peut etre un ReactNode : tooltip, 2 lignes,
 * abreviation « F »/« D »). On expose le libelle long/explicite au lecteur d'ecran.
 */
export const SORT_ARIA_LABEL_KEYS: Record<string, ExplorerManifestKey> = {
  start_time: 'explorer.matches.col_date',
  map_ui: 'explorer.filters.map',
  playlist_label: 'explorer.filters.playlist',
  mode_ui: 'explorer.filters.mode',
  is_with_friends: 'explorer.matches.col_squad',
  outcome_code: 'explorer.matches.col_outcome',
  dominance_flag: 'explorer.matches.col_dominance',
  kills: 'explorer.matches.col_kills_long',
  deaths: 'explorer.matches.col_deaths_long',
  assists: 'explorer.matches.col_assists_long',
  kda: 'explorer.matches.col_kda',
  score_label: 'explorer.matches.col_score',
  duration_seconds: 'explorer.matches.col_duration',
  perf_score: 'explorer.matches.col_perf',
  delta_perf: 'explorer.matches.col_delta_perf',
  rating_type: 'explorer.matches.col_rating',
  skill_tier_label: 'explorer.matches.col_rank',
  team_mmr: 'explorer.matches.col_team_mmr',
  enemy_mmr: 'explorer.matches.col_enemy_mmr',
  delta_mmr: 'explorer.matches.col_delta_mmr',
}
