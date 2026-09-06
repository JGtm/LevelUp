/**
 * vehicleEngineSound.ts — LE PLAN SONORE DES MOTEURS DE VÉHICULES : quelles familles sonnent,
 * quand un moteur tourne, et comment ses clips s'enchaînent. Logique PURE, sans Web Audio ni
 * React — la lecture temps réel vit dans `vehicleEnginePlayer.ts`, le mixage d'export dans
 * `vehicleEngineMix.ts`, les deux consomment CE plan.
 *
 * # POURQUOI UN MODULE À PART, ET PAS `replaySound.ts`
 *
 * La piste du rejeu est une suite d'ÉVÉNEMENTS one-shot (un tir, un kill, un jingle) que le
 * curseur déclenche au passage. Un moteur n'est pas un événement : c'est un ÉTAT CONTINU —
 * il démarre, boucle tant que le véhicule est occupé, et s'arrête. Le lecteur d'événements
 * (`ReplayAudioPlayer.play`, enveloppe tenue + fondu) ne sait pas tenir un état, et lui
 * apprendre casserait sa seule règle (« la durée jouée est celle du fichier »).
 *
 * # LE CONTRAT DE LA BANQUE (HANDOFF SFX_vehicules, 2026-09-04 — NON NÉGOCIABLE)
 *
 *   enter -> loop x N -> exit
 *
 *  - Les BOUCLES rebouclent sans clic : le fondu est DANS le fichier (`loop = true` sur la
 *    source, rien à ajouter au rebouclage).
 *  - Les JONCTIONS entre clips se font par fondu croisé de 150 ms À PUISSANCE CONSTANTE
 *    (sin/cos, jamais linéaire — un fondu linéaire entre signaux décorrélés creuse l'énergie
 *    de 3 dB pile au raccord, mesuré sur le Warthog et le Chopper pendant la production).
 *  - Les FINS SONT GRAVÉES : chaque `exit` finit à zéro tout seul. AUCUNE enveloppe de sortie
 *    à ajouter — c'est l'inverse exact de la règle des sons d'événement.
 *  - `scorpion_idle` est le moteur au ralenti, véhicule occupé et immobile : il REMPLACE la
 *    boucle de course, fondu croisé de 150 ms dans les deux sens.
 *
 * # LE BOOST N'EST PAS CÂBLÉ, ET C'EST UNE MESURE, PAS UN OUBLI (premiere CI du chantier : 2026-09-05)
 *
 * Décision de cadrage du 2026-09-04 : ne câbler le boost QUE pour ghost et chopper
 * (wraith_boost est RÉFUTÉ par l'utilisateur — le clip n'est pas un boost ; banshee n'a pas
 * de clip), avec un gate chiffré écrit AVANT la mesure : un boost = vitesse soutenue
 * > k x régime de croisière propre du véhicule pendant >= N frames, et un véhicule SANS boost
 * (mongoose, warthog) ne doit produire AUCUNE détection.
 *
 * MESURE (2026-09-04, artefacts de démo 0d76e8f1 et fccc61cd, vitesse = distance entre
 * échantillons / dt, croisière = médiane des vitesses en mouvement pendant l'occupation) :
 *  - k = 1,3, N = 15 frames (1,5 s) : le TÉMOIN mongoose détecte (1,5 s sur 0d76e8f1,
 *    2,7 s sur fccc61cd) — faux positifs, gate invalide ;
 *  - k = 1,5 : témoin propre, mais le SEUL ghost occupé du corpus (slot 777) ne détecte
 *    RIEN non plus (croisière 8,72, p90 9,10 — la trajectoire échantillonnée à 10 Hz lisse
 *    le boost sous le bruit de conduite).
 * Aucun couple (k, N) ne sépare donc un boost d'une descente de pente sur ces données : la
 * détection par vitesse N'EST PAS FIABLE, et le cadrage (§4) prescrit alors de livrer SANS
 * boost. Les clips `ghost_boost`/`chopper_boost` restent dans la banque source, non copiés —
 * un fichier livré que rien ne joue serait un asset mort (garde-rail d'assets). Condition de
 * reprise : une mesure du boost qui ne passe pas par la vitesse (un champ du film, ou un
 * échantillonnage plus fin).
 */
