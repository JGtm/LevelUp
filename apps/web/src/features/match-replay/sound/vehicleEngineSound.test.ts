/**
 * Tests du PLAN moteur des véhicules (vehicleEngineSound.ts) : fusion des épisodes
 * d'occupation, moteur qui suit le mouvement, refus du décor, table des stems, et le fondu croisé à
 * puissance constante — les règles du cadrage du 2026-09-04 et du contrat de la banque.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  ENGINE_RIDE_GAP_MERGE_MS,
  enginePhaseAt,
  equalPowerCurves,
  ENGINE_MOVE_MIN_MS,
  ENGINE_STOP_MERGE_MS,
  episodeOfMoves,
  mergeRideSpans,
  movingSpansOf,
  planVehicleEngines,
  VEHICLE_ENGINE_BUS_GAIN,
  VEHICLE_MOVE_SPEED,
  VEHICLE_ENGINE_STEMS,
  allEngineStems,
} from './vehicleEngineSound'

/** Une frame = 100 ms, la cadence des artefacts réels (frameIntervalMs du schéma). */
const FRAME_MS = 100

const ride = (t0: number, t1: number) => ({ t0, t1, slot: 1, src: 'x' })

function track(family: string, rides: { t0: number; t1: number }[], samples: { t: number; x: number; y: number }[] = []): ReplayVehicleTrackReady {
  return { family, rides: rides.map((r) => ({ ...ride(r.t0, r.t1), aim: [] })), samples } as unknown as ReplayVehicleTrackReady
}

describe('mergeRideSpans — l union des épisodes d occupation', () => {
  it('fusionne un trou plus court que la tolérance (le moteur ne se coupe pas)', () => {
    // 19 frames de trou = 1,9 s < 2 s : un seul épisode.
    expect(mergeRideSpans([ride(0, 50), ride(69, 100)], FRAME_MS)).toEqual([
      { t0Ms: 0, t1Ms: 10_000 },
    ])
  })

  it('coupe sur un trou à la tolérance ou au-delà', () => {
    // 20 frames = 2,0 s : deux épisodes.
    expect(mergeRideSpans([ride(0, 50), ride(70, 100)], FRAME_MS)).toEqual([
      { t0Ms: 0, t1Ms: 5_000 },
      { t0Ms: 7_000, t1Ms: 10_000 },
    ])
  })

  it('unit des rides chevauchés (plusieurs passagers) et trie les entrées', () => {
    expect(mergeRideSpans([ride(30, 80), ride(0, 50), ride(40, 60)], FRAME_MS)).toEqual([
      { t0Ms: 0, t1Ms: 8_000 },
    ])
  })

  it('ignore un ride vide ou inversé', () => {
    expect(mergeRideSpans([ride(10, 10), ride(20, 5)], FRAME_MS)).toEqual([])
  })

  it('la tolérance est bien la constante du cadrage (2 s)', () => {
    expect(ENGINE_RIDE_GAP_MERGE_MS).toBe(2_000)
  })
})

/**
 * Une trajectoire : en mouvement (5 unités/s, soit 0,5 par frame) sur les plages de frames
 * `roule`, immobile ailleurs.
 */
function trajet(nFrames: number, roule: [number, number][]): { t: number; x: number; y: number }[] {
  const out = [{ t: 0, x: 0, y: 0 }]
  for (let t = 1; t <= nFrames; t++) {
    const bouge = roule.some(([a, b]) => t > a && t <= b)
    out.push({ t, x: out[t - 1].x + (bouge ? 0.5 : 0), y: 0 })
  }
  return out
}

describe('movingSpansOf et episodeOfMoves — le moteur suit le mouvement', () => {
  const span = { t0Ms: 0, t1Ms: 10_000 }

  it('immobile du début à la fin : aucun mouvement, aucun épisode', () => {
    expect(movingSpansOf(trajet(100, []), span, FRAME_MS)).toEqual([])
    expect(episodeOfMoves([])).toBeNull()
  })

  it('monte, attend 2 s, roule 3 s, s arrête : l épisode tient le seul mouvement', () => {
    const moves = movingSpansOf(trajet(100, [[20, 50]]), span, FRAME_MS)
    expect(moves).toEqual([{ t0Ms: 2_000, t1Ms: 5_000 }])
    expect(episodeOfMoves(moves)).toEqual({ t0Ms: 2_000, t1Ms: 5_000, arrets: [] })
  })

  it('un arrêt court (1 s < 1,5 s) ne coupe rien', () => {
    const moves = movingSpansOf(trajet(100, [[0, 30], [40, 70]]), span, FRAME_MS)
    expect(moves).toEqual([{ t0Ms: 0, t1Ms: 7_000 }])
    expect(ENGINE_STOP_MERGE_MS).toBe(1_500)
  })

  it('un arrêt long (3 s) devient un arrêt intérieur de l épisode', () => {
    const moves = movingSpansOf(trajet(100, [[0, 30], [60, 90]]), span, FRAME_MS)
    expect(episodeOfMoves(moves)).toEqual({
      t0Ms: 0,
      t1Ms: 9_000,
      arrets: [{ t0Ms: 3_000, t1Ms: 6_000 }],
    })
  })

  it('une secousse (0,3 s de mouvement) n allume rien ; la dérive lente non plus', () => {
    expect(movingSpansOf(trajet(100, [[20, 23]]), span, FRAME_MS)).toEqual([])
    const derive = Array.from({ length: 101 }, (_, t) => ({ t, x: t * 0.05, y: 0 })) // 0,5 unité/s
    expect(movingSpansOf(derive, span, FRAME_MS)).toEqual([])
    expect(VEHICLE_MOVE_SPEED).toBe(1)
    expect(ENGINE_MOVE_MIN_MS).toBe(500)
  })

  it('borne les mouvements à l occupation', () => {
    const moves = movingSpansOf(trajet(100, [[0, 100]]), { t0Ms: 2_000, t1Ms: 4_000 }, FRAME_MS)
    expect(moves).toEqual([{ t0Ms: 2_000, t1Ms: 4_000 }])
  })
})

