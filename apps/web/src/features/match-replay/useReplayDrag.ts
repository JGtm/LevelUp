/**
 * useReplayDrag — LE GLISSER : attraper la carte et la déplacer.
 *
 * # CE QU'IL CONVERTIT, ET POURQUOI C'EST LUI QUI LE FAIT
 *
 * Le pointeur parle en PIXELS, le cadrage en UNITÉS MONDE. La conversion vit ici, et nulle part
 * ailleurs : `useReplayZoom` ne connaît pas la toile, et c'est ce qui le rend testable sans
 * canvas. `canvasScale` donne les pixels par unité monde ; le reste est une division.
 *
 * # LE SENS EST CELUI DE LA MAIN, PAS CELUI DE LA CAMÉRA
 *
 * Tirer vers la droite doit amener vers soi ce qui était à gauche — donc déplacer la FENÊTRE
 * vers la gauche. D'où le signe négatif en X. En Y s'ajoute l'inversion d'axe : le monde a son Y
 * vers le haut, la toile vers le bas.
 *
 * # PENDANT LE GESTE, LES CALQUES NE CUISENT PAS
 *
 * C'est tout l'intérêt de `dragging`. Un glisser change le cadrage à chaque mouvement de
 * pointeur ; recuire les quatre calques statiques à cette cadence est hors de question (le sol
 * reconstruit fait ~45 000 cellules, 10 à 45 ms par cuisson). Mais un glisser est une
 * TRANSLATION PURE : le dessin recopie donc le calque déjà cuit avec un décalage
 * (`layerOffset`), ce qui est exact et gratuit. La cuisson reprend au relâchement.
 *
 * Le seul artefact est une bande non peinte au bord d'attaque pendant le geste, sur les calques
 * cuits — pas sur le fond de carte, qui est dessiné directement et suit donc parfaitement. Elle
 * disparaît au relâchement. C'est le prix, assumé, de ne rien coûter en mémoire : la solution
 * sans bande demanderait de cuire une fenêtre élargie, donc de la mémoire en permanence pour un
 * geste occasionnel.
 *
 * # LA CAPTURE DU POINTEUR N'EST PAS UN DÉTAIL
 *
 * `setPointerCapture` fait suivre les événements même quand le curseur sort de la toile. Sans
 * elle, un glisser qui déborde du terrain — c'est-à-dire le cas normal quand on tire vers un
 * bord — se terminerait sans `pointerup`, et la carte resterait collée au curseur.
 */
import { useCallback, useRef, useState, type PointerEvent } from 'react'

import { canvasScale } from './replayLogic'
import type { CanvasView } from './replayDraw'
import type { ReplayZoom } from './useReplayZoom'

export interface ReplayDrag {
  /** `true` tant que le geste dure : le dessin gèle alors la cuisson des calques. */
  dragging: boolean
  onPointerDown: (e: PointerEvent<HTMLCanvasElement>) => void
  onPointerMove: (e: PointerEvent<HTMLCanvasElement>) => void
  onPointerUp: (e: PointerEvent<HTMLCanvasElement>) => void
}

export function useReplayDrag(zoom: ReplayZoom, view: CanvasView): ReplayDrag {
  const [dragging, setDragging] = useState(false)
  const last = useRef<{ x: number; y: number } | null>(null)

  const onPointerDown = useCallback(
    (e: PointerEvent<HTMLCanvasElement>) => {
      // RIEN À DÉPLACER À 1x : la fenêtre vaut la scène. On laisse alors le clic tranquille —
      // il sert au survol des calques, qui partage la même balise.
      if (!zoom.canPan || e.button !== 0) return
      last.current = { x: e.clientX, y: e.clientY }
      setDragging(true)
      e.currentTarget.setPointerCapture(e.pointerId)
    },
    [zoom.canPan],
  )

  const onPointerMove = useCallback(
    (e: PointerEvent<HTMLCanvasElement>) => {
      const from = last.current
      if (!from) return
      const k = canvasScale(view.bounds, view.width, view.height, view.pad)
      if (!(k > 0)) return
      last.current = { x: e.clientX, y: e.clientY }
      zoom.panBy(-(e.clientX - from.x) / k, (e.clientY - from.y) / k)
    },
    [zoom, view],
  )

  const onPointerUp = useCallback((e: PointerEvent<HTMLCanvasElement>) => {
    if (!last.current) return
    last.current = null
    setDragging(false)
    // `hasPointerCapture` avant de relâcher : le navigateur a pu la rendre lui-même (le pointeur
    // a disparu, l'onglet a perdu le focus), et relâcher une capture qu'on n'a plus lève.
    if (e.currentTarget.hasPointerCapture?.(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId)
    }
  }, [])

  return { dragging, onPointerDown, onPointerMove, onPointerUp }
}