import type { ReplayVehicleTrackReady } from '../replayNormalize'
import { vehicleIsDecor } from '../vehiclesLayer'

/**
 * LE BUS MOTEUR EST À 0,85 x LE MAÎTRE (décision utilisateur du 2026-09-04 : « les moteurs
 * ne doivent pas être au niveau des autres sons »). Un seul nœud de gain, posé entre les
 * sources moteur et le maître : les moteurs suivent donc le volume réglé, la chaîne de
 * distance et le robinet d'enregistrement comme tout le monde, 15 % en retrait.
 */
export const VEHICLE_ENGINE_BUS_GAIN = 0.85

/** Fondu croisé aux jonctions entre clips, en secondes (contrat de la banque, §2). */
export const ENGINE_CROSSFADE_S = 0.15

/**
 * Rampe d'arrêt d'urgence, en secondes : pause, coupure du son, démontage. Ce n'est PAS une
 * enveloppe de sortie musicale (interdite par le contrat) — c'est l'anti-clic minimal d'un
 * arrêt que la timeline n'a pas raconté, même durée que `VOLUME_RAMP_S` du lecteur.
 */
export const ENGINE_STOP_RAMP_S = 0.02

/**
 * Un trou D'OCCUPATION plus court que ceci ne coupe pas le moteur (décision de cadrage n° 1) :
 * les épisodes d'un même véhicule se FUSIONNENT. Un conducteur qui descend deux secondes pour
 * ramasser un drapeau n'éteint pas le moteur à l'oreille du spectateur.
 */
export const ENGINE_RIDE_GAP_MERGE_MS = 2_000

/**
 * SCORPION AU RALENTI : sous cette vitesse (unités monde/s — les coordonnées du film sont
 * métriques, cf. les mesures de `vehiclesLayer.ts`), un Scorpion occupé joue `idle` au lieu
 * de `loop`. RÉSERVE ÉCRITE : aucun Scorpion dans les artefacts de démo du 2026-09-04 — le
 * seuil vient du raisonnement physique (un tank « quasi immobile » avance sous 0,5 m/s, la
 * moitié d'un pas humain) et non d'une mesure ; à recaler à la première écoute réelle.
 */
export const SCORPION_IDLE_SPEED = 0.5

/**
 * Durée minimale d'un état ralenti/course avant d'en changer, en millisecondes : sans cette
 * hystérésis, un Scorpion qui manœuvre ferait clignoter idle/loop plus vite que le fondu de
 * 150 ms qui les raccorde.
 */
export const SCORPION_IDLE_MIN_MS = 1_000

/** Les clips d'une famille. `idle` n'existe que pour le Scorpion (contrat de la banque). */
export interface EngineStems {
  enter: string
  loop: string
  exit: string
  idle?: string
}

/**
 * LA TABLE FAMILLE -> STEMS, alignée sur `family` du document (`VehicleTrack.family`) et sur
 * les fichiers `static/sounds/halo_infinite/vehicle_<famille>_<clip>.wav` (garde-rail
 * `replaySoundAssets.guard.test.ts` : manifeste et dossier sont la même liste). Une famille
 * absente de cette table (tourelles, châssis non résolu) est un SILENCE PROPRE.
 *
 * LE FALCON EST DANS LA TABLE ET NE SONNE JAMAIS AUJOURD'HUI : c'est une famille de DÉCOR
 * (`FAMILLES_NON_JOUABLES`), et le plan refuse le décor AVANT de consulter la table. La banque
 * le livre quand même (il est pilotable dans d'autres contextes) : le référencer ici garde ses
 * fichiers vivants aux yeux du garde-rail sans qu'aucun décor ne sonne.
 */
