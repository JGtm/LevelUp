/**
 * useReplaySound — LE CÂBLAGE du son du rejeu : préférence de l'utilisateur, cycle de vie
 * du lecteur Web Audio, et le battement qui fait sonner ce que le curseur vient de passer.
 *
 * Il n'y a ici NI règle sonore (replaySound.ts : quoi sonne, quand, et le curseur qui ne
 * rejoue rien deux fois) NI lecture (replayAudio.ts : contexte, enveloppe, voix) — ce
 * fichier n'est que la couture React entre les deux, pour que le composant canvas n'ait
 * pas à connaître Web Audio (anti-pattern « logique dans le composant »).
 *
 * DEUX ÉTATS, PAS UN — ET C'EST LA LEÇON DU CORRECTIF DU 2026-08-27. La PRÉFÉRENCE (« je veux
 * du son », persistée) et le LECTEUR (l'AudioContext, qui ne peut naître que dans un geste)
 * sont deux choses distinctes, et un rechargement de page les désaccorde forcément : la
 * préférence revient du stockage local, le lecteur non. Pendant cette fenêtre le panneau
 * affichait « son activé » alors que rien ne pouvait sonner, et le clic suivant — le geste qui
 * aurait pu tout réparer — basculait la préférence à « coupé ». Il fallait donc DEUX clics pour
 * entendre quoi que ce soit, ce qui ressemblait à une panne.
 *
 * LE DÉSACCORD SE RÉSOUT DANS LE PREMIER GESTE, quel qu'il soit, et jamais avant :
 *  - le bouton du son ACTIVE quand rien ne joue, au lieu de couper une préférence déjà à
 *    « activé » (`toggle`) ;
 *  - un geste de transport — « Lecture », « Recommencer » — éveille le lecteur si la
 *    préférence le demande (`wake`, appelé par `useReplayPlayback`), pour que le son revienne
 *    sans même passer par le bouton.
 * Dans les deux cas l'AudioContext naît DANS un geste utilisateur : la doctrine ne bouge pas,
 * c'est la liste des gestes qui comptent qui s'allonge.
 *
 * TROIS SILENCES SONT VOULUS, et aucun n'est un bug :
 *  - COUPÉ PAR DÉFAUT : rien ne sonne, rien ne se télécharge même, tant que l'utilisateur
 *    n'a pas cliqué. L'AudioContext naît DANS ce clic (politique d'autoplay : un contexte
 *    créé hors geste démarre suspendu) ;
 *  - LECTEUR EN PAUSE OU ONGLET EN ARRIÈRE-PLAN : la boucle d'animation ne bat plus, donc
 *    `tick` n'est plus appelé. Au retour, le saut de temps recale le curseur en silence
 *    (SOUND_RESYNC_JUMP_MS) : ce qui a été enjambé ne se rejoue pas ;
 *  - AVANCE RAPIDE : au-delà de SOUND_MAX_SPEED, le curseur suit sans jouer.
 *
 * LE SON DE FIN DE PARTIE (lot C, 2026-08-27) EST LE SEUL QUI NE VIENNE PAS DE LA PISTE. Il
 * n'a pas d'instant sur l'horloge du film : c'est la LECTURE qui l'appelle en arrivant sur la
 * borne de fin (`useReplayPlayback.onEnded`), une fois par arrivée. Il passe par le même
 * lecteur, donc par la même préférence et le même volume — son coupé, rien ; et il obéit aussi
 * au silence d'avance rapide, sans quoi le panneau annoncerait « son coupé par la vitesse »
 * pendant qu'une fanfare joue. Ses prises sont PRÉCHARGÉES avec la piste : le tirage a lieu à
 * l'arrivée en fin, et un fichier demandé à cet instant arriverait trop tard.
 */
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { useSettings } from '@/features/settings/queries'
import { staticAssetURL } from '@/lib/staticAssets'

