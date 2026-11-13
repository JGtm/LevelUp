/**
 * orientation — L'ORIENTATION DE LA CARTE, PAR QUARTS DE TOUR.
 *
 * Une carte se lit mieux quand sa grande dimension suit celle de l'écran, ou quand le point de
 * vue habituel du joueur est en bas : l'utilisateur tourne donc la carte, par quarts de tour, et
 * ce choix est retenu PAR CARTE (`settings/replayPreferences.ts`).
 *
 * LA ROTATION SE FAIT DANS LE MONDE, avant la projection, et pas sur la toile. Tourner la toile
 * ferait tourner aussi les noms, les icônes et les libellés ; ici seul le TERRAIN tourne — les
 * positions, les zones, les murs, le fond — et tout ce qui est dessiné en pixels reste debout.
 *
 * SENS : un quart de tour est un quart de tour HORAIRE vu à l'écran (le nord du monde, en haut à
 * 0°, passe à droite à 90°), autour du CENTRE de la scène. Dans le plan XY monde (+Y en haut),
 * `(dx, dy)` devient `(dy, -dx)`. Les caps (degrés, 0° = +X, 90° = +Y) PERDENT donc 90° par quart
 * de tour (`shiftHeadingDeg`). C'est ICI, et dans `orientAbout`, que le sens est défini : le
 * dessin du fond (`mapBackgroundDraw`) et la table des cellules de chaleur (`cellAfterTurns`) n'en
 * sont que la traduction.
 *
 * Pas de React, pas de canvas : géométrie pure, testée dans `orientation.test.ts`.
 */
import type { ReplayBounds } from '@/lib/api/types'

import type { HeatGrid } from '../../../lib/replay/heatPaint'
import type { XY } from '../../../lib/replay/replayLogic'

/** Le nombre de quarts de tour horaires : 0, 90, 180 ou 270 degrés. */
export type QuarterTurns = 0 | 1 | 2 | 3

/** Les quatre positions, dans l'ordre où la commande les propose. */
export const QUARTER_TURNS: readonly QuarterTurns[] = [0, 1, 2, 3]

/** Les degrés d'une position (0, 90, 180, 270) — ce que la commande affiche. */
export function turnsToDegrees(turns: QuarterTurns): number {
  return turns * 90
}

/** normalizeTurns ramène un entier quelconque (négatif compris) à 0..3 ; non entier -> 0. */
export function normalizeTurns(raw: number): QuarterTurns {
  if (!Number.isInteger(raw)) return 0
  return (((raw % 4) + 4) % 4) as QuarterTurns
}

/**
 * L'ORIENTATION D'UN CADRAGE : le nombre de quarts de tour et le pivot (le centre de la scène,
 * dans le repère monde non tourné). `CanvasView.orient` la porte ; absente = 0 quart de tour.
 */
export interface ViewOrientation {
  turns: QuarterTurns
  pivot: XY
}

/** Le centre d'une emprise — le pivot de toute rotation de la carte. */
export function pivotOf(bounds: ReplayBounds): XY {
  return { x: (bounds.minX + bounds.maxX) / 2, y: (bounds.minY + bounds.maxY) / 2 }
}

/**
 * orientAbout tourne `p` de `turns` quarts de tour horaires autour de `pivot` (la définition du
 * sens). Son inverse est la rotation de `4 - turns` quarts de tour.
 */
export function orientAbout(p: XY, turns: QuarterTurns, pivot: XY): XY {
  const dx = p.x - pivot.x
  const dy = p.y - pivot.y
  switch (turns) {
    case 1:
      return { x: pivot.x + dy, y: pivot.y - dx }
    case 2:
      return { x: pivot.x - dx, y: pivot.y - dy }
    case 3:
      return { x: pivot.x - dy, y: pivot.y + dx }
    default:
      return { x: p.x, y: p.y }
  }
}

