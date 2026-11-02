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
 *
 * # LE MOTEUR NE SONNE QUE SI LE VÉHICULE BOUGE VRAIMENT (décision utilisateur du 2026-11-02)
 *
 * Un véhicule occupé mais immobile est SILENCIEUX : aucun clip de ralenti (« on va pas en
 * mettre, ça simplifie »). L'épisode commence au premier mouvement réel et finit au dernier ;
 * un arrêt plus court que `ENGINE_STOP_MERGE_MS` ne change rien (la course continue), un arrêt
 * plus long éteint la boucle par un fondu (`ENGINE_STOP_FADE_S`) et la rallume par un fondu
 * (`ENGINE_RESTART_FADE_S`), sans rejouer `enter` ni `exit` (2 à 7 s chacun : les rejouer à
 * chaque arrêt raconterait un moteur qu'on coupe et relance).
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
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleIsDecor } from '../model/vehiclesLayer'

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
 * Au-dessus de cette vitesse horizontale (unités du film par seconde, distance entre deux
 * échantillons / dt), le véhicule BOUGE. Mesure sur 500 rejeux (2026-11-02, conducteurs de
 * véhicules terrestres) : à l'arrêt la vitesse reste sous 0,05 ; la croisière va de 3 à 12. Le
 * seuil de 1 écarte la dérive d'un véhicule posé ou poussé sans compter comme un mouvement.
 */
export const VEHICLE_MOVE_SPEED = 1

/**
 * Un arrêt plus court que ceci (ms) ne coupe pas la boucle : un conducteur qui freine puis
 * repart ne fait pas clignoter le moteur. Plus long, la boucle s'éteint en fondu.
 */
export const ENGINE_STOP_MERGE_MS = 1_500

/**
 * Un mouvement (après fusion des arrêts courts) plus court que ceci (ms) n'allume rien : un
 * véhicule bousculé par une explosion n'a pas démarré.
 */
export const ENGINE_MOVE_MIN_MS = 500

/** Fondu d'extinction de la boucle quand le véhicule s'arrête, en secondes. */
export const ENGINE_STOP_FADE_S = 0.6

/** Fondu de reprise de la boucle quand le véhicule repart, en secondes. */
export const ENGINE_RESTART_FADE_S = 0.3

/** Les clips d'une famille. */
export interface EngineStems {
  enter: string
  loop: string
  exit: string
}

/**
 * LA TABLE FAMILLE -> STEMS, alignée sur `family` du document (`VehicleTrack.family`) et sur
 * les fichiers `static/sounds/halo_infinite/vehicle_<famille>_<clip>.ogg` (garde-rail
 * `replaySoundAssets.guard.test.ts` : manifeste et dossier sont la même liste). Une famille
 * absente de cette table (tourelles, châssis non résolu) est un SILENCE PROPRE.
 *
 * LE FALCON SONNE DEPUIS LE 2026-09-24 : il est sorti des familles de DÉCOR
 * (`FAMILLES_NON_JOUABLES`, décision utilisateur « le Falcon ça dépend »), et un Falcon occupé
 * joue son moteur comme tout véhicule. Un Falcon sans occupant n'a pas de moteur, qu'il soit
 * garé, de décor de carte (`vehicleIsScenery`, lot M7) ou planant en décor de Behemoth.
 */
export const VEHICLE_ENGINE_STEMS: Readonly<Record<string, EngineStems>> = {
  warthog: { enter: 'vehicle_warthog_enter', loop: 'vehicle_warthog_loop', exit: 'vehicle_warthog_exit' },
  scorpion: { enter: 'vehicle_scorpion_enter', loop: 'vehicle_scorpion_loop', exit: 'vehicle_scorpion_exit' },
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
  return Object.values(VEHICLE_ENGINE_STEMS).flatMap((s) => [s.enter, s.loop, s.exit])
}

/** Une période fermée sur l'axe du rejeu, en millisecondes. */
export interface EngineSpan {
  t0Ms: number
  t1Ms: number
}

/**
 * Un épisode moteur : du premier mouvement (`t0Ms`, `enter`) au dernier (`t1Ms`, `exit`),
 * silencieux pendant ses `arrets`.
 */
