/**
 * Tests du PLAN moteur des véhicules (vehicleEngineSound.ts) : fusion des épisodes
 * d'occupation, ralenti du Scorpion, refus du décor, table des stems, et le fondu croisé à
 * puissance constante — les règles du cadrage du 2026-09-04 et du contrat de la banque.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleTrackReady } from './replayNormalize'
import {
  ENGINE_RIDE_GAP_MERGE_MS,
  enginePhaseAt,
  equalPowerCurves,
  idleSpansOf,
  mergeRideSpans,
  planVehicleEngines,
  SCORPION_IDLE_MIN_MS,
  VEHICLE_ENGINE_BUS_GAIN,
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

describe('idleSpansOf — le ralenti du Scorpion', () => {
  // Un véhicule immobile 3 s (frames 10-40), puis en mouvement.
  const samples = [
    ...Array.from({ length: 41 }, (_, i) => ({ t: i, x: i < 10 ? i * 2 : 20, y: 0 })),
    ...Array.from({ length: 20 }, (_, i) => ({ t: 41 + i, x: 20 + (i + 1) * 2, y: 0 })),
  ]

  it('détecte la période immobile, bornée à l épisode', () => {
    const spans = idleSpansOf(samples, { t0Ms: 0, t1Ms: 6_000 }, FRAME_MS)
    expect(spans).toHaveLength(1)
    expect(spans[0].t0Ms).toBeGreaterThanOrEqual(900)
    expect(spans[0].t1Ms).toBe(4_000)
  })

  it('absorbe un arrêt plus court que l hystérésis', () => {
    const court = [
      { t: 0, x: 0, y: 0 },
      { t: 5, x: 10, y: 0 },
      { t: 9, x: 10, y: 0 }, // 0,4 s d arrêt < SCORPION_IDLE_MIN_MS
      { t: 14, x: 20, y: 0 },
    ]
    expect(idleSpansOf(court, { t0Ms: 0, t1Ms: 2_000 }, FRAME_MS)).toEqual([])
    expect(SCORPION_IDLE_MIN_MS).toBe(1_000)
  })
})

describe('planVehicleEngines — quoi sonne, quoi se tait', () => {
  it('refuse le décor (falcon), la famille inconnue, et le véhicule jamais occupé', () => {
    const plans = planVehicleEngines(
      [
        track('falcon', [{ t0: 0, t1: 100 }]), // décor : jamais de son (cadrage n° 1)
        track('shade_turret', [{ t0: 0, t1: 100 }]), // sans banque : silence propre (n° 2)
        track('warthog', []), // jamais occupé : pas de moteur
        track('ghost', [{ t0: 10, t1: 50 }]),
      ],
      FRAME_MS,
    )
    expect(plans).toEqual([
      { family: 'ghost', episodes: [{ t0Ms: 1_000, t1Ms: 5_000, idle: [] }] },
    ])
  })

  it('ne calcule le ralenti que pour les familles qui ont un clip idle (Scorpion)', () => {
    const still = Array.from({ length: 60 }, (_, i) => ({ t: i, x: 0, y: 0 }))
    const plans = planVehicleEngines(
      [track('scorpion', [{ t0: 0, t1: 50 }], still), track('warthog', [{ t0: 0, t1: 50 }], still)],
      FRAME_MS,
    )
    expect(plans[0].episodes[0].idle.length).toBeGreaterThan(0)
    expect(plans[1].episodes[0].idle).toEqual([])
  })
})

describe('enginePhaseAt — l état nominal d un épisode', () => {
  const ep = { t0Ms: 1_000, t1Ms: 5_000, idle: [{ t0Ms: 2_000, t1Ms: 3_000 }] }
  it('course dedans, ralenti sur ses périodes, rien dehors', () => {
    expect(enginePhaseAt(ep, 500)).toBeNull()
    expect(enginePhaseAt(ep, 1_000)).toBe('loop')
    expect(enginePhaseAt(ep, 2_500)).toBe('idle')
    expect(enginePhaseAt(ep, 3_000)).toBe('loop')
    expect(enginePhaseAt(ep, 5_000)).toBeNull()
  })
})

describe('la table des stems et le bus', () => {
  it('les neuf familles de la banque, l idle du Scorpion seul, AUCUN boost référencé', () => {
    expect(Object.keys(VEHICLE_ENGINE_STEMS).sort()).toEqual([
      'banshee', 'chopper', 'falcon', 'ghost', 'mongoose', 'scorpion', 'warthog', 'wasp', 'wraith',
    ])
    for (const [famille, stems] of Object.entries(VEHICLE_ENGINE_STEMS)) {
      expect(stems.idle !== undefined, famille).toBe(famille === 'scorpion')
      // Le boost est RÉFUTÉ au témoin (mesure du 2026-09-04) : aucune clé ne doit y mener.
      expect(JSON.stringify(stems), famille).not.toContain('boost')
    }
    expect(allEngineStems()).toHaveLength(9 * 3 + 1)
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
