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
 * # UN CADRE SANS TOILE EST TOUJOURS LE STANDARD
 *
 * Les icônes de cadre vivent dans la barre de lecture, donc dans la toile : sans toile (rejeu en
 * chargement, en erreur ou absent) aucune n'est joignable, et la route garde le même composant d'un
 * match à l'autre. L'état est donc rangé avec sa PORTÉE (`scopeKey`) : la clé du match quand la page a un
 * rejeu à montrer, `null` sinon. Quand la portée change, l'état revient à `standard` AU RENDU, sans
 * effet (patron de `usePlayerPanel`) : aucune image ne montre le cadre d'un autre match ni un Cinéma
 * sans barre pour en sortir.
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
import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react'

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

/** L'état et la portée où il a été posé : hors de sa portée, il vaut l'état de départ. */
interface Scoped {
  scopeKey: string | null
  state: ReplayFrameState
}

/**
 * @param scopeKey la clé du match quand la page a un rejeu à montrer (toile montée), `null` sinon : tout
 *   changement de portée ramène le cadre au Standard.
 */
export function useReplayFrame(scopeKey: string | null): ReplayFrameControl {
  const disabled = useExportActive()
  const [stored, setStored] = useState<Scoped>({ scopeKey, state: INITIAL_REPLAY_FRAME })
  // LA VÉRITÉ DU GESTE, lue hors rendu : deux frappes peuvent arriver entre deux rendus.
  const truthRef = useRef<Scoped>(stored)
  // LE RETOUR AU STANDARD SE FAIT AU RENDU, pas dans un effet : aucune image ne montre l'état d'une autre portée.
  if (stored.scopeKey !== scopeKey) setStored({ scopeKey, state: INITIAL_REPLAY_FRAME })
  const state = stored.scopeKey === scopeKey ? stored.state : INITIAL_REPLAY_FRAME
  // LA VÉRITÉ DU GESTE SUIT L'ÉTAT VALIDÉ : à chaque validation elle reprend l'état rangé (donc le retour au
  // Standard d'un changement de portée), et les gestes qui arrivent ensuite l'écrivent avant le rendu suivant.
  useLayoutEffect(() => {
    truthRef.current = stored
  }, [stored])

  const send = useCallback(
    (event: ReplayFrameEvent) => {
      const current = truthRef.current.state
      const next = reduceReplayFrame(current, event, { exportActive: disabled })
      if (next === current) return
      truthRef.current = { scopeKey, state: next }
      setStored(truthRef.current)
    },
    [disabled, scopeKey],
  )
  const toggleCinema = useCallback(() => send({ type: 'toggleCinema' }), [send])
  const toggleFullFrame = useCallback(() => send({ type: 'toggleFullFrame' }), [send])
  const exit = useCallback(() => send({ type: 'exit' }), [send])
  const peek = useCallback(() => truthRef.current.state, [])

  const announcedRef = useRef(state.frame)
  useEffect(() => {
    if (announcedRef.current === state.frame) return
    announcedRef.current = state.frame
    window.dispatchEvent(new Event('resize'))
  }, [state.frame])

  return { frame: state.frame, disabled, toggleCinema, toggleFullFrame, exit, peek }
}