export interface EngineEpisode extends EngineSpan {
  /** Les arrêts longs, strictement intérieurs à l'épisode, triés : la boucle s'y tait. */
  arrets: EngineSpan[]
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
 * movingSpansOf — les périodes où le véhicule BOUGE VRAIMENT, bornées à `span`, triées.
 *
 * La vitesse se dérive des échantillons de trajectoire (distance horizontale / dt — la seule
 * cinématique que le film publie) : un intervalle entre deux échantillons compte s'il dépasse
 * `VEHICLE_MOVE_SPEED`. Les intervalles séparés de moins de `ENGINE_STOP_MERGE_MS` se fusionnent
 * (un arrêt court ne coupe rien) ; un mouvement fusionné plus court que `ENGINE_MOVE_MIN_MS` est
 * écarté. Sans échantillon qui bouge, la liste est vide : un véhicule immobile se tait.
 */
export function movingSpansOf(
  samples: readonly { t: number; x: number; y: number }[],
  span: EngineSpan,
  frameMs: number,
): EngineSpan[] {
  const merged: EngineSpan[] = []
  for (let i = 1; i < samples.length; i++) {
    const t0Ms = Math.max(samples[i - 1].t * frameMs, span.t0Ms)
    const t1Ms = Math.min(samples[i].t * frameMs, span.t1Ms)
    if (t1Ms <= t0Ms) continue
    const dtS = ((samples[i].t - samples[i - 1].t) * frameMs) / 1000
    const v = Math.hypot(samples[i].x - samples[i - 1].x, samples[i].y - samples[i - 1].y) / dtS
    if (v <= VEHICLE_MOVE_SPEED) continue
    const last = merged[merged.length - 1]
    if (last && t0Ms - last.t1Ms < ENGINE_STOP_MERGE_MS) last.t1Ms = Math.max(last.t1Ms, t1Ms)
    else merged.push({ t0Ms, t1Ms })
  }
  return merged.filter((s) => s.t1Ms - s.t0Ms >= ENGINE_MOVE_MIN_MS)
}

/**
 * episodeOfMoves — l'épisode moteur d'une occupation : du premier mouvement au dernier, les
 * trous entre mouvements devenant des arrêts. `null` si le véhicule n'a pas bougé.
 */
export function episodeOfMoves(moves: readonly EngineSpan[]): EngineEpisode | null {
  if (moves.length === 0) return null
  const arrets: EngineSpan[] = []
  for (let i = 1; i < moves.length; i++) arrets.push({ t0Ms: moves[i - 1].t1Ms, t1Ms: moves[i].t0Ms })
  return { t0Ms: moves[0].t0Ms, t1Ms: moves[moves.length - 1].t1Ms, arrets }
}

/**
 * planVehicleEngines — LE PLAN COMPLET d'un document : un plan par véhicule qui sonne.
 *
 * QUATRE REFUS, dans l'ordre : le DÉCOR (jamais de son — décision de cadrage n° 1, même
 * ensemble que le calque), la famille SANS BANQUE (silence propre — n° 2), le véhicule
 * JAMAIS OCCUPÉ (un moteur ne tourne que pendant l'union des rides), et l'occupation SANS
 * MOUVEMENT (un véhicule immobile se tait, décision utilisateur du 2026-11-02).
 */
export function planVehicleEngines(
  vehicles: readonly ReplayVehicleTrackReady[],
  frameMs: number,
): EnginePlan[] {
  const out: EnginePlan[] = []
  for (const v of vehicles) {
    if (!v.family || vehicleIsDecor(v.family)) continue
    if (!VEHICLE_ENGINE_STEMS[v.family]) continue
    const episodes: EngineEpisode[] = []
    for (const span of mergeRideSpans(v.rides, frameMs)) {
      const ep = episodeOfMoves(movingSpansOf(v.samples, span, frameMs))
      if (ep) episodes.push(ep)
    }
    if (episodes.length > 0) out.push({ family: v.family, episodes })
  }
  return out
}

/** Ce qu'un moteur DOIT jouer à un instant donné, hors transitions (le lecteur pose celles-ci). */
export type EnginePhase = 'loop' | 'arret' | null

/** enginePhaseAt — l'état nominal d'un épisode à `ms` : course, arrêt (silence), ou rien. */
export function enginePhaseAt(episode: EngineEpisode, ms: number): EnginePhase {
  if (ms < episode.t0Ms || ms >= episode.t1Ms) return null
  for (const s of episode.arrets) {
    if (ms >= s.t0Ms && ms < s.t1Ms) return 'arret'
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
