/**
 * useReplayPlaybackCommands — LES COMMANDES DE DÉPLACEMENT ET DE TRANSPORT de la lecture
 * (`useReplayPlayback`) : sauts, pas d'image, « Recommencer », « Lecture » / « Pause ».
 *
 * Extraites de `useReplayPlayback` pour tenir la règle des 80 lignes par fonction. Chacune passe par le
 * `seekTo` borné du hook (une seule définition de « où le curseur a le droit d'aller ») et par ses
 * rappels ; le temps de la lecture et le contrat de chaque rappel restent décrits dans
 * `useReplayPlayback.ts`.
 */
import { type Dispatch, type RefObject, type SetStateAction } from 'react'

interface PlaybackCommandsInput {
  playing: boolean
  setPlaying: Dispatch<SetStateAction<boolean>>
  frameRef: RefObject<number>
  baseFps: number
  /** La borne haute de la lecture et le préambule où elle se pose. */
  endFrame: number
  leadInFrame: number
  seekTo: (frame: number) => void
  /** Ramène le curseur au préambule, sans rien lancer. */
  rewind: () => void
  onUserMove?: () => void
  onTransportGesture: () => void
  onStarted?: () => void
}

export interface PlaybackCommands {
  togglePlay: () => void
  restart: () => void
  /** Saut de ±N SECONDES sur l'axe réel (converti en images par `baseFps`). */
  seekBy: (seconds: number) => void
  stepFrames: (frames: number) => void
  seekToFrame: (frame: number) => void
}

export function usePlaybackCommands(i: PlaybackCommandsInput): PlaybackCommands {
  const { playing, setPlaying, frameRef, baseFps, endFrame, leadInFrame, seekTo, rewind } = i
  const { onUserMove, onTransportGesture, onStarted } = i

  const seekBy = (seconds: number) => {
    onUserMove?.()
    seekTo(frameRef.current + seconds * baseFps)
  }

  const seekToFrame = (frame: number) => {
    onUserMove?.()
    seekTo(frame)
  }

  const stepFrames = (frames: number) => {
    onUserMove?.()
    // EN PAUSE D'ABORD (cf. l'en-tête de `useReplayPlayback`) : sous la boucle, le pas d'image serait
    // écrasé à la frame suivante et le geste n'aurait aucun effet visible.
    setPlaying(false)
    seekTo(Math.round(frameRef.current) + frames)
  }

  // LES DEUX COMMANDES DE TRANSPORT PRÉVIENNENT LE SON, en PREMIER : ce sont des gestes
  // utilisateur, la seule fenêtre où un AudioContext démarre en marche. C'est ce qui rend son
  // son à un rejeu rechargé dont la préférence était restée à « activé » (cf. useReplaySound).
  const restart = () => {
    onUserMove?.()
    onTransportGesture()
    rewind()
    onStarted?.() // toujours au préambule après le rembobinage
    setPlaying(true)
  }

  const togglePlay = () => {
    onTransportGesture()
    // LA REPRISE EFFACE LA MISE EN VALEUR, PAS LA PAUSE (cf. `onUserMove`).
    if (!playing) onUserMove?.()
    // REPARTIR DU DÉBUT SUR UN REJEU TERMINÉ (cf. l'en-tête de `useReplayPlayback`) : sans ce
    // rembobinage, la boucle relancée à `endFrame` conclurait « fin » à son premier pas et se
    // rendormirait — le bouton « Lecture » n'aurait aucun effet visible.
    if (!playing && frameRef.current >= endFrame) rewind()
    // UN DÉPART, PAS UNE PAUSE, ET DEPUIS LE PRÉAMBULE (cf. `onStarted`).
    if (!playing && frameRef.current <= leadInFrame) onStarted?.()
    setPlaying((p) => !p)
  }

  return { togglePlay, restart, seekBy, stepFrames, seekToFrame }
}
