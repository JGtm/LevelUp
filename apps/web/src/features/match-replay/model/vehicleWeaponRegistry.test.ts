/**
 * vehicleWeaponRegistry.test.ts — LE LECTEUR DU REGISTRE DES ARMES DE VÉHICULE (schéma 69, lot M4a
 * des retours du rejeu 2026-09-23). Le style, le son et le montage d'un tir d'arme de véhicule
 * viennent du DOCUMENT (`vehicleWeapons`, keyé par `Shot.w`), plus d'une table du client.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayVehicleWeapon } from '@/lib/api/types'

import { shotSoundStem } from '../sound/replaySound'
import { testReplayDoc } from '../test/testDoc'
import { buildShotFx } from './shotFx'
import {
  vehicleShotSoundStem,
  vehicleShotStyleOf,
  vehicleWeaponMountOf,
  vehicleWeaponOf,
} from './vehicleWeaponRegistry'

const ROQUETTES = '0xC7D5091200000000'
const GRENADES = '0x0BB6976B00000000'
const INCONNU = '0x850902EF00000000'
const ARME_DE_JOUEUR = '0x84BD29ED42C9679F'

const roquettes: ReplayVehicleWeapon = {
  vehicle: 'rockethog', en: 'Rocket Launcher', fr: 'Lance-roquettes', fire: 'single',
  fx: 'explosive', tint: 'blast', sound: 'vehicle_shot_warthog_rocket_1',
  mount: { aim: 'turret', ax: 0, ay: 0.26 },
}
const grenades: ReplayVehicleWeapon = {
  vehicle: 'falcon', en: 'Grenade Launcher', fr: 'Lance-grenades', fire: 'single',
  fx: 'explosive', tint: 'kinetic', mount: { aim: 'turret', ax: 0.06, ay: 0.13 },
}

function docAvecRegistre() {
  return testReplayDoc({
    vehicleWeapons: { [ROQUETTES]: roquettes, [GRENADES]: grenades },
    vehicles: [700, 701, 702].map((slot) => ({
      slot, gen: 0, family: 'warthog', t0: 0, t1: 10, t1max: 10, end: 'film_end',
    })),
    shots: [
      { t: 1, slot: 10, x: 0, y: 0, w: ROQUETTES, v: 700 },
      { t: 2, slot: 11, x: 0, y: 0, w: GRENADES, v: 701 },
      { t: 3, slot: 12, x: 0, y: 0, w: INCONNU, v: 702 },
    ],
  })
}

describe('vehicleWeaponRegistry — le registre du document fait foi', () => {
  it('une arme du registre rend son style, son son et son montage', () => {
    const doc = docAvecRegistre()
    expect(vehicleShotStyleOf(doc, ROQUETTES)).toEqual({ fx: 'explosive', tint: 'blast' })
    expect(vehicleShotSoundStem(doc, ROQUETTES)).toBe('vehicle_shot_warthog_rocket_1')
    expect(vehicleWeaponMountOf(doc, ROQUETTES)).toEqual({ classe: 'tourelle', ax: 0, ay: 0.26 })
  })

  it('un silence DÉCIDÉ garde le style et le montage, et ne sonne pas', () => {
    const doc = docAvecRegistre()
    expect(vehicleShotSoundStem(doc, GRENADES)).toBeUndefined()
    expect(vehicleShotStyleOf(doc, GRENADES)?.fx).toBe('explosive')
    expect(vehicleWeaponMountOf(doc, GRENADES)?.classe).toBe('tourelle')
  })

  it('une arme absente du registre n emprunte rien : rendu neutre, silence, centre du véhicule', () => {
    const doc = docAvecRegistre()
    for (const w of [INCONNU, ARME_DE_JOUEUR, undefined, '']) {
      expect(vehicleWeaponOf(doc, w)).toBeNull()
      expect(vehicleShotStyleOf(doc, w)).toBeNull()
      expect(vehicleShotSoundStem(doc, w)).toBeUndefined()
      expect(vehicleWeaponMountOf(doc, w)).toBeNull()
    }
  })

  it('un document sans registre (artefact antérieur au schéma 69) ne style ni ne fait sonner rien', () => {
    const doc = testReplayDoc({ shots: [{ t: 1, slot: 10, x: 0, y: 0, w: ROQUETTES, v: 700 }] })
    expect(vehicleShotStyleOf(doc, ROQUETTES)).toBeNull()
    expect(shotSoundStem(doc, { w: ROQUETTES })).toBeUndefined()
  })

  it('une classe de visée inconnue ne fabrique pas de montage', () => {
    const doc = testReplayDoc({
      vehicleWeapons: { [ROQUETTES]: { ...roquettes, mount: { aim: 'free', ax: 0, ay: 0 } } },
    })
    expect(vehicleWeaponMountOf(doc, ROQUETTES)).toBeNull()
  })

  it('la chaîne du rendu lit le registre : buildShotFx et shotSoundStem', () => {
    const doc = docAvecRegistre()
    const fx = buildShotFx(doc, 10)
    expect(fx[0]).toMatchObject({ fam: 'explosive', tint: 'blast' })
    expect(fx[0].vehicleShot?.mount).toEqual({ classe: 'tourelle', ax: 0, ay: 0.26 })
    expect(fx[2].vehicleShot?.mount ?? null).toBeNull()
    expect(shotSoundStem(doc, { w: ROQUETTES })).toBe('vehicle_shot_warthog_rocket_1')
    expect(shotSoundStem(doc, { w: GRENADES })).toBeUndefined()
  })
})