describe('planVehicleEngines — quoi sonne, quoi se tait', () => {
  const roule = trajet(60, [[0, 60]])

  it('refuse le décor (pelican), la famille inconnue, et le véhicule jamais occupé', () => {
    const plans = planVehicleEngines(
      [
        track('pelican', [{ t0: 0, t1: 100 }], roule), // décor : jamais de son (cadrage n° 1)
        track('shade_turret', [{ t0: 0, t1: 100 }], roule), // sans banque : silence propre (n° 2)
        track('warthog', [], roule), // jamais occupé : pas de moteur
        track('ghost', [{ t0: 10, t1: 50 }], roule),
      ],
      FRAME_MS,
    )
    expect(plans).toEqual([
      { family: 'ghost', episodes: [{ t0Ms: 1_000, t1Ms: 5_000, arrets: [] }] },
    ])
  })

  it('un Falcon OCCUPÉ sonne : il n’est plus une famille de décor (décision du 2026-09-24)', () => {
    const plans = planVehicleEngines([track('falcon', [{ t0: 10, t1: 50 }], roule)], FRAME_MS)
    expect(plans).toEqual([
      { family: 'falcon', episodes: [{ t0Ms: 1_000, t1Ms: 5_000, arrets: [] }] },
    ])
  })

  it('un véhicule occupé qui ne bouge pas se tait (décision du 2026-11-02)', () => {
    const immobile = trajet(60, [])
    const plans = planVehicleEngines(
      [track('warthog', [{ t0: 0, t1: 50 }], immobile), track('scorpion', [{ t0: 0, t1: 50 }], immobile)],
      FRAME_MS,
    )
    expect(plans).toEqual([])
  })
})

describe('enginePhaseAt — l état nominal d un épisode', () => {
  const ep = { t0Ms: 1_000, t1Ms: 5_000, arrets: [{ t0Ms: 2_000, t1Ms: 3_000 }] }
  it('course dedans, arrêt sur ses périodes, rien dehors', () => {
    expect(enginePhaseAt(ep, 500)).toBeNull()
    expect(enginePhaseAt(ep, 1_000)).toBe('loop')
    expect(enginePhaseAt(ep, 2_500)).toBe('arret')
    expect(enginePhaseAt(ep, 3_000)).toBe('loop')
    expect(enginePhaseAt(ep, 5_000)).toBeNull()
  })
})

describe('la table des stems et le bus', () => {
  it('les neuf familles de la banque, trois clips chacune, AUCUN boost référencé', () => {
    expect(Object.keys(VEHICLE_ENGINE_STEMS).sort()).toEqual([
      'banshee', 'chopper', 'falcon', 'ghost', 'mongoose', 'scorpion', 'warthog', 'wasp', 'wraith',
    ])
    for (const [famille, stems] of Object.entries(VEHICLE_ENGINE_STEMS)) {
      expect(Object.keys(stems).sort(), famille).toEqual(['enter', 'exit', 'loop'])
      // Le boost est RÉFUTÉ au témoin (mesure du 2026-09-04) : aucune clé ne doit y mener.
      expect(JSON.stringify(stems), famille).not.toContain('boost')
    }
    expect(allEngineStems()).toHaveLength(9 * 3)
  })

  it('le bus moteur est à 0,85 x le maître (décision utilisateur du 2026-09-04)', () => {
    expect(VEHICLE_ENGINE_BUS_GAIN).toBe(0.85)
  })
})

describe('equalPowerCurves — le fondu à puissance constante (piège n° 5 de la banque)', () => {
  it('sin² + cos² = 1 sur toute la jonction, bornes comprises', () => {
    const { fadeIn, fadeOut } = equalPowerCurves(64)
    expect(fadeIn[0]).toBe(0)
    expect(fadeOut[0]).toBe(1)
    expect(fadeIn[63]).toBeCloseTo(1, 10)
    expect(fadeOut[63]).toBeCloseTo(0, 10)
    for (let i = 0; i < 64; i++) {
      // Précision Float32 des courbes (setValueCurveAtTime les exige) : ~1e-7, pas 1e-10.
      expect(fadeIn[i] ** 2 + fadeOut[i] ** 2).toBeCloseTo(1, 6)
    }
  })
})