import { persistPreference, readStoredFlag, readStoredNumber } from './replayPreferences'
import { endMatchSounds, endMatchSoundStems, type EndMatchSoundSpec } from './endMatchSound'
import type { ReplayLocale } from './i18n'
import { ReplayAudioPlayer } from './replayAudio'
import type { ReplayDocumentReady } from './replayNormalize'
import { distanceChain, drawVariation } from './weaponSoundLogic'
import { WEAPON_SOUND_VARIATIONS } from './weaponSoundVariations'
import {
  buildSoundTimeline,
  SOUND_CATEGORIES,
  SOUND_CATEGORIES_DEFAULT,
  type SoundCategory,
  type SoundCategoryFilter,
} from './replaySound'
import {
  allyTeamFromScoreboard,
  sideResolverFromScoreboard,
  type ScoreboardSide,
} from './objectiveSound'
import { pickVariantStem, stemsOf } from './replaySoundVariants'
import {
  advanceSoundCursor,
  resyncSoundCursor,
  soundPlaysAtSpeed,
  type SoundCursor,
} from './replaySoundCursor'

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
  /**
   * À appeler dans un GESTE DE TRANSPORT (lecture, recommencer) : rend son lecteur à un rejeu
   * dont la préférence est restée à « activé » d'une session à l'autre. Sans effet si le son
   * est coupé ou si le lecteur vit déjà.
   */
  wake: () => void
  volume: number
  setVolume: (v: number) => void
  /** Le son est activé mais tu par la vitesse de lecture (à dire, pas à cacher). */
  mutedBySpeed: boolean
  /** Filtre par catégorie (tiroir de réglages, phase 2) : `true` = catégorie audible. */
  categories: SoundCategoryFilter
  toggleCategory: (category: SoundCategory) => void
  /** À appeler à chaque pas d'animation avec l'instant courant du rejeu, en ms. */
  tick: (ms: number) => void
  /**
   * LA CONCLUSION : voix de l'annonceur + fanfare, à appeler UNE fois quand la lecture atteint
   * la borne de fin du match. Rien ne sonne si le son est coupé, si la fin n'est pas lisible
   * (cf. `endMatchSoundSpec`) ou si la vitesse dépasse SOUND_MAX_SPEED.
   */
  endMatch: () => void
  /**
   * LA PISTE AUDIO À JOINDRE À UNE VIDÉO enregistrée, ou `null`.
   *
   * `null` dans deux cas qui n'en font qu'un du point de vue de l'utilisateur : le son est
   * coupé, ou le lecteur n'est pas né (il ne naît que dans le geste qui active le son). Le
   * clip sort alors muet, et c'est la décision 6 du plan — la piste est câblée AU DÉMARRAGE
   * de l'enregistrement, donc activer le son ensuite ne l'ajoute pas au clip en cours. La
   * raison n'est pas technique mais de lisibilité : un fichier dont le son démarrerait en
   * cours de route passerait pour un fichier abîmé.
   */
  recordingTrack: () => MediaStreamTrack | null
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
    objective: !categoriesOff.has('objective'),
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
 *  entier (58 fichiers) pour un match qui n'en joue que cinq. `extra` porte les stems qui
 *  n'ont pas d'instant sur l'horloge — les prises de la fin de partie, dont le tirage n'aura
 *  lieu qu'à l'arrivée en fin. */
function soundURLsFor(
  timeline: readonly { stem: string; variants?: readonly string[] }[],
  extra: readonly string[],
): Map<string, string> {
  const urls = new Map<string, string>()
  for (const e of timeline) {
    // TOUTES les variantes sont préchargées, pas seulement celle qui sortira du tirage : le
    // tirage a lieu à la lecture, et un fichier pas encore décodé y serait un silence.
    for (const stem of stemsOf(e)) {
      if (!urls.has(stem)) urls.set(stem, staticAssetURL('sound', stem, '.wav'))
    }
  }
  for (const stem of extra) {
    if (!urls.has(stem)) urls.set(stem, staticAssetURL('sound', stem, '.wav'))
  }
  return urls
}

