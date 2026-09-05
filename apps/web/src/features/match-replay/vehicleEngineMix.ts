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
 *    la sortie est DANS le clip (ses fins sont gravées, il déborde alors en queue).
 */
import {
  ENGINE_CROSSFADE_S,
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

/** Les phases tenues d'un épisode, dans l'ordre : alternance loop/idle bornée à [a, b]. */
function heldPhases(ep: EngineEpisode, a: number, b: number): { stemKey: 'loop' | 'idle'; span: EngineSpan }[] {
  const cuts: { stemKey: 'loop' | 'idle'; span: EngineSpan }[] = []
  let cursor = a
  for (const s of ep.idle) {
    const i0 = Math.max(s.t0Ms, a)
    const i1 = Math.min(s.t1Ms, b)
    if (i1 <= i0) continue
    if (i0 > cursor) cuts.push({ stemKey: 'loop', span: { t0Ms: cursor, t1Ms: i0 } })
    cuts.push({ stemKey: 'idle', span: { t0Ms: i0, t1Ms: i1 } })
    cursor = i1
  }
  if (cursor < b) cuts.push({ stemKey: 'loop', span: { t0Ms: cursor, t1Ms: b } })
  return cuts
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
    const phases = heldPhases(ep, a, b)
    for (let i = 0; i < phases.length; i++) {
      const p = phases[i]
      const stem = p.stemKey === 'idle' ? stems.idle : stems.loop
      if (!stem || durOf(stem) === null) continue
      const first = i === 0
      const last = i === phases.length - 1
      const atS = first ? heldStart : rel(p.span.t0Ms)
      // La fin d'une phase : jonction interne (le suivant entre au même instant), sortie vers
      // `exit`, ou bord du clip — dans les trois cas le fondu de 150 ms termine la voix.
      const stopS = (last ? rel(b) : rel(p.span.t1Ms)) + ENGINE_CROSSFADE_S
      const clippedEnd = last && !exits
      out.push({
        stem,
        atS,
        loop: true,
        stopS: clippedEnd ? rel(b) : stopS,
        fadeInS: first ? heldFadeIn : ENGINE_CROSSFADE_S,
        fadeOutS: ENGINE_CROSSFADE_S,
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
    for (const s of [stems.enter, stems.loop, stems.idle, stems.exit]) {
      if (s) out.add(s)
    }
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
