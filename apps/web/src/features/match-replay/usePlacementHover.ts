/**
 * usePlacementHover — LE SURVOL D'UNE POSE D'ÉQUIPEMENT sur le canvas.
 *
 * POURQUOI UN HOOK À PART. Le canvas ne connaît pas ses formes : une fois peintes, ce sont des
 * pixels. Le survol se rejoue donc sur la DONNÉE (`placementAt`, pure et testée), pas sur le
 * dessin. Le poser ici garde `ReplayCanvas.tsx` — qui porte déjà une dette de taille gelée — à
 * sa longueur, comme `useSlotIdentity` avant lui (CLAUDE.md n°5).
 *
 * L'IMAGE COURANTE EST LUE DANS UNE RÉFÉRENCE, jamais dans un état React : pendant la lecture
 * elle change soixante fois par seconde, et la publier à React pour un survol coûterait tout le
 * budget d'animation. Conséquence assumée, et c'est la seule : si une pose expire SOUS un
 * pointeur immobile, son infobulle reste jusqu'au prochain mouvement. À l'arrêt — le cas où
 * l'on inspecte — la lecture est exacte.
 */
import { useCallback, useState, type PointerEvent, type RefObject } from 'react'

import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import type { PlacementView, PlacementWindowTime } from './equipmentPlacementsLayer'
import { placementAt } from './placementHitTest'
import type { XY } from './replayLogic'

/** Ce qui est survolé : la pose, l'endroit du canvas où poser son infobulle, et sa naissance. */
export interface PlacementHover {
  placement: ReplayEquipmentPlacement
  at: XY
  /**
   * L'INSTANT DE LA POSE au chronomètre du rejeu, en millisecondes (`t0` converti ici, où la
   * durée réelle d'une image est connue).
   *
   * UN INSTANT ABSOLU, JAMAIS UN ÂGE. L'image courante est lue dans une référence et l'infobulle
   * ne se recalcule qu'au mouvement du pointeur (cf. l'en-tête) : un « il y a 12 s » se
   * périmerait sous un pointeur immobile pendant la lecture, un « à 4:32 » reste vrai.
   */
  atMs: number
}

export interface PlacementHoverInput {
  placements: readonly ReplayEquipmentPlacement[]
  view: PlacementView
  /** L'image courante, telle que la boucle de lecture la tient. */
  frameRef: RefObject<number>
  /**
   * L'axe de temps du rejeu (`frameMs` = durée RÉELLE d'une image, `frames` = leur nombre) :
   * il borne la fenêtre d'affichage d'une pose (cf. `placementEndFrame`), et il sert aussi au
   * traqueur, qui ne montre son impulsion que quelques centaines de millisecondes — on ne
   * survole pas ce qui n'est plus dessiné (cf. `placementShows`).
   */
  time: PlacementWindowTime
  /** Faux quand le calque est éteint : rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  showUnnamed: boolean
  /**
   * Les objets de PUISSANCE lâchés se survolent-ils ? La valeur arrive DÉJÀ croisée avec la
   * garde de mode (jamais en Fiesta) : le survol suit le dessin, jamais l'inverse.
   */
  showDropped: boolean
}

export interface PlacementHoverHandlers {
  hover: PlacementHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

export function usePlacementHover({
  placements,
  view,
  time,
  frameRef,
  enabled,
  showUnnamed,
  showDropped,
}: PlacementHoverInput): PlacementHoverHandlers {
  const [hover, setHover] = useState<PlacementHover | null>(null)

  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || placements.length === 0 || view.width === 0) {
        setHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      // Le contexte dessine en pixels CSS (la densité est absorbée par sa transformation) ;
      // le rapport ne vaut 1 que si la mise en page ne remet pas le canevas à l'échelle — on
      // le calcule plutôt que de le supposer.
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const found = placementAt(
        placements,
        view,
        { frame: frameRef.current, showUnnamed, showDropped, frameMs: time.frameMs, frames: time.frames },
        at,
      )
      setHover((prev) => {
        if (!found) return prev === null ? prev : null
        if (prev && prev.placement === found && prev.at.x === at.x && prev.at.y === at.y) return prev
        return { placement: found, at, atMs: found.t0 * time.frameMs }
      })
    },
    [enabled, placements, view, frameRef, showUnnamed, showDropped, time.frameMs, time.frames],
  )

  const onPointerLeave = useCallback(() => {
    setHover((prev) => (prev === null ? prev : null))
  }, [])

  return { hover, onPointerMove, onPointerLeave }
}
