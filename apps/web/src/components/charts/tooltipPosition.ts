/**
 * tooltipPosition.ts — placement des infobulles ECharts (source unique).
 *
 * CONTRAT. L'infobulle peut sortir du cadre du graphe (elle est ajoutée au `body`, hors de
 * la carte qui rogne) mais ne sort JAMAIS de la fenêtre, et ne recouvre pas la marque
 * survolée tant qu'une place libre existe. Retours utilisateur 2026-10-17 : « elles se
 * superposent sur les barres ou quittent l'écran ».
 *
 * Deux couches :
 *  - `placeTooltip` : fonction PURE en coordonnées de la fenêtre (testée aux bords) ;
 *  - `chartTooltipPosition` : adaptateur du `tooltip.position` d'ECharts. ECharts donne des
 *    coordonnées LOCALES au graphe, et l'origine du graphe dans la fenêtre n'est exposée
 *    nulle part : on la déduit de la dernière position du pointeur (écouteur global posé à
 *    l'import) moins le point local que fournit ECharts pour ce même pointeur.
 */

/** Rectangle en pixels (même repère que ses voisins : fenêtre pour `placeTooltip`). */
export interface Box {
  left: number
  top: number
  width: number
  height: number
}

export interface TooltipPlacementInput {
  /** Pointeur, coordonnées fenêtre. */
  pointer: readonly [number, number]
  /** Taille de l'infobulle (largeur, hauteur). */
  size: readonly [number, number]
  /** Taille de la fenêtre (largeur, hauteur). */
  viewport: readonly [number, number]
  /** Marque survolée en coordonnées fenêtre ; absente pour une infobulle d'axe. */
  mark?: Box | null
}

/** Écart entre l'infobulle et le pointeur / la marque. */
export const TOOLTIP_GAP = 12
/** Marge minimale entre l'infobulle et le bord de la fenêtre. */
export const TOOLTIP_MARGIN = 8

function clamp(v: number, min: number, max: number): number {
  return Math.max(min, Math.min(v, Math.max(min, max)))
}

/**
 * Coin haut-gauche de l'infobulle, en coordonnées fenêtre.
 *
 * Avec une marque : au-dessus / au-dessous d'une marque haute (barre verticale), à droite /
 * à gauche d'une marque large (barre horizontale), c'est-à-dire du côté où il n'y a pas
 * d'autres marques ; repli sur l'autre axe. Sans marque (axe) ou sans place libre : en
 * diagonale du pointeur, côté où elle tient. Le résultat est toujours borné à la fenêtre.
 */
export function placeTooltip(input: TooltipPlacementInput): [number, number] {
  const { pointer, size, viewport, mark } = input
  const [px, py] = pointer
  const [w, h] = size
  const [vw, vh] = viewport
  const minX = TOOLTIP_MARGIN
  const minY = TOOLTIP_MARGIN
  const maxX = vw - w - TOOLTIP_MARGIN
  const maxY = vh - h - TOOLTIP_MARGIN
  const fitsX = (x: number) => x >= minX && x <= maxX
  const fitsY = (y: number) => y >= minY && y <= maxY

  if (mark) {
    const centredX = px - w / 2
    const centredY = py - h / 2
    const above = mark.top - TOOLTIP_GAP - h
    const below = mark.top + mark.height + TOOLTIP_GAP
    const right = mark.left + mark.width + TOOLTIP_GAP
    const left = mark.left - TOOLTIP_GAP - w
    const vertical = (): [number, number] | null => {
      if (fitsY(above)) return [clamp(centredX, minX, maxX), above]
      if (fitsY(below)) return [clamp(centredX, minX, maxX), below]
      return null
    }
    const horizontal = (): [number, number] | null => {
      if (fitsX(right)) return [right, clamp(centredY, minY, maxY)]
      if (fitsX(left)) return [left, clamp(centredY, minY, maxY)]
      return null
    }
    const tall = mark.height >= mark.width
    const placed = tall ? (vertical() ?? horizontal()) : (horizontal() ?? vertical())
    if (placed) return placed
  }

  const x = fitsX(px + TOOLTIP_GAP) ? px + TOOLTIP_GAP : px - TOOLTIP_GAP - w
  const y = fitsY(py + TOOLTIP_GAP) ? py + TOOLTIP_GAP : py - TOOLTIP_GAP - h
  return [clamp(x, minX, maxX), clamp(y, minY, maxY)]
}

// --- Suivi du pointeur ---------------------------------------------------------------

let lastPointer: [number, number] | null = null

function rememberPointer(e: PointerEvent | MouseEvent): void {
  lastPointer = [e.clientX, e.clientY]
}

if (typeof window !== 'undefined') {
  // Phase de capture : le pointeur est à jour AVANT que le graphe ne réagisse au même évènement.
  window.addEventListener('pointermove', rememberPointer, { capture: true, passive: true })
  window.addEventListener('pointerdown', rememberPointer, { capture: true, passive: true })
  window.addEventListener('mousemove', rememberPointer, { capture: true, passive: true })
}

/** Pointeur mémorisé (coordonnées fenêtre), ou null avant tout mouvement. */
export function lastPointerPosition(): readonly [number, number] | null {
  return lastPointer
}

interface EChartsPositionSize {
  contentSize: [number, number]
}
interface ZrRectLike {
  x: number
  y: number
  width: number
  height: number
}

/**
 * `tooltip.position` d'ECharts : à poser avec `appendToBody: true` (coordonnées retournées
 * en repère local du graphe, converties par ECharts vers le `body`). Retourne `undefined`
 * si le pointeur est inconnu (placement par défaut d'ECharts).
 */
export function chartTooltipPosition(
  point: [number, number],
  _params: unknown,
  _dom: unknown,
  rect: ZrRectLike | null | undefined,
  size: EChartsPositionSize,
): [number, number] | undefined {
  if (typeof window === 'undefined' || !lastPointer) return undefined
  const [ox, oy] = [lastPointer[0] - point[0], lastPointer[1] - point[1]]
  const root = document.documentElement
  const [x, y] = placeTooltip({
    pointer: lastPointer,
    size: size.contentSize,
    viewport: [root.clientWidth || window.innerWidth, root.clientHeight || window.innerHeight],
    mark: rect ? { left: rect.x + ox, top: rect.y + oy, width: rect.width, height: rect.height } : null,
  })
  return [x - ox, y - oy]
}
