/**
 * vehiclesLayer.heading.test.ts — LE CAP TOURNE ENTRE DEUX ÉCHANTILLONS (contrôles v8, 2026-10-21,
 * témoin 7b0d89c4 : « rotations saccadées des véhicules »). Séparé de vehiclesLayer.test.ts, qui
 * est au plafond de taille du dépôt.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleSample } from '@/lib/api/types'

import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleHeadingAt } from './vehiclesLayer'

function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'inconnue',
    family: 'warthog',
    samples: [],
    rides: [],
    ...over,
  }
}

function sample(over: Partial<ReplayVehicleSample>): ReplayVehicleSample {
  return { t: 0, x: 0, y: 0, ...over }
}

describe('vehicleHeadingAt — le cap TOURNE entre deux échantillons (contrôles v8, rotations saccadées)', () => {
  const deux = (h0: number, h1: number, over: Partial<ReplayVehicleSample> = {}) =>
    track({
      samples: [
        sample({ t: 100, x: 0, y: 0, h: h0 }),
        sample({ t: 101, x: 1, y: 0, h: h1, ...over }),
      ],
    })

  it('à mi-chemin de deux échantillons, le cap est à mi-chemin', () => {
    expect(vehicleHeadingAt(deux(10, 50), 100.5)).toBeCloseTo(30, 6)
    expect(vehicleHeadingAt(deux(10, 50), 100.25)).toBeCloseTo(20, 6)
  })

  it('PAR LE PLUS COURT CHEMIN : de 350 à 10 degrés, on passe par 0, pas par 180', () => {
    expect(vehicleHeadingAt(deux(350, 10), 100.5)).toBeCloseTo(0, 6)
    expect(vehicleHeadingAt(deux(10, 350), 100.5)).toBeCloseTo(0, 6)
    expect(vehicleHeadingAt(deux(350, 10), 100.75)).toBeCloseTo(5, 6)
  })

  it('aux échantillons eux-mêmes, le cap est exactement le leur', () => {
    expect(vehicleHeadingAt(deux(10, 50), 100)).toBe(10)
    expect(vehicleHeadingAt(deux(10, 50), 101)).toBe(50)
  })

  it('JAMAIS AU TRAVERS D UN SILENCE de réplication (g > 0) : le cap est tenu', () => {
    expect(vehicleHeadingAt(deux(10, 50, { g: 800 }), 100.5)).toBe(10)
  })

  it('un trou de plus d une seconde ne se comble pas : le cap reste tenu', () => {
    const t = track({
      samples: [sample({ t: 0, x: 0, y: 0, h: 10 }), sample({ t: 200, x: 1, y: 0, h: 90 })],
    })
    expect(vehicleHeadingAt(t, 100)).toBe(10)
  })

  it('un échantillon sans cap entre les deux ne casse pas la rotation', () => {
    const t = track({
      samples: [
        sample({ t: 100, x: 0, y: 0, h: 0 }),
        sample({ t: 102, x: 1, y: 0 }),
        sample({ t: 104, x: 2, y: 0, h: 40 }),
      ],
    })
    // 100 -> 104 : le cap tourne de 0 à 40 degrés, l'échantillon sans cap ne l'interrompt pas.
    expect(vehicleHeadingAt(t, 102)).toBeCloseTo(20, 6)
    expect(vehicleHeadingAt(t, 103)).toBeCloseTo(30, 6)
  })
})
