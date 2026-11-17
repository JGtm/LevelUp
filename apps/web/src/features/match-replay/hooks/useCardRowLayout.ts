/**
 * useCardRowLayout — COMMENT LES ÉQUIPES SE RANGENT, au Cinéma et au Plein cadre : en rangée au-dessus du
 * plan, ou dans la colonne. Il mesure le corps du cadre et applique la règle pure de `model/rowLayout.ts`
 * (`dispositionEquipes`) aux camps et aux places du film.
 *
 * LA LARGEUR EST CELLE DU CORPS DU CADRE, bord à bord, marges latérales comprises : c'est la grandeur que
 * `largeurRequiseRangee` compare (elle compte ses deux marges). Le hook rend `bodyRef`, que le cadre pose sur
 * le corps (champ `bodyRef` de `ReplayFrameZoneContent`) ; la mesure est lue avant la première peinture, puis
 * suivie par un `ResizeObserver` (le redimensionnement de la fenêtre change la largeur du corps). Tant que le
 * corps n'est pas mesuré, ou hors du Cinéma et du Plein cadre (`canPlaceCardRow`), la disposition est la
 * colonne : le score reste sur la toile.
 *
 * Le hook ne rend rien : il ne monte ni ne démonte aucun nœud, la page distribue sa décision (disposition,
 * emplacement du score). Le calcul des places par camp est mémoïsé sur le document et les joueurs : la page se
 * rend à chaque image de lecture publiée, pas la règle.
 */
import { useLayoutEffect, useMemo, useState } from 'react'

import type { ReplayFrameView } from '../ui/frames/replayFrameTypes'
import { canPlaceCardRow, type ReplayFrame } from '../model/replayFrame'
import { dispositionEquipes, emplacementScoreDe, type DispositionEquipes } from '../model/rowLayout'
import { placesParCamp } from '../model/seatLogic'
import type { EmplacementScore } from '../model/cotesBandeauScore'

export interface CardRowLayout {
  /** À poser sur le corps du cadre : c'est lui qu'on mesure. */
  bodyRef: (el: HTMLElement | null) => void
  /** Comment les équipes se rangent dans ce cadre, à cette largeur. */
  disposition: DispositionEquipes
  /** Où le bandeau de score est posé ; `null` = pas sur la toile (en-tête du Standard, ou rangée). */
  scorePlacement: EmplacementScore | null
}

/** La largeur du corps en pixels entiers, relue à chaque changement de taille ; 0 tant qu'il n'est pas mesuré. */
function useBodyWidth(): { width: number; ref: (el: HTMLElement | null) => void } {
  const [body, setBody] = useState<HTMLElement | null>(null)
  const [width, setWidth] = useState(0)
  useLayoutEffect(() => {
    if (!body) return
    const read = () => setWidth(Math.floor(body.getBoundingClientRect().width))
    read()
    // L'observateur est optionnel (jsdom ne le fournit pas) : sans lui, la mesure initiale reste.
    if (typeof ResizeObserver === 'undefined') return
    const observer = new ResizeObserver(read)
    observer.observe(body)
    return () => observer.disconnect()
  }, [body])
  return { width, ref: setBody }
}

export function useCardRowLayout(frame: ReplayFrame, view: ReplayFrameView | null): CardRowLayout {
  const { width, ref } = useBodyWidth()
  const players = view?.model.players
  const doc = view?.doc
  const places = useMemo(() => (players && doc ? placesParCamp(players, doc) : []), [players, doc])
  const disposition: DispositionEquipes = canPlaceCardRow(frame)
    ? dispositionEquipes({ camps: places.length, placesParCamp: places, largeurCorps: width })
    : 'colonne'
  return { bodyRef: ref, disposition, scorePlacement: emplacementScoreDe(frame, disposition) }
}
