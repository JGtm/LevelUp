/**
 * replayTransportCommands — CE QUE LA TOILE REMET À SA BARRE DE LECTURE ET AUX SUPERPOSITIONS DE SON BLOC.
 *
 * La lecture, la frise, l'horloge, le son, la capture, le zoom et la vitesse naissent dans la toile
 * (`ReplayCanvas`) : aucun cadre ne les possède. La toile les regroupe donc en UN objet de commandes
 * (`ReplayTransportCommands`) et le remet à deux FONCTIONS DE RENDU que le cadre lui fournit :
 *  - `transport` : la barre de lecture, posée sous le bloc de la toile, dans la colonne que mesure
 *    `useReplayViewport` ;
 *  - `blockOverlays` : ce qui se pose DANS le bloc de la toile, à côté de la légende de la chaleur.
 *
 * Une autre barre ou une autre superposition pilote la même toile par ces commandes, sans que la toile la
 * connaisse. Une fonction de rendu est appelée pendant le rendu de la toile : elle est PURE, rend un
 * élément à partir des commandes, et n'appelle aucun hook (le hook serait celui de la toile).
 */
import { useMemo, type ReactNode, type RefObject } from 'react'

import type { ReplayCapture } from '../export/useReplayCapture'
import type { ReplayFullscreen } from '../hooks/useReplayFullscreen'
import type { ReplayPlayback } from '../hooks/useReplayPlayback'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplayZoom } from '../hooks/useReplayZoom'
import type { ReplayLocale } from '../i18n/i18n'
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
  /** Le zoom et le cadrage de la carte (paliers, déplacement). */
  zoom: ReplayZoom
}

/** Une fonction de rendu d'un emplacement de la toile : pure, sans hook, elle ne retient aucune commande. */
export type ReplayCommandsRender = (commands: ReplayTransportCommands) => ReactNode

/** Ce que la toile reçoit d'un cadre : sa barre de lecture, et les superpositions de son bloc. */
export interface ReplayCanvasSlots {
  transport: ReplayCommandsRender
  blockOverlays?: ReplayCommandsRender
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
  zoom: ReplayZoom
}

/**
 * useReplayTransportCommands regroupe les commandes de la toile en un objet dont l'identité ne change
 * qu'avec l'une d'elles. Les valeurs sont celles des hooks : rien n'est copié, retenu ni décidé ici.
 */
export function useReplayTransportCommands(s: ReplayTransportCommandSources): ReplayTransportCommands {
  const { playing, togglePlay, restart, seekBy } = s.playback
  const { autoPlay, toggleAutoPlay, speed, setSpeed } = s.settings
  const { clockRef, timeline, sound, capture, locale, fullscreen, zoom } = s
  return useMemo(
    () => ({
      playing, onTogglePlay: togglePlay, onRestart: restart, onSeekBy: seekBy, clockRef, timeline,
      autoPlay, onToggleAutoPlay: toggleAutoPlay, speed, onSetSpeed: setSpeed, sound, capture, locale, fullscreen, zoom,
    }),
    [
      playing, togglePlay, restart, seekBy, clockRef, timeline,
      autoPlay, toggleAutoPlay, speed, setSpeed, sound, capture, locale, fullscreen, zoom,
    ],
  )
}
