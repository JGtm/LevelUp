/**
 * Tests — placementDropped : LES OBJETS DE PUISSANCE TOMBÉS AU SOL, et tout ce qui n'en est pas.
 *
 * LA RÈGLE TIENT EN DEUX CONDITIONS (origine mesurée `dropped`, famille de puissance) et c'est
 * la SECONDE que ces tests défendent le plus : la décision produit du 2026-08-18 vise les
 * power-ups et les équipements, PAS les grenades — qui font 59 à 63 % des poses d'un film — ni
 * les capacités. Une liste qui déraperait d'une famille noierait la carte, et rien dans le
 * rendu ne le dirait.
 *
 * Ce fichier teste la LISTE et le PRÉDICAT. Le passage à l'écran (la règle de rendu, la forme,
 * le comptage) est testé dans `equipmentPlacementsLayer.test.ts` ; la cohérence de la liste
 * avec la table de rendu et avec le vocabulaire des socles est tenue par
 * `placementDropped.guard.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import {
  DROPPED_EQUIPMENT_FAMILIES,
  PLACEMENT_DROPPED_FAMILIES,
  PLACEMENT_ORIGIN_DROPPED,
  placementIsDroppedPower,
} from './placementDropped'
import { PLACEMENT_ORIGIN_DEPLOYED } from './layers/equipmentPlacementsLayer'
import { OVERSHIELD_ID, pose } from './test/placementFixtures'

/** Une pose LÂCHÉE d'une famille donnée — le cas nominal de ce fichier. */
const lache = (family: string, id = OVERSHIELD_ID) =>
  pose({ family, id, origin: PLACEMENT_ORIGIN_DROPPED })

describe('PLACEMENT_DROPPED_FAMILIES — la liste de ce qui vaut le coup', () => {
  it('porte les six équipements déployables du manifeste', () => {
    expect([...PLACEMENT_DROPPED_FAMILIES].sort()).toEqual(
      [
        'powerup_camo',
        'powerup_overshield',
        'repair_field',
        'sensor',
        'shroud_screen',
        'threat_seeker',
        'translocator_beacon',
        'wall',
      ].sort(),
    )
    expect(DROPPED_EQUIPMENT_FAMILIES).toHaveLength(6)
  })

  it('ne porte NI grenade NI capacité — c’est la moitié de la règle', () => {
    for (const f of [
      'grenade_frag',
      'grenade_plasma',
      'grenade_dynamo',
      'grenade_spike',
      'grapple',
      'thruster',
      'repulsor',
    ]) {
      expect(PLACEMENT_DROPPED_FAMILIES, `${f} ne doit JAMAIS entrer dans la liste`).not.toContain(
        f,
      )
    }
  })

  it('ne porte pas l’objet non identifié : on ne promeut pas ce qu’on ne sait pas nommer', () => {
    expect(PLACEMENT_DROPPED_FAMILIES).not.toContain('other')
  })
})

describe('placementIsDroppedPower — les deux conditions, et aucune n’est déduite', () => {
  it('un power-up lâché en est un — le cas du témoin 01e1f945', () => {
    expect(placementIsDroppedPower(lache('powerup_overshield'))).toBe(true)
    expect(placementIsDroppedPower(lache('powerup_camo'))).toBe(true)
  })

  it('un équipement déployable lâché en est un — mur et capteur du témoin 000d5950', () => {
    for (const f of DROPPED_EQUIPMENT_FAMILIES) {
      expect(placementIsDroppedPower(lache(f)), f).toBe(true)
    }
  })

  it('une grenade lâchée n’en est PAS une, quelle que soit la netteté de la mesure', () => {
    expect(placementIsDroppedPower(lache('grenade_frag'))).toBe(false)
    expect(placementIsDroppedPower(lache('grenade_spike'))).toBe(false)
  })

  it('une capacité lâchée n’en est PAS une (grappin, propulseur, répulseur)', () => {
    for (const f of ['grapple', 'thruster', 'repulsor']) {
      expect(placementIsDroppedPower(lache(f)), f).toBe(false)
    }
  })

  it('le même objet DÉPLOYÉ n’en est pas un : c’est l’origine qui décide, pas la famille', () => {
    for (const f of DROPPED_EQUIPMENT_FAMILIES) {
      const deploye = pose({ family: f, origin: PLACEMENT_ORIGIN_DEPLOYED })
      expect(placementIsDroppedPower(deploye), f).toBe(false)
    }
  })

  it('une origine INCONNUE ou ABSENTE n’est jamais un lâcher — le parc antérieur au schéma 10', () => {
    expect(placementIsDroppedPower(pose({ family: 'sensor', origin: 'unknown' }))).toBe(false)
    expect(placementIsDroppedPower(pose({ family: 'sensor', origin: undefined }))).toBe(false)
  })

  it('une famille que le manifeste n’a pas encore n’en est pas une non plus', () => {
    expect(placementIsDroppedPower(lache('famille_future'))).toBe(false)
  })
})
