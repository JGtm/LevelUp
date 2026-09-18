/**
 * floorRings.ts — LE LANGAGE D'ÉTAGE DU REJEU : des anneaux concentriques, un par étage au-dessus
 * du sol. UNE SEULE IMPLÉMENTATION, partagée par les pions (`replayMarkers.ts`) et par les
 * objectifs du mode (`objectivesLayer.ts`, lot du 2026-09-18).
 *
 * LA DOCTRINE VIENT DES PIONS (planche du 2026-08-16, A1 du 2026-08-18) et ne change pas en
 * s'étendant aux objectifs : jamais un décalage du marqueur (en vue de dessus, déplacer un point
 * vers le haut de l'écran voudrait dire « plus au nord »), l'altitude est un PALIER et non un
 * dégradé, c'est le NOMBRE d'anneaux qui dit la hauteur et lui seul — la couleur ne dit que le
 * camp. Un socle de drapeau à l'étage porte donc exactement ce que porte un joueur à l'étage.
 *
 * POURQUOI UN FICHIER, ET PAS UN EXPORT DE `replayMarkers.ts` : les objectifs se cuisent hors
 * écran, sans document ni image courante ; importer le module des pions pour trois constantes
 * et une boucle aurait tiré avec lui le cône de visée, les étiquettes et les chevrons. La règle
 * du dépôt (≤ 2 copies d'un motif, la 3e se centralise) s'applique dès la DEUXIÈME ici : deux
 * boucles d'anneaux divergeraient au premier réglage, et l'écart serait invisible parce que
 * crédible.
 *
 * LE PREMIER RAYON EST UN PARAMÈTRE, le pas ne l'est pas. Le pion pose son premier anneau à
 * `FLOOR_RING_RADIUS` (6,5 px, détaché du point) ; un marqueur d'objectif porte déjà un anneau
 * de LIVRAISON à 8 px et doit poser le sien au-delà, sinon les deux se confondraient. Le PAS
 * entre anneaux, lui, est le même partout : c'est lui que l'œil compte.
 */
import type { XY } from '../../../lib/replay/replayLogic'

/** Rayon du PREMIER anneau d'un pion (planche du 2026-08-16 : détaché du point, pas collé). */
export const FLOOR_RING_RADIUS = 6.5
/** Pas entre deux anneaux successifs — le même pour les pions et les objectifs. */
export const FLOOR_RING_GAP = 2.8
export const FLOOR_RING_WIDTH = 1
export const FLOOR_RING_ALPHA = 0.9
/** Chaque anneau au-delà du premier pâlit d'autant : le premier reste le plus franc. */
export const FLOOR_RING_ALPHA_DECAY = 0.18

/**
 * floorRingRadius : rayon (px, avant facteur d'échelle) de l'anneau n° `r` (1 = le premier
 * au-dessus du sol), pour un premier anneau posé à `first`.
 */
export function floorRingRadius(r: number, first: number = FLOOR_RING_RADIUS): number {
  return first + FLOOR_RING_GAP * (r - 1)
}

/** Ce qui varie d'un appelant à l'autre : le facteur d'échelle et le premier rayon. */
export interface FloorRingsGeometry {
  /** Facteur d'échelle du marqueur (les pions en portent un, les objectifs non). */
  k?: number
  /** Rayon du premier anneau — cf. l'en-tête. */
  first?: number
}

/**
 * drawFloorRings trace `fl` anneaux concentriques autour de `c`, dans `color`, en pâlissant
 * vers l'extérieur. `fl = 0` ne trace rien. Laisse `globalAlpha` à 1 en sortant.
 */
export function drawFloorRings(
  ctx: CanvasRenderingContext2D,
  c: XY,
  fl: number,
  color: string,
  geometry: FloorRingsGeometry = {},
): void {
  if (fl <= 0) return
  const k = geometry.k ?? 1
  const first = geometry.first ?? FLOOR_RING_RADIUS
  ctx.strokeStyle = color
  ctx.lineWidth = FLOOR_RING_WIDTH * k
  for (let r = 1; r <= fl; r++) {
    ctx.globalAlpha = FLOOR_RING_ALPHA - FLOOR_RING_ALPHA_DECAY * (r - 1)
    ctx.beginPath()
    ctx.arc(c.x, c.y, floorRingRadius(r, first) * k, 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}