export const VEHICLE_ENGINE_STEMS: Readonly<Record<string, EngineStems>> = {
  warthog: { enter: 'vehicle_warthog_enter', loop: 'vehicle_warthog_loop', exit: 'vehicle_warthog_exit' },
  scorpion: {
    enter: 'vehicle_scorpion_enter',
    loop: 'vehicle_scorpion_loop',
    exit: 'vehicle_scorpion_exit',
    idle: 'vehicle_scorpion_idle',
  },
  mongoose: { enter: 'vehicle_mongoose_enter', loop: 'vehicle_mongoose_loop', exit: 'vehicle_mongoose_exit' },
  chopper: { enter: 'vehicle_chopper_enter', loop: 'vehicle_chopper_loop', exit: 'vehicle_chopper_exit' },
  falcon: { enter: 'vehicle_falcon_enter', loop: 'vehicle_falcon_loop', exit: 'vehicle_falcon_exit' },
  banshee: { enter: 'vehicle_banshee_enter', loop: 'vehicle_banshee_loop', exit: 'vehicle_banshee_exit' },
  wraith: { enter: 'vehicle_wraith_enter', loop: 'vehicle_wraith_loop', exit: 'vehicle_wraith_exit' },
  wasp: { enter: 'vehicle_wasp_enter', loop: 'vehicle_wasp_loop', exit: 'vehicle_wasp_exit' },
  ghost: { enter: 'vehicle_ghost_enter', loop: 'vehicle_ghost_loop', exit: 'vehicle_ghost_exit' },
}

/** Tous les stems moteur, pour le préchargement et le garde-rail d'assets. */
export function allEngineStems(): string[] {
  return Object.values(VEHICLE_ENGINE_STEMS).flatMap((s) =>
    s.idle ? [s.enter, s.loop, s.idle, s.exit] : [s.enter, s.loop, s.exit],
  )
}

/** Une période fermée sur l'axe du rejeu, en millisecondes. */
export interface EngineSpan {
  t0Ms: number
  t1Ms: number
}

/** Un épisode moteur : le véhicule sonne de `t0Ms` à `t1Ms`, ralenti sur `idle`. */
export interface EngineEpisode extends EngineSpan {
  /** Les périodes RALENTI (Scorpion seul aujourd'hui), déjà bornées à l'épisode, triées. */
  idle: EngineSpan[]
}

/** Le plan moteur d'UN véhicule : sa famille (clé de `VEHICLE_ENGINE_STEMS`) et ses épisodes. */
export interface EnginePlan {
  family: string
  episodes: EngineEpisode[]
}

/**
 * mergeRideSpans — L'UNION des épisodes d'occupation (décision de cadrage n° 1) : les rides
 * peuvent se CHEVAUCHER (plusieurs passagers) et se suivre de près (changement de conducteur) ;
 * le moteur, lui, sonne d'un seul tenant. Un trou < `gapMergeMs` ne le coupe pas.
 */
export function mergeRideSpans(
  rides: readonly { t0: number; t1: number }[],
  frameMs: number,
  gapMergeMs = ENGINE_RIDE_GAP_MERGE_MS,
): EngineSpan[] {
  const sorted = rides
    .filter((r) => r.t1 > r.t0)
    .map((r) => ({ t0Ms: r.t0 * frameMs, t1Ms: r.t1 * frameMs }))
    .sort((a, b) => a.t0Ms - b.t0Ms)
  const out: EngineSpan[] = []
  for (const s of sorted) {
    const last = out[out.length - 1]
    if (last && s.t0Ms - last.t1Ms < gapMergeMs) {
      last.t1Ms = Math.max(last.t1Ms, s.t1Ms)
    } else {
      out.push({ ...s })
    }
  }
  return out
}

/**
 * idleSpansOf — les périodes où un véhicule OCCUPÉ est quasi immobile, bornées à l'épisode.
 *
 * La vitesse se dérive des échantillons de trajectoire (distance / dt — la seule cinématique
 * que le film publie). Un état plus court que `minMs` est absorbé par son voisin : l'hystérésis
 * qui empêche le clignotement idle/loop.
 */
