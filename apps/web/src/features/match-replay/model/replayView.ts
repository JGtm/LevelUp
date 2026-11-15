/**
 * replayView — LE CADRAGE DU REJEU, ET CE QU'IL SAIT FAIRE.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, K3). Le cadrage — les bornes du monde, la
 * taille de la toile, la marge — était un objet que RIEN ne savait projeter : son type était
 * redéclaré NEUF fois à l'identique dans la feature, et le passage monde → canvas se réécrivait
 * à chaque site en dépaquetant les quatre champs à la main :
 *
 *     worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
 *
 * Vingt-neuf fois, dans vingt-deux fichiers. Chaque nouvelle écriture était une occasion de se
 * tromper d'ordre entre `width` et `height` — deux nombres du même type, que le compilateur ne
 * peut pas départager — et la faute se serait vue non pas en erreur, mais en calque étiré.
 *
 * LE CADRAGE SAIT DÉSORMAIS PROJETER. `projectTo(view, p)` et `scaleOf(view)` prennent le
 * cadrage ENTIER : il n'y a plus d'ordre à retenir, et le type ne se déclare qu'ici.
 *
 * UNE SEULE FORME, ET PAS DE PROJECTEUR PRÉ-LIÉ. Les calques qui projettent en boucle gardent
 * un raccourci local (`const px = (p: XY) => projectTo(view, p)`) : c'est de la brièveté au
 * point d'appel, pas une seconde règle de projection — la règle, elle, n'est écrite qu'ici.
 *
 * IL NE PEUT PAS VIVRE DANS `replayLogic`, et c'est structurel : `worldToCanvas` y est défini,
 * donc l'y appeler ferait un cycle. `layerOffset` (replayLogic) est le seul site qui garde
 * l'écriture longue, pour cette raison-là et pour elle seule.
 */
import type { ReplayBounds } from '@/lib/api/types'

import { canvasScale, layerOffset, worldToCanvas, type XY } from '../../../lib/replay/replayLogic'
import { orientAbout, turnsOf, type QuarterTurns, type ViewOrientation } from '@/lib/replay/orientation'

/**
 * Le CADRAGE : quelle portion du monde on montre, dans quelle toile, avec quelle marge.
 *
 * C'est la seule déclaration de ce type dans la feature — les huit copies locales (dont
 * plusieurs écrivaient `bounds` en littéral plutôt qu'en `ReplayBounds`) l'importent.
 */
export interface CanvasView {
  /**
   * Les bornes de la fenêtre, dans le repère de la carte APRÈS rotation (`orient`) : c'est le
   * repère dans lequel le zoom, le déplacement et le glisser raisonnent.
   */
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
  /**
   * L'orientation de la carte (quarts de tour autour du centre de la scène). Absente = la carte
   * n'est pas tournée. `projectTo` la applique AVANT de projeter : tout ce qui projette par le
   * cadrage la suit sans le savoir.
   */
  orient?: ViewOrientation
}

/**
 * projectTo projette une position MONDE dans la toile, selon le cadrage — la rotation de la
 * carte (`view.orient`) comprise.
 *
 * Y EST INVERSÉ (le monde a +Y vers le haut, la toile vers le bas), l'ajustement préserve le
 * ratio et centre — tout cela vit dans `worldToCanvas`, que cette fonction ne fait qu'alimenter
 * avec le cadrage entier.
 */
export function projectTo(view: CanvasView, p: XY): XY {
  const o = view.orient
  return worldToCanvas(o ? orientAbout(p, o.turns, o.pivot) : p, view.bounds, view.width, view.height, view.pad)
}

/**
 * projectFrameTo projette une position DÉJÀ DANS LE REPÈRE DE LA CARTE TOURNÉE (celui de
 * `view.bounds`) : sans rotation de plus. Pour ce qui a été tourné en amont — la grille de
 * chaleur, que `orientHeatGrid` transpose avec la carte —, jamais pour une position du film.
 */
export function projectFrameTo(view: CanvasView, p: XY): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

/** Le nombre de quarts de tour d'un cadrage : 0 quand la carte n'est pas tournée. */
export function turnsOfView(view: Pick<CanvasView, 'orient'>): QuarterTurns {
  return turnsOf(view.orient)
}

/**
 * scaleOf rend le facteur MÈTRES → PIXELS du cadrage.
 *
 * C'est lui qui convertit une grandeur du monde (un rayon d'explosion, la maille d'une carte de
 * chaleur, la portée d'un capteur) en pixels de la toile. Même raison d'être que `projectTo` :
 * il prenait les quatre champs un par un, sept fois.
 */
export function scaleOf(view: CanvasView): number {
  return canvasScale(view.bounds, view.width, view.height, view.pad)
}

/**
 * FOLLOW_COOK_MARGIN — la part de la toile (par axe) dont un calque cuit peut être décalé avant
 * qu'on le recuise : au-delà, la bande non peinte au bord d'attaque deviendrait gênante. Un
 * glisser n'a pas de marge (le geste est bref) ; Suivre dure, il en a une.
 */
export const FOLLOW_COOK_MARGIN = 0.1

/**
 * cookIsReusable dit si un calque cuit pour `cooked` se recopie encore tel quel, décalé, pour
 * `view` : même toile, même ÉCHELLE (un zoom change l'échelle, et un décalage ne replace plus une
 * image cuite pour une autre), et un décalage qui reste dans la marge. C'est ce qui laisse Suivre
 * cadrer un joueur à chaque image sans recuire les calques statiques à chaque image.
 */
export function cookIsReusable(cooked: CanvasView, view: CanvasView): boolean {
  if (cooked.width !== view.width || cooked.height !== view.height || cooked.pad !== view.pad) return false
  // Un calque cuit pour une autre orientation (ou un autre pivot) n'est pas le même dessin décalé.
  if (turnsOfView(cooked) !== turnsOfView(view)) return false
  if (cooked.orient?.pivot.x !== view.orient?.pivot.x || cooked.orient?.pivot.y !== view.orient?.pivot.y) return false
  const before = scaleOf(cooked)
  const after = scaleOf(view)
  if (!(before > 0) || !(after > 0) || Math.abs(before - after) > before * 1e-9) return false
  const off = layerOffset(cooked, view)
  return Math.abs(off.x) <= FOLLOW_COOK_MARGIN * view.width && Math.abs(off.y) <= FOLLOW_COOK_MARGIN * view.height
}
