/**
 * orientationOracle.ts — L'ORACLE DES TESTS D'ORIENTATION : la rotation d'une position monde
 * autour du centre de la scène, et son inverse (la rotation de `4 - turns` quarts de tour).
 *
 * Réservé aux tests : le code de production tourne par `orientAbout` avec le pivot du cadrage
 * (`CanvasView.orient`) et n'a pas besoin de ces deux raccourcis.
 */
import type { ReplayBounds } from '@/lib/api/types'

import type { XY } from '../../../lib/replay/replayLogic'
import { normalizeTurns, orientAbout, pivotOf, type QuarterTurns } from '@/lib/replay/orientation'

/** Une position monde tournée de `turns` quarts de tour horaires autour du centre de `scene`. */
export function turnWorld(p: XY, turns: QuarterTurns, scene: ReplayBounds): XY {
  return orientAbout(p, turns, pivotOf(scene))
}

/** L'inverse de `turnWorld` : la position monde d'origine d'un point tourné. */
export function unturnWorld(p: XY, turns: QuarterTurns, scene: ReplayBounds): XY {
  return orientAbout(p, normalizeTurns(4 - turns), pivotOf(scene))
}
