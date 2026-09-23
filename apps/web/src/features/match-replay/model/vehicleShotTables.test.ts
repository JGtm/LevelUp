/**
 * vehicleShotTables.test.ts — LES ARMES DE VÉHICULE DES RETOURS DU 2026-09-23 (lot L1.5).
 *
 * Décisions utilisateur appliquées (plan des retours du rejeu, §3.0 et §5) :
 *  - Q6 : Ghost et canons de la Banshee en plasma ROUGE (`plasma_hot`, la teinte du Ravageur) ;
 *  - Q7 : mortier du Wraith (`121B4009`, 142 tirs observés) ROUGE ;
 *  - Q8 : canons du Chopper ROUGES, forme plasma ;
 *  - Q9 : le Rockethog (`C7D50912`, 127 tirs, porté par sa tourelle enfant `bcfb852f` de famille
 *    vide) SONNE de nouveau — le tag est celui du lance-roquettes (`WARTHOG_FINAL_2026-09-02.md`
 *    §1 : `vehi bcfb852f -> weap c7d50912`, banque `veh_un_rockethog`), il n'y a rien à départager ;
 *  - le canon du Scorpion est publié sous `49E40D17` (13 tirs), pas sous `00015cfa` (jamais vu) ;
 *  - Q10 : `0BB6976B` (51 tirs, tourelle enfant `1a043c29`) = LANCE-GRENADES du Falcon : explosion
 *    à l'impact, teinte cinétique, poste de porte ; aucune reconstruction sonore = silence décidé.
 */
import { describe, expect, it } from 'vitest'

import { shotSoundStem } from '../sound/replaySound'
import { testReplayDoc } from '../test/testDoc'
import { vehicleShotStyleOf } from './vehicleShotFx'
import { vehicleWeaponMountOf, vehicleWeapTag } from './vehicleWeaponMounts'

const TAG = {
  ghost: vehicleWeapTag('00015435'),
  bansheeM1: vehicleWeapTag('0000aa68'),
  wraith: vehicleWeapTag('121b4009'),
  chopper: vehicleWeapTag('b40e9618'),
  rockethog: vehicleWeapTag('c7d50912'),
  scorpion: vehicleWeapTag('49e40d17'),
  falconGL: vehicleWeapTag('0bb6976b'),
}

describe('style — le plasma Banished est ROUGE (Q6, Q7, Q8)', () => {
  for (const nom of ['ghost', 'bansheeM1', 'wraith', 'chopper'] as const) {
    it(`${nom} : forme plasma, teinte plasma_hot`, () => {
      expect(vehicleShotStyleOf(TAG[nom])).toEqual({ fx: 'plasma', tint: 'plasma_hot' })
    })
  }
})

describe('Scorpion — le tag PUBLIÉ 49E40D17', () => {
  it('a un style d’obus (explosion à l’impact)', () => {
    expect(vehicleShotStyleOf(TAG.scorpion)?.fx).toBe('explosive')
  })
  it('sonne le canon du Scorpion', () => {
    const doc = testReplayDoc({ shots: [{ slot: 1, t: 0, x: 0, y: 0, w: TAG.scorpion, v: 700 }] })
    expect(shotSoundStem(doc, doc.shots[0])).toBe('vehicle_shot_scorpion_1')
  })
  it('part du plateau de tourelle', () => {
    expect(vehicleWeaponMountOf(TAG.scorpion)?.classe).toBe('tourelle')
  })
})

describe('Falcon — le lance-grenades 0BB6976B (Q10)', () => {
  it('explosion à l’impact, teinte cinétique', () => {
    expect(vehicleShotStyleOf(TAG.falconGL)).toEqual({ fx: 'explosive', tint: 'kinetic' })
  })
  it('part du poste de porte (tourelle)', () => {
    expect(vehicleWeaponMountOf(TAG.falconGL)?.classe).toBe('tourelle')
  })
  it('silence DÉCIDÉ : aucune reconstruction du lance-grenades', () => {
    const doc = testReplayDoc({ shots: [{ slot: 1, t: 0, x: 0, y: 0, w: TAG.falconGL, v: 700 }] })
    expect(shotSoundStem(doc, doc.shots[0])).toBeUndefined()
  })
})

describe('Rockethog — de nouveau audible (Q9)', () => {
  /** Le tir tel que le parc le publie : sur la tourelle ENFANT `bcfb852f`, sans famille. */
  function docTourelleEnfant(family: string | undefined) {
    return testReplayDoc({
      frameIntervalMs: 100,
      vehicles: [{ slot: 701, gen: 1, t0: 0, t1: 50, t1max: 50, end: 'unknown', family, samples: [], rides: [] }],
      shots: [{ slot: 1, t: 10, x: 0, y: 0, w: TAG.rockethog, v: 701 }],
    })
  }

  it('tir porté par la tourelle enfant (famille vide) : la roquette sonne', () => {
    const doc = docTourelleEnfant(undefined)
    expect(shotSoundStem(doc, doc.shots[0])).toBe('vehicle_shot_warthog_rocket_1')
  })

  it('tir porté par le châssis Warthog lui-même : la roquette sonne aussi', () => {
    const doc = docTourelleEnfant('warthog')
    expect(shotSoundStem(doc, doc.shots[0])).toBe('vehicle_shot_warthog_rocket_1')
  })

  it('sans véhicule publié : le tag suffit', () => {
    const doc = testReplayDoc({ shots: [{ slot: 1, t: 0, x: 0, y: 0, w: TAG.rockethog, v: 700 }] })
    expect(shotSoundStem(doc, doc.shots[0])).toBe('vehicle_shot_warthog_rocket_1')
  })
})