/**
 * orientBounds rend les bornes de la scène une fois tournée : même centre, largeur et hauteur
 * échangées pour un quart ou trois quarts de tour. L'altitude ne tourne pas.
 */
export function orientBounds(bounds: ReplayBounds, turns: QuarterTurns): ReplayBounds {
  if (turns === 0) return bounds
  const c = pivotOf(bounds)
  const w = bounds.maxX - bounds.minX
  const h = bounds.maxY - bounds.minY
  // Pour un nombre pair de quarts de tour l'emprise garde ses dimensions, sinon elles s'échangent.
  const halfW = (turns % 2 === 0 ? w : h) / 2
  const halfH = (turns % 2 === 0 ? h : w) / 2
  return {
    minX: c.x - halfW,
    maxX: c.x + halfW,
    minY: c.y - halfH,
    maxY: c.y + halfH,
    minZ: bounds.minZ,
    maxZ: bounds.maxZ,
  }
}

/**
 * viewOrientation construit l'orientation d'un cadrage, ou `undefined` à 0 quart de tour : le
 * cadrage non tourné ne porte aucun champ de plus, et la projection garde son chemin d'origine.
 */
export function viewOrientation(turns: QuarterTurns, scene: ReplayBounds): ViewOrientation | undefined {
  return turns === 0 ? undefined : { turns, pivot: pivotOf(scene) }
}

/** Le nombre de quarts de tour d'une orientation optionnelle (0 quand elle est absente). */
export function turnsOf(orient: ViewOrientation | undefined): QuarterTurns {
  return orient ? orient.turns : 0
}

/**
 * shiftHeadingDeg applique à un cap monde (degrés, 0° = +X, 90° = +Y) la rotation de la carte :
 * la carte tourne dans le sens horaire, le cap perd donc 90° par quart de tour (un cap qui
 * désigne le nord désigne l'est à un quart de tour).
 */
export function shiftHeadingDeg(headingDeg: number, turns: QuarterTurns): number {
  return headingDeg - turns * 90
}

/**
 * orientHeatGrid tourne une grille de chaleur — alignée sur les axes du monde — avec la carte. À
 * un multiple de quart de tour les cellules restent carrées et alignées : la grille se TRANSPOSE
 * et se retourne, sans rééchantillonnage, donc sans perdre une valeur. Rend la grille d'origine à
 * 0 quart de tour.
 */
export function orientHeatGrid(grid: HeatGrid, turns: QuarterTurns, pivot: XY): HeatGrid {
  if (turns === 0) return grid
  const { nx, ny, cell } = grid
  const maxX = grid.minX + nx * cell
  const maxY = grid.minY + ny * cell
  const a = orientAbout({ x: grid.minX, y: grid.minY }, turns, pivot)
  const b = orientAbout({ x: maxX, y: maxY }, turns, pivot)
  const odd = turns % 2 === 1
  const nnx = odd ? ny : nx
  const nny = odd ? nx : ny
  const value = new Float32Array(grid.value.length)
  for (let row = 0; row < ny; row++) {
    for (let col = 0; col < nx; col++) {
      const [c2, r2] = cellAfterTurns(col, row, nx, ny, turns)
      value[r2 * nnx + c2] = grid.value[row * nx + col]
    }
  }
  return {
    ...grid,
    nx: nnx,
    ny: nny,
    minX: Math.min(a.x, b.x),
    minY: Math.min(a.y, b.y),
    value,
  }
}

/** Où tombe la cellule (col, row) d'une grille nx x ny après `turns` quarts de tour (turns != 0). */
function cellAfterTurns(col: number, row: number, nx: number, ny: number, turns: QuarterTurns): [number, number] {
  switch (turns) {
    case 1:
      return [row, nx - 1 - col]
    case 2:
      return [nx - 1 - col, ny - 1 - row]
    case 3:
      return [ny - 1 - row, col]
    default:
      return [col, row]
  }
}
