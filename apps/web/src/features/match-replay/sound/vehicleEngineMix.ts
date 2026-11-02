/**
 * vehicleEngineMix.ts — LES MOTEURS DANS L'EXPORT HORS TEMPS RÉEL : la MÊME logique que la
 * page (mêmes bornes d'épisodes fusionnés, mêmes fondus croisés de 150 ms à puissance
 * constante, même bus à 0,85), posée d'avance dans l'`OfflineAudioContext` du mixage.
 *
 * LE PLAN EST CELUI DE LA PAGE (`vehicleEngineSound.planVehicleEngines`, transmis par
 * `useReplaySound.exportTrack`) : le clip ne peut ni inventer un moteur ni en manquer un —
 * exactement la doctrine de `replayAudioMix.ts` pour la piste d'événements.
 *
 * CE QUI CHANGE PAR RAPPORT AU TEMPS RÉEL, ET POURQUOI :
 *  - tout est PRÉ-CALCULÉ en SEGMENTS purs (`planEngineSegments`, testés) : hors ligne, il
 *    n'y a pas de curseur à suivre — chaque source connaît son instant, sa boucle et ses
 *    fondus dès la planification ;
 *  - un épisode COUPÉ par la borne de début reprend directement sur `loop` (même règle que
 *    le seek : jamais de re-enter pour un démarrage que le clip ne montre pas) ;
 *  - un épisode COUPÉ par la borne de fin s'éteint par le fondu de jonction (150 ms) qui se
 *    termine PILE sur la borne — pas d'enveloppe ajoutée à un `exit`, qui ne joue ici que si
 *    la sortie est DANS le clip (ses fins sont gravées, il déborde alors en queue) ;
 *  - un ARRÊT LONG ne pose aucune source : la boucle qui le précède finit par le fondu
 *    d'extinction (`ENGINE_STOP_FADE_S`), celle qui le suit commence par le fondu de reprise
 *    (`ENGINE_RESTART_FADE_S`) — les mêmes que la page.
 */
import {
  ENGINE_CROSSFADE_S,
  ENGINE_RESTART_FADE_S,
  ENGINE_STOP_FADE_S,
  equalPowerCurves,
  VEHICLE_ENGINE_BUS_GAIN,
  VEHICLE_ENGINE_STEMS,
  type EngineEpisode,
  type EnginePlan,
  type EngineSpan,
} from './vehicleEngineSound'

/** Une source moteur posée sur l'axe du CLIP (secondes). */
export interface EngineSegment {
  stem: string
  /** Départ sur l'axe du clip. Peut être négatif de moins d'un fondu : borné au rendu. */
  atS: number
  loop: boolean
  /** Fin forcée (boucles et enter) ; absent = la source joue son fichier entier (exit). */
  stopS?: number
  /** Fondu d'entrée à puissance constante, depuis `atS`. */
  fadeInS?: number
  /** Fondu de sortie à puissance constante, se terminant à `stopS`. */
  fadeOutS?: number
}

/** Les périodes de COURSE d'un épisode bornées à [a, b], dans l'ordre (les arrêts en moins). */
function courseSpans(ep: EngineEpisode, a: number, b: number): EngineSpan[] {
  const out: EngineSpan[] = []
  let cursor = a
  for (const s of ep.arrets) {
    const i0 = Math.max(s.t0Ms, a)
    const i1 = Math.min(s.t1Ms, b)
    if (i1 <= i0) continue
    if (i0 > cursor) out.push({ t0Ms: cursor, t1Ms: i0 })
    cursor = i1
  }
  if (cursor < b) out.push({ t0Ms: cursor, t1Ms: b })
  return out
}

/**
 * planEngineSegments — les segments d'UN plan de véhicule, bornés à la plage exportée.
 *
 * `durOf` rend la durée décodée d'un stem (l'enter en a besoin pour poser sa jonction), ou
 * `null` si le fichier est absent — le segment concerné saute, silence propre comme partout.
 */
