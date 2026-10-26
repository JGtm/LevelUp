/**
 * Données du graphe « Évolution LUSR / CSR » de la Carrière : points traçables et
 * échelle affichée.
 *
 * LUSR et CSR ne partagent pas d'axe : leurs nombres se ressemblent mais pas leurs
 * rangs (1600 = Platine I en LUSR, Onyx en CSR). Le graphe montre UNE échelle à la
 * fois, avec ses propres bandes de rang ; la bascule n'apparaît que si les deux
 * existent. Les matchs de placement (valeur stockée 0, pas un classement) ne se
 * tracent pas : ils faisaient plonger la courbe CSR à 0 à chaque début de saison.
 */
import type { CareerLusrCheckpoint } from '@/lib/api/types'
import { isPlacementTierLabel } from '@/lib/skillTiers'

export const RATING_SCALES = ['lusr', 'csr'] as const
export type RatingScale = typeof RATING_SCALES[number]

/** Échelle d'un checkpoint : csr si déclaré CSR, lusr sinon (défaut historique).
 *  Identifiants internes en minuscules, distincts des libellés affichés (manifeste). */
function scaleOf(cp: CareerLusrCheckpoint): RatingScale {
  return (cp.rating_type ?? '').toUpperCase() === 'CSR' ? 'csr' : 'lusr'
}

/** Checkpoints traçables : datés et hors phase de placement. */
export function chartableCheckpoints(checkpoints: CareerLusrCheckpoint[]): CareerLusrCheckpoint[] {
  return checkpoints.filter(cp => cp.recorded_at && !isPlacementTierLabel(cp.tier_label))
}

/** Échelles présentes parmi les checkpoints, dans l'ordre de RATING_SCALES. */
export function availableScales(checkpoints: CareerLusrCheckpoint[]): RatingScale[] {
  const present = new Set(checkpoints.map(scaleOf))
  return RATING_SCALES.filter(s => present.has(s))
}

/** Échelle effectivement affichée : la sélection si elle existe, sinon la première présente. */
export function resolveScale(selected: RatingScale | null, available: RatingScale[]): RatingScale | null {
  if (selected && available.includes(selected)) return selected
  return available[0] ?? null
}

export function checkpointsForScale(checkpoints: CareerLusrCheckpoint[], scale: RatingScale | null): CareerLusrCheckpoint[] {
  return scale ? checkpoints.filter(cp => scaleOf(cp) === scale) : []
}