export function idleSpansOf(
  samples: readonly { t: number; x: number; y: number }[],
  episode: EngineSpan,
  frameMs: number,
  speedMax = SCORPION_IDLE_SPEED,
  minMs = SCORPION_IDLE_MIN_MS,
): EngineSpan[] {
  const raw: EngineSpan[] = []
  let cur: EngineSpan | null = null
  for (let i = 1; i < samples.length; i++) {
    const tMs = samples[i].t * frameMs
    if (tMs < episode.t0Ms || tMs > episode.t1Ms) continue
    const dtS = ((samples[i].t - samples[i - 1].t) * frameMs) / 1000
    if (dtS <= 0) continue
    const v = Math.hypot(samples[i].x - samples[i - 1].x, samples[i].y - samples[i - 1].y) / dtS
    if (v <= speedMax) {
      if (cur) cur.t1Ms = tMs
      else cur = { t0Ms: Math.max(samples[i - 1].t * frameMs, episode.t0Ms), t1Ms: tMs }
    } else if (cur) {
      raw.push(cur)
      cur = null
    }
  }
  if (cur) raw.push(cur)
  return raw.filter((s) => s.t1Ms - s.t0Ms >= minMs)
}

/**
 * planVehicleEngines — LE PLAN COMPLET d'un document : un plan par véhicule qui sonne.
 *
 * TROIS REFUS, dans l'ordre : le DÉCOR (jamais de son — décision de cadrage n° 1, même
 * ensemble que le calque), la famille SANS BANQUE (silence propre — n° 2), et le véhicule
 * JAMAIS OCCUPÉ (un moteur ne tourne que pendant l'union des rides).
 */
export function planVehicleEngines(
  vehicles: readonly ReplayVehicleTrackReady[],
  frameMs: number,
): EnginePlan[] {
  const out: EnginePlan[] = []
  for (const v of vehicles) {
    if (!v.family || vehicleIsDecor(v.family)) continue
    const stems = VEHICLE_ENGINE_STEMS[v.family]
    if (!stems) continue
    const spans = mergeRideSpans(v.rides, frameMs)
    if (spans.length === 0) continue
    const episodes: EngineEpisode[] = spans.map((span) => ({
      ...span,
      idle: stems.idle ? idleSpansOf(v.samples, span, frameMs) : [],
    }))
    out.push({ family: v.family, episodes })
  }
  return out
}

/** Ce qu'un moteur DOIT jouer à un instant donné, hors transitions (le lecteur pose celles-ci). */
export type EnginePhase = 'loop' | 'idle' | null

/** enginePhaseAt — l'état nominal d'un épisode à `ms` : course, ralenti, ou rien. */
export function enginePhaseAt(episode: EngineEpisode, ms: number): EnginePhase {
  if (ms < episode.t0Ms || ms >= episode.t1Ms) return null
  for (const s of episode.idle) {
    if (ms >= s.t0Ms && ms < s.t1Ms) return 'idle'
  }
  return 'loop'
}

/**
 * equalPowerCurves — les deux rampes d'un fondu croisé À PUISSANCE CONSTANTE (sin/cos).
 *
 * C'est LE piège documenté par la banque (§5 du HANDOFF, deux itérations perdues) : un fondu
 * linéaire entre deux signaux décorrélés creuse l'énergie de 3 dB au raccord. sin²+cos² = 1 —
 * l'énergie totale reste plate sur toute la jonction. Les courbes servent
 * `setValueCurveAtTime` côté temps réel comme côté export.
 */
export function equalPowerCurves(steps = 32): { fadeIn: Float32Array; fadeOut: Float32Array } {
  const fadeIn = new Float32Array(steps)
  const fadeOut = new Float32Array(steps)
  for (let i = 0; i < steps; i++) {
    const phi = (i / (steps - 1)) * (Math.PI / 2)
    fadeIn[i] = Math.sin(phi)
    fadeOut[i] = Math.cos(phi)
  }
  return { fadeIn, fadeOut }
}
