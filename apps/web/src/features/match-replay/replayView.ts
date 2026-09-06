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

import { canvasScale, worldToCanvas, type XY } from './replayLogic'

/**
 * Le CADRAGE : quelle portion du monde on montre, dans quelle toile, avec quelle marge.
 *
 * C'est la seule déclaration de ce type dans la feature — les huit copies locales (dont
 * plusieurs écrivaient `bounds` en littéral plutôt qu'en `ReplayBounds`) l'importent.
 */
export interface CanvasView {
  bounds: ReplayBounds
  width: number
  height: number
  pad: number
}

/**
 * projectTo projette une position MONDE dans la toile, selon le cadrage.
 *
 * Y EST INVERSÉ (le monde a +Y vers le haut, la toile vers le bas), l'ajustement préserve le
 * ratio et centre — tout cela vit dans `worldToCanvas`, que cette fonction ne fait qu'alimenter
 * avec le cadrage entier.
 */
export function projectTo(view: CanvasView, p: XY): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
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
