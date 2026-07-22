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
