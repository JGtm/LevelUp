/**
 * vehiclesAim.test.ts — LA VISÉE D'UN OCCUPANT DE VÉHICULE (schéma 31), sans canvas.
 *
 * CE QUE CES TESTS PROTÈGENT. Une régression ici ne se verrait PAS à l'écran : le cône
 * retomberait sur le cap du châssis, c'est-à-dire sur une direction PLAUSIBLE mais fausse de 15,7
 * à 21,8 deg en médiane (q3 39,6-52,9 deg, lot V11). C'est exactement le genre de défaut qu'un
 * visionnage ne rattrape pas — d'où le `measured` du résultat, sur lequel les cas s'accrochent.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleAim } from '@/lib/api/types'

import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from './replayNormalize'
import { VEHICLE_AIM_HOLD_FRAMES, vehicleOccupantAimAt, vehicleRideAimReading } from './vehiclesAim'
import { vehicleAimAngle } from './vehiclesLayer'

/** Un véhicule dont le châssis pointe PLEIN EST (cap monde 0°) : le repli est alors l'angle 0. */
function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'inconnue',
    family: 'warthog',
    samples: [{ t: 0, x: 0, y: 0, h: 0 }],
    rides: [],
    ...over,
  }
}

function ride(aim: ReplayVehicleAim[], over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady {
  return { t0: 0, t1: 1000, slot: 7, src: 'event', aim, ...over }
}

describe('vehicleRideAimReading — la lecture EN VIGUEUR', () => {
  it('rend la dernière lecture au plus tard à l’image demandée (aucune interpolation)', () => {
    const r = ride([
      { t: 10, h: 10 },
      { t: 14, h: 20 },
      { t: 30, h: 30 },
    ])
    expect(vehicleRideAimReading(r, 14)?.h).toBe(20)
    // ENTRE DEUX LECTURES : la précédente est MAINTENUE, jamais moyennée — interpoler deux caps
    // ferait tourner le cône par le chemin le plus court à travers 0/360 deg.
    expect(vehicleRideAimReading(r, 16)?.h).toBe(20)
  })

  it('rend null AVANT la première lecture, et sur une série vide', () => {
    expect(vehicleRideAimReading(ride([{ t: 10, h: 10 }]), 9)).toBeNull()
    expect(vehicleRideAimReading(ride([]), 50)).toBeNull()
  })

  it('rend null au-delà du MAINTIEN — une visée d’il y a plus d’une seconde n’est plus la sienne', () => {
    const r = ride([{ t: 10, h: 10 }])
    expect(vehicleRideAimReading(r, 10 + VEHICLE_AIM_HOLD_FRAMES)?.h).toBe(10)
    expect(vehicleRideAimReading(r, 10 + VEHICLE_AIM_HOLD_FRAMES + 1)).toBeNull()
  })

  it('SAUTE un point sans cap plutôt que de le lire comme 0° (qui pointerait l’est)', () => {
    const r = ride([{ t: 10, h: 45 }, { t: 12 }])
    expect(vehicleRideAimReading(r, 12)?.h).toBe(45)
  })
})

describe('vehicleOccupantAimAt — la mesure d’abord, le châssis en repli', () => {
  it('la visée MESURÉE l’emporte sur le cap du châssis, et elle porte son élévation', () => {
    const got = vehicleOccupantAimAt(track(), ride([{ t: 50, h: 90, p: -12.5 }]), 50)
    expect(got.measured).toBe(true)
    expect(got.ang).toBeCloseTo(vehicleAimAngle(90), 10)
    expect(got.pitchDeg).toBe(-12.5)
  })

  it('ARTILLEUR ET PASSAGER ont chacun LEUR angle — c’est tout l’objet du lot', () => {
    const t = track()
    const artilleur = vehicleOccupantAimAt(t, ride([{ t: 50, h: 200 }], { slot: 8, seat: 1 }), 50)
    const passager = vehicleOccupantAimAt(t, ride([{ t: 50, h: 300 }], { slot: 9, seat: 2 }), 50)
    expect(artilleur.ang).toBeCloseTo(vehicleAimAngle(200), 10)
    expect(passager.ang).toBeCloseTo(vehicleAimAngle(300), 10)
    expect(artilleur.ang).not.toBeCloseTo(passager.ang, 3)
  })

  it('SANS visée, le repli est le CAP DU CHÂSSIS, à plat — le comportement du schéma 30', () => {
    const got = vehicleOccupantAimAt(track(), ride([]), 50)
    expect(got.measured).toBe(false)
    expect(got.ang).toBeCloseTo(vehicleAimAngle(0), 10) // châssis plein est
    expect(got.pitchDeg).toBe(0)
  })

  it('une élévation ABSENTE se lit « à plat », jamais « inconnue » (contrat de `Point.p`)', () => {
    expect(vehicleOccupantAimAt(track(), ride([{ t: 50, h: 90 }]), 50).pitchDeg).toBe(0)
  })
})