export function planEngineSegments(
  plan: EnginePlan,
  bounds: { startMs: number; endMs: number },
  durOf: (stem: string) => number | null,
): EngineSegment[] {
  const stems = VEHICLE_ENGINE_STEMS[plan.family]
  if (!stems) return []
  const out: EngineSegment[] = []
  const rel = (ms: number) => (ms - bounds.startMs) / 1000
  for (const ep of plan.episodes) {
    const a = Math.max(ep.t0Ms, bounds.startMs)
    const b = Math.min(ep.t1Ms, bounds.endMs)
    if (b <= a) continue
    const entered = ep.t0Ms >= bounds.startMs
    const enterDur = entered ? durOf(stems.enter) : null
    // La voix tenue démarre à la jonction de l'enter quand il joue, au bord du clip sinon.
    let heldStart = rel(a)
    let heldFadeIn = 0
    if (enterDur !== null && enterDur > 0) {
      const fadeAt = rel(a) + Math.max(enterDur - ENGINE_CROSSFADE_S, 0)
      out.push({
        stem: stems.enter,
        atS: rel(a),
        loop: false,
        stopS: fadeAt + ENGINE_CROSSFADE_S,
        fadeOutS: ENGINE_CROSSFADE_S,
      })
      heldStart = fadeAt
      heldFadeIn = ENGINE_CROSSFADE_S
    }
    const exits = ep.t1Ms <= bounds.endMs
    const courses = durOf(stems.loop) === null ? [] : courseSpans(ep, a, b)
    for (let i = 0; i < courses.length; i++) {
      const c = courses[i]
      // La course qui ouvre l'épisode prend la jonction de l'enter (ou le bord du clip) ; les
      // suivantes REPARTENT d'un arrêt.
      const opens = c.t0Ms === a
      const atS = opens ? heldStart : rel(c.t0Ms)
      const fadeInS = opens ? heldFadeIn : ENGINE_RESTART_FADE_S
      // La fin d'une course : vers `exit` (fondu croisé), vers un arrêt (fondu d'extinction),
      // ou coupée net par le bord du clip.
      const closes = c.t1Ms === b
      const fadeOutS = closes ? ENGINE_CROSSFADE_S : ENGINE_STOP_FADE_S
      out.push({
        stem: stems.loop,
        atS,
        loop: true,
        stopS: closes && !exits ? rel(b) : rel(c.t1Ms) + fadeOutS,
        fadeInS,
        fadeOutS,
      })
    }
    if (exits && durOf(stems.exit) !== null) {
      out.push({ stem: stems.exit, atS: rel(ep.t1Ms), loop: false, fadeInS: ENGINE_CROSSFADE_S })
    }
  }
  return out
}

/** Tous les stems dont les segments d'un jeu de plans peuvent avoir besoin (décodage). */
export function engineStemsOf(plans: readonly EnginePlan[]): string[] {
  const out = new Set<string>()
  for (const p of plans) {
    const stems = VEHICLE_ENGINE_STEMS[p.family]
    if (!stems) continue
    for (const s of [stems.enter, stems.loop, stems.exit]) out.add(s)
  }
  return [...out]
}

/**
 * engineTailSeconds — ce que les moteurs DÉBORDENT après la borne du clip : la queue d'un
 * `exit` gravé qui commence près de la fin. Même rôle que `tailSeconds` pour les événements.
 */
export function engineTailSeconds(
  segments: readonly EngineSegment[],
  durOf: (stem: string) => number | null,
  clipDurS: number,
): number {
  let tail = 0
  for (const s of segments) {
    const end = s.stopS ?? s.atS + (durOf(s.stem) ?? 0)
    tail = Math.max(tail, end - clipDurS)
  }
  return Math.max(0, tail)
}

/**
 * scheduleEngineMix pose les segments dans le contexte hors ligne, derrière le bus moteur
 * (0,85 — la même retenue que la page, décision utilisateur du 2026-09-04).
 */
export function scheduleEngineMix(
  ctx: BaseAudioContext,
  destination: AudioNode,
  segments: readonly EngineSegment[],
  buffers: ReadonlyMap<string, AudioBuffer | null>,
): void {
  if (segments.length === 0) return
  const bus = ctx.createGain()
  bus.gain.value = VEHICLE_ENGINE_BUS_GAIN
  bus.connect(destination)
  const curves = equalPowerCurves()
  for (const s of segments) {
    const buf = buffers.get(s.stem)
    if (!buf) continue
    const at = Math.max(s.atS, 0)
    const gain = ctx.createGain()
    gain.gain.setValueAtTime(s.fadeInS ? 0 : 1, at)
    if (s.fadeInS) gain.gain.setValueCurveAtTime(curves.fadeIn, at, s.fadeInS)
    if (s.stopS !== undefined && s.fadeOutS) {
      gain.gain.setValueCurveAtTime(curves.fadeOut, Math.max(s.stopS - s.fadeOutS, at), s.fadeOutS)
    }
    const src = ctx.createBufferSource()
    src.buffer = buf
    src.loop = s.loop
    src.connect(gain)
    gain.connect(bus)
    src.start(at)
    if (s.stopS !== undefined) src.stop(Math.max(s.stopS, at))
  }
}
