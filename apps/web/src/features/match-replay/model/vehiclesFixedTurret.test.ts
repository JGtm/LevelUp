/**
 * vehiclesFixedTurret.test.ts — LA TOURELLE FIXE (retours du rejeu, lot M6.2, 2026-09-24).
 *
 * Les tourelles gatling / mortier de Takamanohara (châssis `0x3a8060e2`, information de
 * l'utilisateur) sont posées par la carte ET occupées par un joueur. Le titre les qualifie
 * `fixed_turret` : même pictogramme qu'un élément de carte faute d'asset, mais le pion de son
 * occupant EMBARQUE — l'inverse exact de `map_element`, qui n'embarque jamais personne.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import {
  buildEmbarkedPredicate,
  FAMILLES_ARME_FIXE,
  vehicleCanEmbark,
  vehicleMapElementGlyph,
  VEHICLE_KIND_FIXED_TURRET,
  VEHICLE_KIND_MAP_ELEMENT,
} from './vehiclesLayer'

/** Une vie de tourelle fixe, occupée par le slot 7 de la frame 0 à 100. */
function tourelle(): ReplayVehicleTrackReady {
  return {
    slot: 768,
    gen: 1,
    t0: 0,
    t1: 1000,
    t1max: 1000,
    end: 'film_end',
    family: 'tourelle_fixe',
    samples: [],
    rides: [{ t0: 0, t1: 100, slot: 7, src: 'event', aim: [] }],
  }
}

describe('tourelle fixe (lot M6.2) — dessinée par son pictogramme, et occupée', () => {
  it('se dessine par le pictogramme de tourelle quand le document publie sa nature', () => {
    expect(vehicleMapElementGlyph('tourelle_fixe', VEHICLE_KIND_FIXED_TURRET)).toBe('turret')
    // Sans nature publiée (document antérieur), aucune forme n'est affirmée.
    expect(vehicleMapElementGlyph('tourelle_fixe', undefined)).toBeNull()
  })

  it('EMBARQUE son occupant, à l’inverse d’un élément de carte', () => {
    expect(vehicleCanEmbark(tourelle(), VEHICLE_KIND_FIXED_TURRET)).toBe(true)
    expect(vehicleCanEmbark(tourelle(), VEHICLE_KIND_MAP_ELEMENT)).toBe(false)
    const embarque = buildEmbarkedPredicate([tourelle()], () => VEHICLE_KIND_FIXED_TURRET)
    expect(embarque(7, 50)).toBe(true)
    expect(embarque(7, 150)).toBe(false)
  })

  it('son arme est solidaire de son corps : le cap suit la visée de l’occupant', () => {
    expect(FAMILLES_ARME_FIXE.has('tourelle_fixe')).toBe(true)
  })
})
