/**
 * filmGap.logic.ts — « ANALYSE DU FILM EN COURS » / « FILM NON ANALYSÉ » : la décision, PARTAGÉE.
 *
 * Une section tirée du film (usages des ressources, portée, vies, équipement, records de distance) qui
 * n'a rien à montrer parce que le film du périmètre n'est pas lu reste AFFICHÉE avec un état vide : une
 * section absente laissait croire à un bug. Capability absente (le titre ne lit pas le film) : section
 * absente, comme avant. Film lu partout et section sans sujet : section absente aussi.
 *
 * Les comptes viennent du Go (règle unique `analysis.AnalysisPendingParts` via `squadagg.NoteFilmGaps`,
 * `weapon_records_gap` pour la Synthèse), jamais recalculés ici. Les pages Escouade, Séries temporelles et
 * Sessions lisent LE MÊME bloc d'Emprise : une seule lecture des compteurs, garde-rail `filmGap.guard.test.ts`.
 */
import type { SquadEmpriseBlock } from '@/lib/api/types'

/** `pending` : au moins une analyse en cours (« Analyse du film en cours ») ; `missing` : aucune en cours (« Film non analysé »). */
export type FilmGap = 'pending' | 'missing'

/** Le titre lit le film : le bloc existe et ne dit pas `film_unsupported` (capability `film.usage_summary`). */
export function filmExpected(block: Pick<SquadEmpriseBlock, 'film_unavailable'> | null | undefined): boolean {
  return block != null && block.film_unavailable !== 'film_unsupported'
}

/**
 * Le film du périmètre n'est pas lu pour au moins un match : `pending` (au moins une analyse en cours) ou
 * `missing` (aucune en cours). `null` : le titre ne lit pas le film, ou tous les matchs ont leur résumé —
 * une section sans donnée est alors légitimement absente.
 */
export function filmGapOfBlock(
  block: Pick<SquadEmpriseBlock, 'film_unavailable' | 'matches_film_pending' | 'matches_film_missing'> | null | undefined,
): FilmGap | null {
  if (!filmExpected(block)) return null
  if ((block?.matches_film_pending ?? 0) > 0) return 'pending'
  return (block?.matches_film_missing ?? 0) > 0 ? 'missing' : null
}
