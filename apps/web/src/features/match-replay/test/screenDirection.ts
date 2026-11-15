/**
 * screenDirection.ts — L'ORACLE DES TESTS DE CAPS : l'angle écran (radians canevas) du
 * déplacement de 3 m le long d'un cap monde, par le cadrage — donc avec la rotation de la carte.
 * Il ne réécrit aucune formule d'angle : il projette deux points.
 */
import type { CanvasView } from '../model/replayView'
import { projectTo } from '../model/replayView'

/** L'angle écran du déplacement de 3 m depuis `from` le long du cap monde `headingDeg`. */
export function screenDirection(view: CanvasView, headingDeg: number, from: { x: number; y: number }): number {
  const rad = (headingDeg * Math.PI) / 180
  const a = projectTo(view, from)
  const b = projectTo(view, { x: from.x + 3 * Math.cos(rad), y: from.y + 3 * Math.sin(rad) })
  return Math.atan2(b.y - a.y, b.x - a.x)
}

/** Vrai quand deux angles désignent la même direction (modulo un tour). */
export function sameDirection(a: number, b: number): boolean {
  return Math.abs(Math.atan2(Math.sin(a - b), Math.cos(a - b))) < 1e-9
}
