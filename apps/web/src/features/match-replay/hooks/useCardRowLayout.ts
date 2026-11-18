/**
 * useCardRowLayout — COMMENT LES ÉQUIPES SE RANGENT, au Cinéma et au Plein cadre : en rangée au-dessus du
 * plan, ou dans la colonne ; et OÙ EST LE SCORE, aux trois cadres (le Standard mesure son corps pour cette seule
 * question). Il mesure le corps du cadre et applique la règle pure de `model/rowLayout.ts`
 * (`dispositionAvecMemoire`) aux camps et aux places du film. Il se souvient de sa dernière décision : la rangée
 * et l'emplacement du score ne basculent qu'au-delà de la marge `MARGE_BASCULE_PX` (la largeur d'une barre de
 * défilement qui apparaît ne fait pas clignoter la page).
 *
 * LA LARGEUR EST CELLE DU CORPS DU CADRE, bord à bord, marges latérales comprises : c'est la grandeur que
 * `largeurRequiseRangee` compare (elle compte ses deux marges). Le hook rend `bodyRef`, que le cadre pose sur
 * le corps (champ `bodyRef` de `ReplayFrameZoneContent`) ; la mesure est lue avant la première peinture, puis
 * suivie par un `ResizeObserver` (le redimensionnement de la fenêtre change la largeur du corps). Tant que le
 * corps n'est pas mesuré, ou hors du Cinéma et du Plein cadre (`canPlaceCardRow`), la disposition est la
 * colonne.
 *
 * Le hook ne rend rien : il ne monte ni ne démonte aucun nœud, la page distribue sa décision (disposition,
 * emplacement du score, qui dépend aussi de la largeur). Le calcul des places par camp est mémoïsé sur le document et les joueurs : la page se
 * rend à chaque image de lecture publiée, pas la règle.
 */
import { useLayoutEffect, useMemo, useState } from 'react'

import type { ReplayFrameView } from '../ui/frames/replayFrameTypes'
import { canPlaceCardRow, type ReplayFrame } from '../model/replayFrame'
import { dispositionAvecMemoire, emplacementScoreDe, planFeedPosed, type DispositionEquipes } from '../model/rowLayout'
import { placesParCamp } from '../model/seatLogic'
import type { EmplacementScore } from '../model/cotesBandeauScore'

export interface CardRowLayout {
  /** À poser sur le corps du cadre : c'est lui qu'on mesure. */
  bodyRef: (el: HTMLElement | null) => void
  /** Comment les équipes se rangent dans ce cadre, à cette largeur. */
  disposition: DispositionEquipes
  /** Où le bandeau de score est posé : l'en-tête, la rangée ou la toile (`emplacementScoreDe`). */
  scorePlacement: EmplacementScore
  /** Le fil des éliminations est posé sur le plan, et la colonne ne le montre plus (`planFeedPosed`). */
  planFeedPosed: boolean
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

/** Ce que la page a posé à la dernière décision : la règle s'en sert pour ne pas basculer sur un pixel. */
interface DecisionPrecedente {
  disposition: DispositionEquipes
  scorePlacement: EmplacementScore
}

export function useCardRowLayout(frame: ReplayFrame, view: ReplayFrameView | null): CardRowLayout {
  const { width, ref } = useBodyWidth()
  const players = view?.model.players
  const doc = view?.doc
  const places = useMemo(() => (players && doc ? placesParCamp(players, doc) : []), [players, doc])
  // LA MÉMOIRE DE LA DÉCISION est calculée au rendu (jamais depuis un effet, qui ferait peindre une image à la
  // mauvaise disposition) : la règle a besoin de savoir où la page en était pour appliquer la marge de bascule.
  const [precedente, setPrecedente] = useState<DecisionPrecedente | null>(null)
  const disposition: DispositionEquipes = canPlaceCardRow(frame)
    ? dispositionAvecMemoire(
        { camps: places.length, placesParCamp: places, largeurCorps: width },
        precedente?.disposition ?? null,
      )
    : 'colonne'
  const scorePlacement = emplacementScoreDe(frame, disposition, width, precedente?.scorePlacement ?? null)
  // Une décision prise sans mesure (largeur nulle : l'en-tête est présumé assez large) n'est pas retenue : la
  // première mesure décide au seuil nu, sans marge en faveur d'une présomption.
  if (width > 0 && (precedente?.disposition !== disposition || precedente.scorePlacement !== scorePlacement)) {
    setPrecedente({ disposition, scorePlacement })
  }
  return {
    bodyRef: ref,
    disposition,
    scorePlacement,
    planFeedPosed: planFeedPosed(frame, disposition),
  }
}
