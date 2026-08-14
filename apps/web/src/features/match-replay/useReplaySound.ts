/**
 * useReplaySound — LE CÂBLAGE du son du rejeu : préférence de l'utilisateur, cycle de vie
 * du lecteur Web Audio, et le battement qui fait sonner ce que le curseur vient de passer.
 *
 * Il n'y a ici NI règle sonore (replaySound.ts : quoi sonne, quand, et le curseur qui ne
 * rejoue rien deux fois) NI lecture (replayAudio.ts : contexte, enveloppe, voix) — ce
 * fichier n'est que la couture React entre les deux, pour que le composant canvas n'ait
 * pas à connaître Web Audio (anti-pattern « logique dans le composant »).
 *
 * TROIS SILENCES SONT VOULUS, et aucun n'est un bug :
 *  - COUPÉ PAR DÉFAUT : rien ne sonne, rien ne se télécharge même, tant que l'utilisateur
 *    n'a pas cliqué. L'AudioContext naît DANS ce clic (politique d'autoplay : un contexte
 *    créé hors geste démarre suspendu) ;
 *  - LECTEUR EN PAUSE OU ONGLET EN ARRIÈRE-PLAN : la boucle d'animation ne bat plus, donc
 *    `tick` n'est plus appelé. Au retour, le saut de temps recale le curseur en silence
 *    (SOUND_RESYNC_JUMP_MS) : ce qui a été enjambé ne se rejoue pas ;
 *  - AVANCE RAPIDE : au-delà de SOUND_MAX_SPEED, le curseur suit sans jouer.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { staticAssetURL } from '@/lib/staticAssets'

import { ReplayAudioPlayer } from './replayAudio'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  advanceSoundCursor,
  buildSoundTimeline,
  resyncSoundCursor,
  soundPlaysAtSpeed,
  type SoundCursor,
} from './replaySound'

/** Préférences persistées (patron localStorage du repo : clé simple, lecture sous try). */
const SOUND_ON_KEY = 'replay-sound-on'
const SOUND_VOLUME_KEY = 'replay-sound-volume'

/** Volume d'ouverture : assez présent pour s'entendre, assez bas pour ne pas faire sursauter. */
export const SOUND_VOLUME_DEFAULT = 0.7

/** Ce que le composant reçoit : un état à afficher, deux commandes, un battement. */
export interface ReplaySound {
  /** La piste porte au moins un son : sinon, pas de commande à offrir. */
  available: boolean
  on: boolean
  toggle: () => void
  volume: number
  setVolume: (v: number) => void
  /** Le son est activé mais tu par la vitesse de lecture (à dire, pas à cacher). */
  mutedBySpeed: boolean
  /** À appeler à chaque pas d'animation avec l'instant courant du rejeu, en ms. */
  tick: (ms: number) => void
}

function readStoredOn(): boolean {
  try {
    return localStorage.getItem(SOUND_ON_KEY) === 'true'
  } catch {
    return false
  }
}

function readStoredVolume(): number {
  try {
    const v = Number(localStorage.getItem(SOUND_VOLUME_KEY))
    return Number.isFinite(v) && v > 0 && v <= 1 ? v : SOUND_VOLUME_DEFAULT
  } catch {
    return SOUND_VOLUME_DEFAULT
  }
}

function persist(key: string, value: string): void {
  try {
    localStorage.setItem(key, value)
  } catch {
    /* stockage refusé (navigation privée) : la préférence ne survit pas, le son marche. */
  }
}

