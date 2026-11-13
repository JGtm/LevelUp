/**
 * wallPlan — LE PLAN DES MURS D'UNE CARTE, NORMALISÉ À L'ENTRÉE.
 *
 * Le sidecar du fond de carte porte, quand la carte l'a, les arêtes de ses murs : des segments
 * en mètres MONDE, dans le repère même du rejeu (`[x1, y1, x2, y2]`). Le transport est
 * nullable de bout en bout (liste nulle, arête nulle, arête trop courte) ; ce module résout
 * cette nullabilité UNE fois, comme `normalizeCallouts` pour les zones nommées, afin que le
 * calque (`layers/wallPlanLayer.ts`) ne reçoive que des nombres finis.
 *
 * PAS DE PLAN N'EST PAS UNE ERREUR : Live Fire et deux cartes Forge n'en ont pas, et le rejeu
 * se passe alors du calque. Une ARÊTE mal formée, elle, est écartée — le reste du plan se
 * dessine — et signalée une seule fois par plan, jamais à chaque rendu.
 *
 * Pas de React, pas de canvas : logique pure, testée dans `wallPlan.test.ts`.
 */
import { log } from '@/lib/accessibility/_logger'
import type { ReplayMapBackgroundPlan } from '@/lib/api/types'

/** Une arête monde : x1, y1, x2, y2 en mètres. */
export type WallEdge = readonly [number, number, number, number]

/** Le plan prêt à dessiner. `dropped` compte les arêtes refusées (0 pour un plan sain). */
export interface WallPlanReady {
  edges: ReadonlyArray<WallEdge>
  dropped: number
}

/** Une arête est exploitable si elle porte quatre nombres finis (les coordonnées au-delà sont ignorées). */
function wellFormed(edge: readonly number[] | null | undefined): edge is readonly number[] {
  if (!edge || edge.length < 4) return false
  for (let i = 0; i < 4; i++) {
    if (!Number.isFinite(edge[i])) return false
  }
  return true
}

/**
 * normalizeWallPlan lit le plan du sidecar. Rend null sans plan, ou quand aucune arête n'est
 * exploitable ; sinon les arêtes bien formées, et le nombre de celles qui ont été refusées.
 *
 * Le refus est journalisé UNE fois par plan (`log.error` déduplique par clé : la source, le
 * nombre de sommets et de faces identifient le plan d'un document).
 */
export function normalizeWallPlan(plan: ReplayMapBackgroundPlan | null | undefined): WallPlanReady | null {
  if (!plan?.edges) return null
  const edges: WallEdge[] = []
  let dropped = 0
  for (const e of plan.edges) {
    if (wellFormed(e)) edges.push([e[0], e[1], e[2], e[3]])
    else dropped++
  }
  if (dropped > 0) {
    log.error(
      `replay-wall-plan:${plan.source}:${plan.vertices}:${plan.faces}`,
      `Plan des murs : ${dropped} arête(s) mal formée(s) écartée(s) sur ${plan.edges.length}.`,
    )
  }
  return edges.length > 0 ? { edges, dropped } : null
}