/**
 * Sous-hook : les REGLAGES D'INSTANCE des sons d'armes (page admin — variation RANGED et
 * distance), extrait pour la meme raison que le filtre par categorie : garder le hook
 * principal sous le seuil de lisibilite. La variation se lit au TIRAGE (ref, pas une
 * dependance du battement) ; la distance se POSE sur le lecteur des qu'elle change — et
 * `apply` la pose aussi a la creation du lecteur, qui nait apres le premier reglage.
 */
function useInstanceSoundTuning(playerRef: { current: ReplayAudioPlayer | null }): {
  variationPercentRef: { current: number }
  apply: (player: ReplayAudioPlayer) => void
} {
  const { data: settings } = useSettings()
  const variationPercent = settings?.replay_sound_variation_percent ?? 100
  const distancePercent = settings?.replay_sound_distance_percent ?? 0
  const variationPercentRef = useRef(variationPercent)
  const apply = useCallback(
    (player: ReplayAudioPlayer) => player.setDistance(distanceChain(distancePercent)),
    [distancePercent],
  )
  useEffect(() => {
    variationPercentRef.current = variationPercent
    if (playerRef.current) apply(playerRef.current)
  }, [variationPercent, apply, playerRef])
  return { variationPercentRef, apply }
}

export function useReplaySound(
  doc: ReplayDocumentReady,
  kills: KillEvent[] | undefined,
  t0Ms: number | undefined,
  speed: number,
  // Le tableau de score, d'où se DÉDUIT le camp de l'auteur d'une action d'objectif (résolveur
  // pur : `sideResolverFromScoreboard`). Absent, ou sans ligne « moi » : les actions qui ont
  // deux variantes d'équipe restent MUETTES — le rejeu ne devine jamais un camp, même règle
  // que l'encre des calques.
  scoreboard?: readonly ScoreboardSide[],
  endMatch: EndMatchSoundSpec | null = null,
  // La LANGUE de l'interface : elle ne sert QU'au son « manche terminée » (voix d'annonceur,
  // `roundOverSound.ts`), la seule entrée locale-aware de la piste. Absente, ce son se tait.
  locale?: ReplayLocale,
): ReplaySound {
  const sideOfXuid = useMemo(() => sideResolverFromScoreboard(scoreboard), [scoreboard])
  // Le camp allié EN NUMÉRO : les sons d'état de zone joignent sur le propriétaire d'une zone,
  // pas sur le xuid d'un joueur. Même lecture du tableau de score que le résolveur ci-dessus.
  const allyTeam = useMemo(() => allyTeamFromScoreboard(scoreboard), [scoreboard])
  const [on, setOn] = useState(() => readStoredFlag(SOUND_ON_KEY, false))
  const [volume, setVolumeState] = useState(() =>
    readStoredNumber(SOUND_VOLUME_KEY, SOUND_VOLUME_DEFAULT, (v) => v > 0 && v <= 1),
  )
  const { categories, toggleCategory } = useSoundCategoryFilter()
  const playerRef = useRef<ReplayAudioPlayer | null>(null)
  const tuning = useInstanceSoundTuning(playerRef)

  // Piste JOUÉE, catégories coupées retirées À LA CONSTRUCTION (jamais en aval, dans le
  // lecteur) ; DISPONIBILITÉ DU PANNEAU indépendante de ce filtre (hasSoundEvents ci-dessus).
  const timeline = useMemo(
    () => buildSoundTimeline(doc, kills ?? [], t0Ms ?? 0, categories, sideOfXuid, allyTeam, locale),
    [doc, kills, t0Ms, categories, sideOfXuid, allyTeam, locale],
  )
  const hasAnySound = useMemo(
    () => hasSoundEvents(doc, kills ?? [], t0Ms ?? 0),
    [doc, kills, t0Ms],
  )
  // Les prises de la FIN entrent dans le préchargement avec la piste : le tirage n'a lieu qu'à
  // l'arrivée en fin, et un fichier demandé à cet instant sonnerait après le silence.
  const urls = useMemo(
    () => soundURLsFor(timeline, endMatchSoundStems(endMatch)),
    [timeline, endMatch],
  )

  const cursorRef = useRef<SoundCursor>({ ms: 0, idx: 0 })
  // Le prochain battement POSE le curseur sans rien jouer. Vrai à l'activation et à tout
  // changement de piste : sans cela, activer le son à 500 ms ferait partir d'un coup tout
  // ce qui précède (l'écart au curseur, resté à 0, passerait pour une lecture continue).
  const resyncRef = useRef(true)
  // Lus par le battement sans en être des dépendances : couper le son ou changer de vitesse
  // ne doit pas recréer la boucle d'animation du canvas.
  const onRef = useRef(on)
  // Le NIVEAU réglé, lu par les gestes qui ouvrent le lecteur : le mettre en dépendance de
  // leurs rappels les recréerait à chaque pas du curseur de volume. Il s'écrit dans
  // `setVolume`, son seul point d'entrée.
  const volumeRef = useRef(volume)
  const speedRef = useRef(speed)
  const timelineRef = useRef(timeline)
  const urlsRef = useRef(urls)
  // La lecture de fin suit la même règle : `onEnded` doit rester STABLE, sans quoi la boucle
  // d'animation se recréerait le jour où l'en-tête du match arrive.
  const endMatchRef = useRef(endMatch)

  useEffect(() => { onRef.current = on }, [on])
  useEffect(() => { speedRef.current = speed }, [speed])
  useEffect(() => { endMatchRef.current = endMatch }, [endMatch])
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

  /**
   * openPlayer — LE LECTEUR PREND VIE. À N'APPELER QUE DANS UN GESTE UTILISATEUR : c'est la
   * seule fenêtre où un AudioContext démarre en marche (politique d'autoplay).
   *
   * Il lit le volume dans une REF et non dans l'état : cette fonction est appelée par des
   * rappels stables (bascule du bouton, gestes de transport), et en faire une dépendance du
   * volume les recréerait à chaque mouvement du curseur.
   */
  const openPlayer = useCallback(() => {
    const level = volumeRef.current
    let player = playerRef.current
    if (!player) {
      player = new ReplayAudioPlayer(level)
      playerRef.current = player
      tuning.apply(player)
    }
    player.resume()
    player.setVolume(level)
    player.preload(urlsRef.current.values())
    resyncRef.current = true
  }, [tuning])

  /**
   * wake — CE QUE FAIT UN GESTE DE TRANSPORT QUAND LA PRÉFÉRENCE EST DÉJÀ À « ACTIVÉ ».
   *
   * Rien du tout dans le cas nominal : le lecteur existe déjà, ou le son est coupé. Il ne sert
   * qu'à la situation décrite plus haut — préférence restaurée à `true` par le stockage local,
   * aucun lecteur, donc aucun son. Le premier « Lecture » ou « Recommencer » est un geste
   * utilisateur ; il vaut donc pour la politique d'autoplay, et il rend au rejeu le son que
   * l'utilisateur n'a jamais coupé.
   */
  const wake = useCallback(() => {
    if (!onRef.current || playerRef.current) return
    openPlayer()
  }, [openPlayer])

  const toggle = useCallback(() => {
    // RIEN À COMMANDER, RIEN NE BOUGE (correctif du 2026-08-28, revue R1). Le bouton du son ne
    // se rend pas quand le match n'a aucun son — mais le RACCOURCI CLAVIER, lui, n'a pas de
    // rendu à respecter : « M » sur un tel match basculait la préférence, la PERSISTAIT
    // (`SOUND_ON_KEY`) et ouvrait un lecteur, sans qu'un seul pixel ne change à l'écran. Le
    // rejeu suivant démarrait alors dans l'état inverse de celui qu'on croyait avoir laissé.
    // La garde vit ICI, chez le propriétaire de l'état, et pas dans le clavier : c'est le seul
    // endroit qui la rend vraie pour TOUS les appelants, présents et à venir.
    if (!hasAnySound) return
    // LE PREMIER CLIC APRÈS UN RECHARGEMENT ACTIVE, IL NE COUPE PAS (correctif du 2026-08-27).
    // La préférence est à « activé » mais aucun lecteur n'existe : l'état affiché et l'état
    // réel se contredisent, et basculer la préférence à « coupé » ferait payer à l'utilisateur
    // une incohérence dont il n'est pas l'auteur — il faudrait DEUX clics pour entendre quelque
    // chose. Ce clic-ci réconcilie les deux : le son se met à jouer, la préférence ne bouge pas.
    if (onRef.current && !playerRef.current) {
      openPlayer()
      return
    }
    const next = !onRef.current
    onRef.current = next
    persistPreference(SOUND_ON_KEY, String(next))
    // DANS LE GESTE, toujours. Coupure IMMÉDIATE dans l'autre sens (rampe du maître) : ce qui
    // est en vol s'éteint avec, plutôt que de traîner une seconde après le clic.
    if (next) openPlayer()
    else playerRef.current?.setVolume(0)
    setOn(next)
  }, [openPlayer, hasAnySound])

  const setVolume = useCallback((v: number) => {
    const clamped = Math.min(Math.max(v, 0), 1)
    volumeRef.current = clamped
    setVolumeState(clamped)
    persistPreference(SOUND_VOLUME_KEY, String(clamped))
    if (onRef.current) playerRef.current?.setVolume(clamped)
  }, [])

  // La piste d'enregistrement se lit sur les REFS (`onRef`, `playerRef`) et non sur l'état :
  // elle est demandée au moment d'un clic sur « Enregistrer », jamais pendant un rendu, et
  // cette fonction doit rester stable pour ne pas recréer l'objet de capture à chaque son.
  const recordingTrack = useCallback((): MediaStreamTrack | null => {
    if (!onRef.current) return null
    return playerRef.current?.recordingTrack() ?? null
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
      // LE TIRAGE DE VARIANTE A LIEU ICI, pas à la construction de la piste : une piste est
      // bâtie une fois pour tout le match, y tirer ferait rejouer le même fichier à chaque
      // occurrence du geste — exactement ce que la variante évite (`replaySoundVariants.ts`).
      const stem = pickVariantStem(e)
      const url = urlsRef.current.get(stem)
      if (!url) continue
      // Le tirage de variation ne concerne que les ARMES : la table est keyee par stem
      // d'arme, tout autre stem se joue tel quel (drawVariation rend le neutre exact).
      player.play(url, drawVariation(WEAPON_SOUND_VARIATIONS[stem], tuning.variationPercentRef.current))
    }
  }, [tuning])

  // LA CONCLUSION. Elle ne passe PAS par le curseur : elle n'a pas d'instant sur la piste, et
  // le curseur existe pour ne pas rejouer ce qu'on a enjambé — une question qui n'a pas de sens
  // pour un événement qui n'arrive qu'au bout. C'est l'appelant qui garantit l'unicité (la
  // lecture n'atteint sa borne qu'une fois par passage, cf. `useReplayPlayback`).
  //
  // AUCUNE VARIATION RANGED ICI : ces fourchettes sont celles du moteur du jeu pour les ARMES
  // (weaponSoundVariations.ts). Une réplique d'annonceur et une fanfare se jouent telles quelles.
  const playEndMatch = useCallback(() => {
    const spec = endMatchRef.current
    const player = playerRef.current
    if (!spec || !onRef.current || !player || !soundPlaysAtSpeed(speedRef.current)) return
    for (const stem of endMatchSounds(spec.outcome, spec.ffa, spec.locale)) {
      const url = urlsRef.current.get(stem)
      if (url) player.play(url)
    }
  }, [])

  return {
    available: hasAnySound,
    on,
    toggle,
    wake,
    volume,
    setVolume,
    mutedBySpeed: on && !soundPlaysAtSpeed(speed),
    categories,
    toggleCategory,
    tick,
    endMatch: playEndMatch,
    recordingTrack,
  }
}
