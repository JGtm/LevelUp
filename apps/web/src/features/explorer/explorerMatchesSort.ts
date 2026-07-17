/**
 * Tri serveur du tableau Explorer (mode Matchs) — logique pure partagee entre
 * le <select> « Trier par » (ExplorerPage.matchesMode.tsx) et les en-tetes de
 * colonnes cliquables (ExplorerMatchesTable.tsx).
 *
 * Le tri est SERVEUR (la reponse est cappee a 10000 lignes ; un tri client
 * serait silencieusement faux) → TanStack Table tourne en `manualSorting`.
 * L'etat canonique est la chaine `sortKey` = `"{champ}:{dir}"` portee par le
 * scope Explorer (ExplorerScope.sortKey), source de verite UNIQUE du select ET
 * des en-tetes. Ce module convertit entre cette chaine et le SortingState
 * TanStack, sans jamais trier cote client.
 *
 * Seules les colonnes dont la cle de tri est reellement honoree par le backend
 * (service.compareMatchHistoryRows) sont triables par en-tete. La colonne
 * « Resultat » n'en fait PAS partie : le select envoie `outcome` mais le backend
 * ne connait que `outcome_code` → le tri retombe sur `start_time` (bug backend
 * preexistant, hors perimetre — cf. plan V2 §6, non traite ici).
 */
import type { SortingState } from '@tanstack/react-table'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

/**
 * id de colonne TanStack (accessorKey) → cle de tri SERVEUR (stem de la valeur
 * `sortKey` = `"{champ}:{dir}"`). Miroir EXACT des valeurs du <select> ; la cle
 * serveur peut differer de l'accessorKey (`perf_score` →
 * `performance_score_relative`). Ces cinq champs sont ceux honores par
 * `compareMatchHistoryRows` cote Go.
 */
export const EXPLORER_COLUMN_SORT_KEYS: Record<string, string> = {
  start_time: 'start_time',
  perf_score: 'performance_score_relative',
  kills: 'kills',
  kda: 'kda',
  delta_mmr: 'delta_mmr',
}

/** cle serveur → id de colonne (inverse de EXPLORER_COLUMN_SORT_KEYS). */
const COLUMN_ID_BY_SORT_KEY: Record<string, string> = Object.fromEntries(
  Object.entries(EXPLORER_COLUMN_SORT_KEYS).map(([col, key]) => [key, col]),
)

/**
 * id de colonne triable → cle i18n du libelle texte-plein, pour l'aria-label
 * « Trier par {col} » (le header rendu peut etre un ReactNode : tooltip, 2 lignes).
 */
export const EXPLORER_SORT_LABEL_KEYS: Record<string, ExplorerManifestKey> = {
  start_time: 'explorer.matches.col_date',
  perf_score: 'explorer.matches.col_perf',
  kills: 'explorer.matches.col_kills',
  kda: 'explorer.matches.col_kda',
  delta_mmr: 'explorer.matches.col_delta_mmr',
}

/** Une colonne est-elle triable par en-tete (cle de tri serveur honoree) ? */
export function isSortableColumn(columnId: string | undefined | null): boolean {
  return columnId != null && columnId in EXPLORER_COLUMN_SORT_KEYS
}

/**
 * `"kda:desc"` → SortingState TanStack `[{ id: 'kda', desc: true }]`.
 * Direction : `asc` → `desc:false`, tout le reste (`desc`/absent) → `desc:true`
 * (la direction reste explicite cote UI). Colonne inconnue / chaine vide → `[]`
 * (aucun en-tete actif, ex. tri « Resultat » pose via le select).
 */
export function sortKeyToSorting(sortKey: string | undefined | null): SortingState {
  if (!sortKey) return []
  const [field, dir] = sortKey.split(':')
  const columnId = COLUMN_ID_BY_SORT_KEY[field]
  if (!columnId) return []
  return [{ id: columnId, desc: dir !== 'asc' }]
}

/**
 * SortingState TanStack → `"{champ serveur}:{dir}"`. `null` si l'etat est vide
 * ou si la 1re colonne n'a pas de cle serveur (jamais propage au scope).
 */
export function sortingToSortKey(sorting: SortingState): string | null {
  const first = sorting[0]
  if (!first) return null
  const field = EXPLORER_COLUMN_SORT_KEYS[first.id]
  if (!field) return null
  return `${field}:${first.desc ? 'desc' : 'asc'}`
}
