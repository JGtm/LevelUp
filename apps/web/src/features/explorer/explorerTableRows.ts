import type { ExplorerMatchRow, ExplorerMatchesQueryResponse } from '@/lib/api/types'

/**
 * Normalise les lignes du tableau Explorer renvoyées par le contrat
 * (`map_ui` / `mode_ui` / `playlist_label` peuvent être `null` côté Go) vers
 * `ExplorerMatchRow[]` attendu par <ExplorerMatchesTable> (libellés string).
 * Coalesce `null` → '' au moment du passage de prop ; aucun changement de rendu
 * (une cellule null s'affichait déjà vide).
 */
export function normalizeExplorerTableRows(
  items: ExplorerMatchesQueryResponse['table']['items'],
): ExplorerMatchRow[] {
  return (items ?? []).map((r) => ({
    ...r,
    map_ui: r.map_ui ?? '',
    mode_ui: r.mode_ui ?? '',
    playlist_label: r.playlist_label ?? '',
  }))
}

/**
 * Lignes des matchs communs, réparties entre les deux tableaux du mode Joueur.
 *
 * UNE SEULE requête matches-query porte tous les matchs communs : chaque ligne est calculée
 * match par match côté serveur (le contenu d'une ligne ne dépend pas des autres matchs de la
 * liste blanche), donc la répartition côté client rend les mêmes lignes que deux requêtes
 * séparées, dans le même ordre (le filtre conserve l'ordre serveur).
 */
export interface CommonMatchRowsSplit {
  ally: ExplorerMatchRow[]
  enemy: ExplorerMatchRow[]
}

export function splitCommonMatchRows(
  items: ExplorerMatchesQueryResponse['table']['items'],
  allyMatchIds: readonly string[],
  enemyMatchIds: readonly string[],
): CommonMatchRowsSplit {
  const ally = new Set(allyMatchIds)
  const enemy = new Set(enemyMatchIds)
  const rows = normalizeExplorerTableRows(items)
  return {
    ally: rows.filter((r) => ally.has(r.match_id)),
    enemy: rows.filter((r) => enemy.has(r.match_id)),
  }
}
