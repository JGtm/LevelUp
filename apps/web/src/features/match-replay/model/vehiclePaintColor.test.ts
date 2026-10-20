/**
 * vehiclePaintColor.test.ts — la RÈGLE d'encre d'un véhicule selon sa nature publiée, sans canvas.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleRideReady, ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { vehicleIsMapElement, vehiclePaintColor } from './vehiclesLayer'

function track(over: Partial<ReplayVehicleTrackReady> = {}): ReplayVehicleTrackReady {
  return {
    slot: 700, gen: 1, t0: 0, t1: 1000, t1max: 1000, end: 'inconnue', family: 'warthog',
    samples: [], rides: [], ...over,
  }
}

function ride(over: Partial<ReplayVehicleRideReady> = {}): ReplayVehicleRideReady {
  return { t0: 0, t1: 100, slot: 1, src: 'event', aim: [], ...over }
}

describe('vehiclePaintColor — quelle encre pour quelle nature', () => {
  const ink = { colorOfSlot: () => '#equipe', colorOfXuid: () => null }
  const inks = { neutral: '#neutre', mapElement: '#gris' }
  const occupe = () => track({ rides: [ride({ slot: 1, seat: 0 })] })

  it('un élément de carte est TOUJOURS gris, occupant ou non', () => {
    expect(vehiclePaintColor(track({ rides: [] }), 50, 'map_element', ink, inks)).toBe('#gris')
    expect(vehiclePaintColor(occupe(), 50, 'map_element', ink, inks)).toBe('#gris')
  })

  it('la tourelle fixe et le véhicule gardent la couleur de l’occupant, puis le neutre', () => {
    for (const kind of ['fixed_turret', undefined]) {
      expect(vehiclePaintColor(occupe(), 50, kind, ink, inks)).toBe('#equipe')
      expect(vehiclePaintColor(track({ rides: [] }), 50, kind, ink, inks)).toBe('#neutre')
    }
  })

  it('seule la nature map_element est un élément de carte', () => {
    expect(vehicleIsMapElement('map_element')).toBe(true)
    expect(vehicleIsMapElement('fixed_turret')).toBe(false)
    expect(vehicleIsMapElement(undefined)).toBe(false)
  })
})
