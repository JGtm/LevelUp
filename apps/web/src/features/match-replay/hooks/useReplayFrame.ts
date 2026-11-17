/**
 * useReplayFrame — LE CADRE DE LA PAGE DU REJEU (Standard, Cinéma, Plein cadre), tenu par la page.
 *
 * Le hook ne porte que l'ÉTAT : `ReplayFrameState` et ses transitions viennent du modèle pur
 * (`model/replayFrame.ts`), jamais redites ici. Il ne lit aucune touche : les raccourcis
 * (`useReplayShortcuts`) et les boutons envoient des événements nommés, et la sortie par Échap
 * vit avec le plein écran (`useReplayFullscreen`), qui sait quelles couches sont ouvertes.
 *
 * # UNE MANIÈRE DE REGARDER, PAS UN ENDROIT
 *
 * L'état naît `standard` à chaque montage. Rien n'est mémorisé d'un match à l'autre, ni dans
 * l'URL, ni dans le stockage du navigateur.
 *
 * # TOUTE BASCULE EST ANNONCÉE COMME UN REDIMENSIONNEMENT
 *
 * Le cadre change sans que la fenêtre change : rien ne prévient ceux qui se calent sur elle (la
 * hauteur offerte au terrain, la position du tiroir de réglages et des menus ancrés). Après
 * chaque changement de cadre, un événement `resize` est émis sur la fenêtre — c'en est un pour
 * eux. Le premier rendu n'annonce rien : la page n'a pas encore changé de cadre.
 *
 * # PENDANT UN EXPORT VIDÉO, AUCUNE BASCULE
 *
 * Le modèle ignore alors les deux bascules ; `disabled` le dit aux boutons. La sortie du Plein
 * cadre reste admise (cf. le modèle).
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import { useExportActive } from '../export/exportLayoutStore'
import {
  INITIAL_REPLAY_FRAME,
  reduceReplayFrame,
  type ReplayFrame,
  type ReplayFrameEvent,
  type ReplayFrameState,
} from '../model/replayFrame'

export interface ReplayFrameControl {
  /** Le cadre affiché. */
  frame: ReplayFrame
  /** `true` tant qu'un export vidéo tient la toile : les bascules sont sans effet. */
  disabled: boolean
  /** Touche T ou bouton Cinéma. */
  toggleCinema: () => void
  /** Touche F ou bouton Plein cadre. */
  toggleFullFrame: () => void
  /** Sortie du Plein cadre (Échap, « Quitter ») : rend le cadre d'où l'on vient ; sans effet ailleurs. */
  exit: () => void
  /** L'état À CET INSTANT, entre deux rendus : deux gestes rapprochés voient chacun le précédent. */
  peek: () => ReplayFrameState
}

export function useReplayFrame(): ReplayFrameControl {
  const disabled = useExportActive()
  const [state, setState] = useState<ReplayFrameState>(INITIAL_REPLAY_FRAME)
  // LA VÉRITÉ DU GESTE, lue hors rendu : deux frappes peuvent arriver entre deux rendus.
  const stateRef = useRef<ReplayFrameState>(INITIAL_REPLAY_FRAME)

  const send = useCallback(
    (event: ReplayFrameEvent) => {
      const next = reduceReplayFrame(stateRef.current, event, { exportActive: disabled })
      if (next === stateRef.current) return
      stateRef.current = next
      setState(next)
    },
    [disabled],
  )
  const toggleCinema = useCallback(() => send({ type: 'toggleCinema' }), [send])
  const toggleFullFrame = useCallback(() => send({ type: 'toggleFullFrame' }), [send])
  const exit = useCallback(() => send({ type: 'exit' }), [send])
  const peek = useCallback(() => stateRef.current, [])

  const announcedRef = useRef(state.frame)
  useEffect(() => {
    if (announcedRef.current === state.frame) return
    announcedRef.current = state.frame
    window.dispatchEvent(new Event('resize'))
  }, [state.frame])

  return { frame: state.frame, disabled, toggleCinema, toggleFullFrame, exit, peek }
}
