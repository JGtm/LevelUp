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

import { persistPreference, readStoredFlag, readStoredNumber } from './replayPreferences'
import { ReplayAudioPlayer } from './replayAudio'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  advanceSoundCursor,
  buildSoundTimeline,
  resyncSoundCursor,
  soundPlaysAtSpeed,
  SOUND_CATEGORIES,
  SOUND_CATEGORIES_DEFAULT,
  type SoundCategory,
  type SoundCategoryFilter,
  type SoundCursor,
} from './replaySound'

/** Préférences persistées — patron partagé (replayPreferences.ts), né ici. */
const SOUND_ON_KEY = 'replay-sound-on'
const SOUND_VOLUME_KEY = 'replay-sound-volume'
/** Catégories COUPÉES, pas les catégories actives : un futur 5e stem reste ON par défaut
 *  pour qui a déjà une préférence stockée — jamais besoin de migrer ce JSON. */
const SOUND_CATEGORIES_OFF_KEY = 'replay-sound-categories-off'

/** Volume d'ouverture : assez présent pour s'entendre, assez bas pour ne pas faire sursauter. */
export const SOUND_VOLUME_DEFAULT = 0.7

/** Ce que le composant reçoit : un état à afficher, des commandes, un battement. */
export interface ReplaySound {
  /** La piste porte au moins un son : sinon, pas de commande à offrir. INDÉPENDANT du
   *  filtre par catégorie — tout décocher ne doit pas faire disparaître le panneau. */
  available: boolean
  on: boolean
  toggle: () => void
  volume: number
  setVolume: (v: number) => void
  /** Le son est activé mais tu par la vitesse de lecture (à dire, pas à cacher). */
  mutedBySpeed: boolean
  /** Filtre par catégorie (tiroir de réglages, phase 2) : `true` = catégorie audible. */
  categories: SoundCategoryFilter
  toggleCategory: (category: SoundCategory) => void
  /** À appeler à chaque pas d'animation avec l'instant courant du rejeu, en ms. */
  tick: (ms: number) => void
}

/** Lit les catégories COUPÉES ; une valeur inconnue (ancienne clé, JSON corrompu) est
 *  ignorée plutôt que de faire échouer toute la lecture — silence propre côté préférence. */
function readStoredCategoriesOff(): ReadonlySet<SoundCategory> {
  try {
    const raw = localStorage.getItem(SOUND_CATEGORIES_OFF_KEY)
    if (!raw) return new Set()
    const parsed: unknown = JSON.parse(raw)
    if (!Array.isArray(parsed)) return new Set()
    const known: readonly string[] = SOUND_CATEGORIES
    return new Set(parsed.filter((v): v is SoundCategory => known.includes(v)))
  } catch {
    return new Set()
  }
}

/**
 * Sous-hook : le filtre par catégorie, sa persistance, sa bascule — extrait pour garder
 * useReplaySound sous le seuil de lisibilité (CLAUDE.md n°5, fonction ≤ 80 lignes) plutôt
 * que de laisser grossir une fonction déjà dense. Même responsabilité que le reste du
 * fichier (préférence utilisateur + câblage React), un cran plus petit.
 */
function useSoundCategoryFilter(): {
  categories: SoundCategoryFilter
  toggleCategory: (category: SoundCategory) => void
} {
  const [categoriesOff, setCategoriesOff] = useState<ReadonlySet<SoundCategory>>(readStoredCategoriesOff)

  const categories = useMemo<SoundCategoryFilter>(() => ({
    weapon: !categoriesOff.has('weapon'),
    grenade: !categoriesOff.has('grenade'),
    melee: !categoriesOff.has('melee'),
    equipment: !categoriesOff.has('equipment'),
  }), [categoriesOff])

  const toggleCategory = useCallback((category: SoundCategory) => {
    setCategoriesOff((prev) => {
      const next = new Set(prev)
      if (next.has(category)) next.delete(category)
      else next.add(category)
      persistPreference(SOUND_CATEGORIES_OFF_KEY, JSON.stringify(Array.from(next)))
      return next
    })
  }, [])

  return { categories, toggleCategory }
}

/** Le match porte-t-il un son du tout, INDÉPENDAMMENT du filtre choisi ? Sert `available` :
 *  si elle suivait la piste filtrée, tout décocher ferait disparaître le seul bouton qui
 *  permet de tout rallumer. */
function hasSoundEvents(doc: ReplayDocumentReady, kills: KillEvent[], t0Ms: number): boolean {
  return buildSoundTimeline(doc, kills, t0Ms, SOUND_CATEGORIES_DEFAULT).length > 0
}

/** Les URL des sons EFFECTIVEMENT présents dans une piste : on ne précharge jamais le pack
 *  entier (27 fichiers) pour un match qui n'en joue que cinq. */
function soundURLsFor(timeline: readonly { stem: string }[]): Map<string, string> {
  const urls = new Map<string, string>()
  for (const e of timeline) {
    if (!urls.has(e.stem)) urls.set(e.stem, staticAssetURL('sound', e.stem, '.wav'))
  }
  return urls
}

export function useReplaySound(
  doc: ReplayDocumentReady,
  kills: KillEvent[] | undefined,
  t0Ms: number | undefined,
  speed: number,
): ReplaySound {
  const [on, setOn] = useState(() => readStoredFlag(SOUND_ON_KEY, false))
  const [volume, setVolumeState] = useState(() =>
    readStoredNumber(SOUND_VOLUME_KEY, SOUND_VOLUME_DEFAULT, (v) => v > 0 && v <= 1),
  )
  const { categories, toggleCategory } = useSoundCategoryFilter()

  // Piste JOUÉE, catégories coupées retirées À LA CONSTRUCTION (jamais en aval, dans le
  // lecteur) ; DISPONIBILITÉ DU PANNEAU indépendante de ce filtre (hasSoundEvents ci-dessus).
  const timeline = useMemo(
    () => buildSoundTimeline(doc, kills ?? [], t0Ms ?? 0, categories),
    [doc, kills, t0Ms, categories],
  )
  const hasAnySound = useMemo(
    () => hasSoundEvents(doc, kills ?? [], t0Ms ?? 0),
    [doc, kills, t0Ms],
  )
  const urls = useMemo(() => soundURLsFor(timeline), [timeline])

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
      persistPreference(SOUND_ON_KEY, String(next))
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
    persistPreference(SOUND_VOLUME_KEY, String(clamped))
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
    available: hasAnySound,
    on,
    toggle,
    volume,
    setVolume,
    mutedBySpeed: on && !soundPlaysAtSpeed(speed),
    categories,
    toggleCategory,
    tick,
  }
}
