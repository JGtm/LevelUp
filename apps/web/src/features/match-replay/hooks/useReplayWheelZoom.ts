/**
 * useReplayWheelZoom — LA MOLETTE, calée sur les paliers du zoom.
 *
 * # POURQUOI ELLE NE FAIT PAS DE ZOOM CONTINU
 *
 * Un zoom continu changerait le cadrage à chaque cran, donc recuirait les quatre calques
 * statiques (le sol et ses ~45 000 cellules) des dizaines de fois par geste. Calée sur les
 * paliers, la molette devient simplement une autre façon d'appuyer sur `+` / `−` : même chemin,
 * même rebornage, même coût. Ce qui interdit qu'un « zoom à la molette » et un « zoom au bouton »
 * finissent par ne plus donner le même résultat.
 *
 * # ELLE GROSSIT VERS LE CURSEUR
 *
 * Le point du monde sous le pointeur reste IMMOBILE à l'écran (`canvasToWorld` puis `zoomAt`).
 * C'est la différence entre une molette qui attrape ce qu'on vise et une molette qui le chasse —
 * et sur une carte, la seconde est franchement désagréable.
 *
 * # DEUX PIÈGES, ET ILS SE VOIENT MAL
 *
 * 1. L'ÉCOUTEUR EST POSÉ À LA MAIN, EN NON PASSIF. React attache `onWheel` en écouteur PASSIF :
 *    `preventDefault()` y est ignoré (avec un avertissement en console), et la page défilerait
 *    sous la carte pendant qu'on zoome. Il n'y a pas de prop React pour lever ça — d'où
 *    `addEventListener(..., { passive: false })`.
 * 2. L'ACCUMULATEUR N'EST PAS UN CONFORT. Un pavé tactile émet des dizaines d'événements de
 *    quelques pixels par geste ; un cran par événement traverserait toute l'échelle en un
 *    mouvement. On additionne donc jusqu'à `WHEEL_STEP` avant de bouger d'un palier.
 *
 * L'état vif (`zoom`, `view`) est lu par RÉFÉRENCE et non capturé : sans cela, l'écouteur se
 * réabonnerait à chaque image de lecture — le cadrage change soixante fois par seconde pendant
 * qu'on joue.
 */
import { useEffect, useRef, type RefObject } from 'react'

import { canvasToWorld } from '../model/replayLogic'
import { type CanvasView } from '../model/replayView'
import type { ReplayZoom } from './useReplayZoom'

/** Le delta cumulé qui vaut un palier. Calé sur un cran de molette classique (~100). */
export const WHEEL_STEP = 60

export function useReplayWheelZoom(
  canvasRef: RefObject<HTMLCanvasElement | null>,
  zoom: ReplayZoom,
  view: CanvasView,
): void {
  // LA RÉFÉRENCE S'ÉCRIT DANS UN EFFET, PAS PENDANT LE RENDU. React interdit d'y toucher au
  // rendu (une valeur écrite là peut être perdue si le rendu est abandonné), et l'outillage le
  // dit. L'effet suffit : l'écouteur ne lit `live` qu'au moment d'un cran de molette, donc
  // toujours après la peinture.
  const live = useRef({ zoom, view })
  useEffect(() => {
    live.current = { zoom, view }
  }, [zoom, view])

  useEffect(() => {
    const el = canvasRef.current
    if (!el) return
    let acc = 0

    function onWheel(e: WheelEvent) {
      if (!el) return
      // La page ne défile pas sous la carte pendant qu'on zoome (cf. piège 1).
      e.preventDefault()
      acc += e.deltaY
      while (Math.abs(acc) >= WHEEL_STEP) {
        const dir = acc > 0 ? -1 : 1
        acc -= Math.sign(acc) * WHEEL_STEP
        const r = el.getBoundingClientRect()
        const { zoom: z, view: v } = live.current
        // LE POINT VISÉ SE RELIT À CHAQUE CRAN, et pas une fois pour le geste : après un palier
        // la projection a changé, donc les mêmes coordonnées d'écran désignent un autre endroit
        // du monde. Le relire est ce qui garde le curseur ancré sur ce qu'il montrait.
        z.zoomAt(dir, canvasToWorld(
          { x: e.clientX - r.left, y: e.clientY - r.top },
          v.bounds,
          v.width,
          v.height,
          v.pad,
        ))
      }
    }

    el.addEventListener('wheel', onWheel, { passive: false })
    return () => el.removeEventListener('wheel', onWheel)
  }, [canvasRef])
}
