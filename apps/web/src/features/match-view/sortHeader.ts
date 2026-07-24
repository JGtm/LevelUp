/**
 * sortHeader.ts — helpers d'en-tête triable partagés par les tableaux hand-rolled
 * de la feature match-view (I16, pattern DetectionsPanel minimal : clic direct sur
 * le <th>, pas de bouton dédié). Utilisé par MatchScoreboard.tsx et
 * MatchEncountersTable.tsx — 2 usages, ne pas dupliquer un 3e site sans étendre ici.
 */

/** aria-sort d'un en-tête triable. */
export function ariaSortOf(dir: false | 'asc' | 'desc'): 'ascending' | 'descending' | 'none' {
  if (dir === 'asc') return 'ascending'
  if (dir === 'desc') return 'descending'
  return 'none'
}

/** Suffixe visuel du sens de tri actif (flèche), vide si colonne inactive. */
export function sortSuffixOf(dir: false | 'asc' | 'desc'): string {
  if (dir === 'asc') return ' ↑'
  if (dir === 'desc') return ' ↓'
  return ''
}
