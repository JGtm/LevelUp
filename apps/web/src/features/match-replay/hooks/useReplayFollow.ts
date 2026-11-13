/**
 * useReplayFollow — « SUIVRE » : la carte se cadre sur un joueur à chaque image (panneau de
 * joueur, lot 4 du plan « maquettes du rejeu »).
 *
 * UNE BOUCLE D'IMAGES D'ÉCRAN, PAS UN ABONNEMENT AU TEMPS. La cellule de dessin du canvas
 * (`frameRef`) est la source de « quelle image on montre », fractionnaire en lecture : la boucle la
 * relit à chaque image d'écran, qu'elle avance par la lecture, par un saut de frise ou pas à pas.
 * Elle n'écrit jamais le temps — Suivre ne change que le cadrage —, et ne vit que tant que Suivre
 * est allumé : éteint, rien ne tourne.
 *
 * LE CADRAGE PASSE PAR `zoom.focusOn`, l'action du programme : elle monte au palier minimal
 * (`FOLLOW_MIN_LEVEL`) sans jamais descendre un palier déjà plus haut, et ne compte pas comme un
 * geste de la main. C'est la main, elle, qui arrête le suivi : `useReplayView` prévient la page
 * (`onManualMove`) au premier geste de zoom ou de déplacement, et la page éteint Suivre.
 *
 * UN JOUEUR IMMOBILE NE REDEMANDE RIEN : le cadrage ne s'écrit que si la position a changé depuis
 * la dernière demande, pour ne pas redessiner la scène à chaque image d'écran d'un joueur arrêté.
 * SANS POSITION (mort depuis plus que la fenêtre après-mort, lacune, avant sa première vie), la
 * carte reste où elle est — ni saut, ni position devinée (`followLogic`).
 */
import { useEffect, useMemo, useRef } from 'react'

import { buildEmbarkedPosAt } from '../model/carrierPosition'
import { FOLLOW_MIN_LEVEL, followPositionAt, sameSpot } from '../model/followLogic'
import { deathWindowFrames } from '../model/livesPosition'
import type { XY } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import type { QuarterTurns } from '../model/orientation'
import type { ReplayZoom } from './useReplayZoom'

export interface ReplayFollowOptions {
  doc: ReplayDocumentReady
  /** Le joueur suivi ; `null` = personne (le panneau est fermé ou son joueur n'est plus au roster). */
  player: ReplayPlayer | null
  /** Suivre est-il allumé ? */
  enabled: boolean
  /** La cellule de dessin du canvas : l'image montrée, fractionnaire en lecture. Lue, jamais écrite. */
  frameRef: { readonly current: number }
  zoom: ReplayZoom
  /**
   * L'orientation de la carte : un changement d'orientation ramène le zoom à la carte entière, et le
   * suivi doit alors recadrer le joueur sur sa position COURANTE sans attendre qu'il bouge. Elle ne
   * sert que de déclencheur (la conversion monde -> carte tournée est faite par le zoom).
   */
  turns?: QuarterTurns
}

export function useReplayFollow({ doc, player, enabled, frameRef, zoom, turns = 0 }: ReplayFollowOptions): void {
  const embarkedAt = useMemo(() => buildEmbarkedPosAt(doc.vehicles), [doc.vehicles])
  const deathFrames = useMemo(() => deathWindowFrames(doc), [doc])
  // L'action de cadrage change de visage à chaque palier : la boucle lit la dernière, sans se relancer.
  const focusOn = useRef(zoom.focusOn)
  useEffect(() => {
    focusOn.current = zoom.focusOn
  })

  useEffect(() => {
    if (!enabled || !player) return
    let raf = 0
    let last: XY | null = null
    const step = () => {
      const at = followPositionAt(player, frameRef.current, embarkedAt, deathFrames)
      if (at && !sameSpot(at, last)) {
        last = at
        focusOn.current(at, FOLLOW_MIN_LEVEL)
      }
      raf = requestAnimationFrame(step)
    }
    step()
    return () => cancelAnimationFrame(raf)
    // `turns` relance la boucle : `last` repart de zéro, donc la position courante est redemandée.
  }, [enabled, player, frameRef, embarkedAt, deathFrames, turns])
}
