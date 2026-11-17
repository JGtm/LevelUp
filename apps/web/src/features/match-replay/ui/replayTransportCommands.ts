/**
 * replayTransportCommands — CE QUE LA TOILE REMET À SA BARRE DE LECTURE ET AUX SUPERPOSITIONS DE SON BLOC.
 *
 * La lecture, la frise, l'horloge, le son, la capture, le zoom et la vitesse naissent dans la toile
 * (`ReplayCanvas`) : aucun cadre ne les possède. La toile les regroupe donc en UN objet de commandes
 * (`ReplayTransportCommands`) et le remet à deux FONCTIONS DE RENDU que le cadre lui fournit :
 *  - `transport` : la barre de lecture, posée sous le bloc de la toile, dans la colonne que mesure
 *    `useReplayViewport` ;
 *  - `blockOverlays` : ce qui se pose DANS le bloc de la toile, à côté de la légende de la chaleur (la
 *    commande de zoom du Standard, le fil des éliminations du Cinéma et du Plein cadre).
 *
 * Une autre barre ou une autre superposition pilote la même toile par ces commandes, sans que la toile la
 * connaisse. Les fonctions de rendu reçoivent en plus la CAMÉRA (`cameraFollow` : le joueur que la carte suit
 * et le geste « Suivre »), que seul le menu de la caméra de la barre compacte lit ; la barre complète
 * l'ignore. Une fonction de rendu est appelée pendant le rendu de la toile : elle est PURE, rend un
 * élément à partir des commandes, et n'appelle aucun hook (le hook serait celui de la toile).
 */
import { useMemo, type ReactNode, type RefObject } from 'react'

import type { ReplayCapture } from '../export/useReplayCapture'
import type { ReplayFrameControl } from '../hooks/useReplayFrame'
import type { ReplayFullscreen } from '../hooks/useReplayFullscreen'
import type { ReplayPlayback } from '../hooks/useReplayPlayback'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplayZoom } from '../hooks/useReplayZoom'
import type { ReplayLocale } from '../i18n/i18n'
import type { CameraFollow } from '../model/cameraMenu'
import type { ReplayOrientation } from '../settings/useReplayOrientation'
import type { ReplaySettings } from '../settings/useReplaySettings'
import type { ReplaySound } from '../sound/useReplaySound'

export interface ReplayTransportCommands {
  playing: boolean
  onTogglePlay: () => void
  onRestart: () => void
  /** Le saut ±10 s, en secondes signées (cf. useReplayPlayback.seekBy). */
  onSeekBy: (seconds: number) => void
  /** L'horloge est écrite par la boucle de dessin (textContent), pas par React. */
  clockRef: RefObject<HTMLSpanElement | null>
  /**
   * LA LECTURE AUTOMATIQUE : elle ne commande PAS le lecteur ouvert, elle décide de son état de départ,
   * lu une fois au montage (useReplayPlayback).
   */
  autoPlay: boolean
  onToggleAutoPlay: () => void
  /** La frise et ses pistes, en UN objet : les repères d'éliminations, de morts, de médias et la dominance. */
  timeline: ReplayTimeline
  speed: number
  onSetSpeed: (speed: number) => void
  sound: ReplaySound
  /** La capture d'image, l'enregistrement, l'export hors temps réel et son état. */
  capture: ReplayCapture
  locale: ReplayLocale
  /** Le mode plein écran de la page (useReplayFullscreen) : la commande, pas l'état de la page. */
  fullscreen: ReplayFullscreen
  /** Le cadre courant et les gestes qui y mènent (useReplayFrame) : les icônes de cadre de la barre les lisent. */
  frameControl: ReplayFrameControl
  /** Le zoom et le cadrage de la carte (paliers, déplacement). */
  zoom: ReplayZoom
  /**
   * L'Auto a tourné le plan : `{ onKeepUpright }` (retient 0 quart de tour pour la carte, le plan reste droit) ;
   * `null` quand le plan est droit ou que son orientation est un choix retenu. Seule la barre d'une ligne le lit.
   */
  autoRotation: { onKeepUpright: () => void } | null
}

/** Ce que la toile remet aux fonctions de rendu de ses emplacements : les commandes de la barre, et la caméra. */
export interface ReplaySlotCommands extends ReplayTransportCommands {
  cameraFollow: CameraFollow
  /** Le fil des éliminations est-il montré sur le plan ? Le cadre le pose ET l'option de la vue est allumée : la toile le décide. */
  planFeedShown: boolean
}

/** Une fonction de rendu d'un emplacement de la toile : pure, sans hook, elle ne retient aucune commande. */
export type ReplayCommandsRender = (commands: ReplaySlotCommands) => ReactNode

/** Ce que la toile reçoit d'un cadre : sa barre de lecture, et les superpositions de son bloc. */
export interface ReplayCanvasSlots {
  transport: ReplayCommandsRender
  blockOverlays?: ReplayCommandsRender
  /**
   * Le cadre pose le fil des éliminations sur le plan, dans `blockOverlays` : la toile propose alors l'option
   * « Fil des éliminations » de la vue et libère le coin bas gauche en posant la légende de la chaleur en haut.
   */
  planFeed?: boolean
}

/** Ce que la toile assemble en commandes : les sorties de ses hooks, telles quelles. */
export interface ReplayTransportCommandSources {
  playback: Pick<ReplayPlayback, 'playing' | 'togglePlay' | 'restart' | 'seekBy'>
  settings: Pick<ReplaySettings, 'autoPlay' | 'toggleAutoPlay' | 'speed' | 'setSpeed'>
  clockRef: RefObject<HTMLSpanElement | null>
  timeline: ReplayTimeline
  sound: ReplaySound
  capture: ReplayCapture
  locale: ReplayLocale
  fullscreen: ReplayFullscreen
  frameControl: ReplayFrameControl
  zoom: ReplayZoom
  cameraFollow: CameraFollow
  planFeedShown: boolean
  /** Vrai quand l'Auto a tourné le plan (`useReplayView`) ; `orientation` retient le retour au plan droit. */
  autoTurned: boolean
  orientation: Pick<ReplayOrientation, 'setTurns'>
}

/**
 * useReplayTransportCommands regroupe les commandes de la toile en un objet dont l'identité ne change
 * qu'avec l'une d'elles. Les valeurs sont celles des hooks : rien n'est copié, retenu ni décidé ici.
 */
export function useReplayTransportCommands(s: ReplayTransportCommandSources): ReplaySlotCommands {
  const { playing, togglePlay, restart, seekBy } = s.playback
  const { autoPlay, toggleAutoPlay, speed, setSpeed } = s.settings
  const { clockRef, timeline, sound, capture, locale, fullscreen, frameControl, zoom, cameraFollow, planFeedShown, autoTurned } = s
  const { setTurns } = s.orientation
  return useMemo(
    () => ({
      playing, onTogglePlay: togglePlay, onRestart: restart, onSeekBy: seekBy, clockRef, timeline,
      autoPlay, onToggleAutoPlay: toggleAutoPlay, speed, onSetSpeed: setSpeed, sound, capture, locale, fullscreen, frameControl, zoom,
      cameraFollow, planFeedShown, autoRotation: autoTurned ? { onKeepUpright: () => setTurns(0) } : null,
    }),
    [
      playing, togglePlay, restart, seekBy, clockRef, timeline,
      autoPlay, toggleAutoPlay, speed, setSpeed, sound, capture, locale, fullscreen, frameControl, zoom, cameraFollow, planFeedShown, autoTurned, setTurns,
    ],
  )
}
