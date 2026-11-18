/**
 * planFeed.ts — CE QUE LE FIL POSÉ SUR LE PLAN MONTRE, logique pure. Le plan ne lit pas le fil de la colonne :
 * il lit LE MÊME fil que la page donne à la colonne (`model.feed`) et n'en garde que les éliminations, sans
 * le filtre du Standard — ni objectifs, ni prises, ni véhicules, ni entrées et sorties de partie, ni morts sans
 * tueur, ni médailles seules.
 *
 * Les cotes (nombre de lignes, estompage) sont celles de `cotesFilPlan.ts`.
 */
import { COTES_FIL_PLAN } from './cotesFilPlan'
import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'

/** Une entrée du fil qui est une élimination : son `kill` est là. */
export type EliminationEntry = ReplayFeedEntry & { kill: ReplayKill }

function isElimination(entree: ReplayFeedEntry): entree is EliminationEntry {
  return entree.kill !== null
}

/**
 * dernieresEliminations rend les `max` dernières éliminations déjà survenues à l'instant lu, de la plus
 * ancienne à la plus récente : l'ordre d'affichage, la plus récente EN BAS. `entries` est le fil de la page,
 * trié par instant ; il n'est ni modifié ni recopié en entier (la lecture part de la fin et s'arrête à `max`).
 */
export function dernieresEliminations(
  entries: readonly ReplayFeedEntry[],
  nowMs: number,
  max: number = COTES_FIL_PLAN.lignes,
): EliminationEntry[] {
  const lignes: EliminationEntry[] = []
  for (let i = entries.length - 1; i >= 0 && lignes.length < max; i--) {
    const entree = entries[i]
    if (isElimination(entree) && entree.replayMs <= nowMs) lignes.push(entree)
  }
  return lignes.reverse()
}

/**
 * opaciteLigneFilPlan rend l'opacité de la ligne d'indice `index` (0 = la plus ancienne) d'un fil de `total`
 * lignes : les deux plus anciennes s'estompent, mais seulement quand le fil est plein.
 */
export function opaciteLigneFilPlan(index: number, total: number): number {
  if (total < COTES_FIL_PLAN.lignes) return 1
  return COTES_FIL_PLAN.estompage[index] ?? 1
}