export function useReplaySound(
  doc: ReplayDocumentReady,
  kills: KillEvent[] | undefined,
  t0Ms: number | undefined,
  speed: number,
): ReplaySound {
  const [on, setOn] = useState(readStoredOn)
  const [volume, setVolumeState] = useState(readStoredVolume)

  const timeline = useMemo(
    () => buildSoundTimeline(doc, kills ?? [], t0Ms ?? 0),
    [doc, kills, t0Ms],
  )
  // Les URL des sons EFFECTIVEMENT présents dans ce match : on ne précharge jamais le pack
  // entier (27 fichiers) pour un match qui n'en joue que cinq.
  const urls = useMemo(() => {
    const m = new Map<string, string>()
    for (const e of timeline) {
      if (!m.has(e.stem)) m.set(e.stem, staticAssetURL('sound', e.stem, '.wav'))
    }
    return m
  }, [timeline])

  const playerRef = useRef<ReplayAudioPlayer | null>(null)
  const cursorRef = useRef<SoundCursor>({ ms: 0, idx: 0 })
  // Le prochain battement POSE le curseur sans rien jouer. Vrai à l'activation et à tout
  // changement de piste : sans cela, activer le son à 500 ms ferait partir d'un coup tout
  // ce qui précède (l'écart au curseur, resté à 0, passerait pour une lecture continue).
  const resyncRef = useRef(true)
  // Lus par le battement sans en être des dépendances : couper le son ou changer de vitesse
  // ne doit pas recréer la boucle d'animation du canvas.
  const onRef = useRef(on)
  const speedRef = useRef(speed)
  const timelineRef = useRef(timeline)
  const urlsRef = useRef(urls)

  useEffect(() => { onRef.current = on }, [on])
  useEffect(() => { speedRef.current = speed }, [speed])
  useEffect(() => {
    timelineRef.current = timeline
    urlsRef.current = urls
    resyncRef.current = true
    // Un match qui change alors que le son tourne : les nouveaux sons se chargent, les
    // anciens restent en cache (même lecteur, mêmes URL le cas échéant).
    if (onRef.current) playerRef.current?.preload(urls.values())
  }, [timeline, urls])

  // Le lecteur meurt AVEC le composant : un AudioContext laissé ouvert survit à la page.
  useEffect(() => () => {
    playerRef.current?.dispose()
    playerRef.current = null
  }, [])

  const toggle = useCallback(() => {
    setOn((prev) => {
      const next = !prev
      persist(SOUND_ON_KEY, String(next))
      if (next) {
        // DANS LE GESTE : c'est la seule fenêtre où un AudioContext démarre en marche.
        if (!playerRef.current) playerRef.current = new ReplayAudioPlayer(volume)
        playerRef.current.resume()
        playerRef.current.setVolume(volume)
        playerRef.current.preload(urlsRef.current.values())
        resyncRef.current = true
      } else {
        // Coupure IMMÉDIATE (rampe du maître) : ce qui est en vol s'éteint avec, plutôt que
        // de traîner une seconde après le clic.
        playerRef.current?.setVolume(0)
      }
      onRef.current = next
      return next
    })
  }, [volume])

  const setVolume = useCallback((v: number) => {
    const clamped = Math.min(Math.max(v, 0), 1)
    setVolumeState(clamped)
    persist(SOUND_VOLUME_KEY, String(clamped))
    if (onRef.current) playerRef.current?.setVolume(clamped)
  }, [])

  const tick = useCallback((ms: number) => {
    const tl = timelineRef.current
    const player = playerRef.current
    // Le curseur SUIT toujours la lecture, même muet : en revenant à 1×, on repart de
    // l'instant courant et non d'un passé enjambé.
    if (!onRef.current || !player || !soundPlaysAtSpeed(speedRef.current)) {
      cursorRef.current = resyncSoundCursor(tl, ms)
      return
    }
    if (resyncRef.current) {
      resyncRef.current = false
      cursorRef.current = resyncSoundCursor(tl, ms)
      return
    }
    const { cursor, fire } = advanceSoundCursor(tl, cursorRef.current, ms)
    cursorRef.current = cursor
    for (const e of fire) {
      const url = urlsRef.current.get(e.stem)
      if (url) player.play(url)
    }
  }, [])

  return {
    available: timeline.length > 0,
    on,
    toggle,
    volume,
    setVolume,
    mutedBySpeed: on && !soundPlaysAtSpeed(speed),
    tick,
  }
}
