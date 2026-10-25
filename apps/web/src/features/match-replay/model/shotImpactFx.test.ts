/// <reference types="node" />
// @vitest-environment node
/**
 * shotImpactFx.test.ts — l'explosion d'un projectile de tir : elle se pose pour une arme qui
 * DÉCLARE `impact`, à la détonation LUE, et jamais d'après la forme de rendu `fx`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { featureRoot } from '../test/featureFiles'
import { testReplayDoc } from '../test/testDoc'
import { IMPACT_SOUND_STEMS } from '../sound/impactSound'
import { grenadeSoundEvents } from '../sound/grenadeSound'
import { explosionSeedOf } from './grenadeFx'
import { buildRestFx, buildShotImpactFx } from './shotImpactFx'

const MORTIER = '0x121B400900000000'
/** weapon_key du titre pour l'arme à pied du témoin. */
const ARME_SPNKR = 'hinf_m41_spnkr'

function doc(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    shots: [
      { slot: 1, t: 10, x: 0, y: 0, w: '0xSPNKR' },
      { slot: 2, t: 10, x: 0, y: 0, w: '0xPLASMA' },
      { slot: 3, t: 20, x: 0, y: 0, w: MORTIER },
      { slot: 4, t: 30, x: 0, y: 0, w: '0xBALLE' },
    ],
    projectiles: [
      { t0: 10, p: [[0, 0, 0], [5, 4, 0]], detoT: 15, detoX: 4, detoY: 0, detoLue: true, shot: 0 },
      { t0: 10, p: [[0, 0, 0], [5, 4, 1]], detoT: 15, detoX: 4, detoY: 1, detoLue: true, shot: 1 },
      { t0: 20, p: [[0, 0, 0], [9, 8, 0]], detoT: 29, detoX: 8, detoY: 0, detoLue: true, shot: 2 },
      { t0: 30, p: [[0, 0, 0], [2, 1, 0]], detoT: 32, detoX: 1, detoY: 0, detoLue: true, shot: 3 },
    ],
    weaponLabels: {
      '0xSPNKR': { en: 'SPNKr', fr: 'SPNKr', fx: 'explosive', tint: 'blast', key: ARME_SPNKR, impact: true },
      // Même forme `plasma` que le mortier, mais PAS d'explosion : la forme ne décide de rien.
      '0xPLASMA': { en: 'Pistolet', fr: 'Pistolet', fx: 'plasma', tint: 'plasma_cool' },
      // `explosive` SANS `impact` : pas d'explosion non plus.
      '0xBALLE': { en: 'Arme', fr: 'Arme', fx: 'explosive' },
    },
    vehicleWeapons: {
      [MORTIER]: {
        vehicle: 'wraith', en: 'Plasma Mortar', fr: 'Mortier à plasma', fire: 'single',
        fx: 'plasma', tint: 'plasma_hot', impact: true, impactSound: 'impact_wraith_mortar',
      },
    },
    ...over,
  })
}

describe('buildShotImpactFx', () => {
  it('explose l’arme à pied et l’arme de véhicule qui déclarent impact, à la détonation publiée', () => {
    const fx = buildShotImpactFx(doc())
    expect(fx).toHaveLength(2)
    expect(fx[0]).toMatchObject({ frame: 15, x: 4, y: 0, tint: 'blast', rank: -1 })
    expect(fx[0]?.impactStem).toBeUndefined()
    expect(fx[1]).toMatchObject({ frame: 29, x: 8, y: 0, tint: 'plasma_hot', impactStem: 'impact_wraith_mortar' })
  })

  it('le germe suit la même formule que les grenades', () => {
    expect(buildShotImpactFx(doc())[0]?.seed).toBe(explosionSeedOf(15))
  })

  it('une arme sans impact n’explose pas, quelle que soit sa forme de rendu (plasma ou explosive)', () => {
    const fx = buildShotImpactFx(doc())
    expect(fx.map((e) => e.frame)).not.toContain(32)
    expect(fx.some((e) => e.x === 4 && e.y === 1)).toBe(false)
  })

  it('un projectile sans lien de tir (artefact ancien) ne pose rien et ne plante pas', () => {
    const d = doc({
      projectiles: [{ t0: 10, p: [[0, 0, 0], [5, 4, 0]], detoT: 15, detoX: 4, detoY: 0, detoLue: true }],
    })
    expect(buildShotImpactFx(d)).toEqual([])
  })

  it('un lien hors bornes est ignoré', () => {
    const d = doc({
      projectiles: [{ t0: 10, p: [[0, 0, 0], [5, 4, 0]], detoT: 15, detoX: 4, detoY: 0, detoLue: true, shot: 99 }],
    })
    expect(buildShotImpactFx(d)).toEqual([])
  })

  it('une détonation non lue (repli sur la dernière position) n’est pas une explosion', () => {
    const d = doc({
      projectiles: [{ t0: 10, p: [[0, 0, 0], [5, 4, 0]], detoT: 15, detoX: 4, detoY: 0, detoLue: false, shot: 0 }],
    })
    expect(buildShotImpactFx(d)).toEqual([])
  })

  it('une teinte inconnue ou absente retombe sur la déflagration', () => {
    const d = doc({
      weaponLabels: { '0xSPNKR': { en: 'S', fr: 'S', fx: 'explosive', impact: true } },
    })
    expect(buildShotImpactFx(d)[0]?.tint).toBe('blast')
  })

  it('buildRestFx place les explosions de tir après celles des grenades', () => {
    const d = doc({
      grenades: [{ t: 5, slot: 0, i: 1, x: 0, y: 0, rank: 0, s: 'projectile', proj: 3 }],
    })
    const all = buildRestFx(d)
    expect(all[0]?.rank).toBe(0)
    expect(all.slice(1).every((e) => e.tint !== undefined)).toBe(true)
  })
})

describe('son d’impact', () => {
  it('le mortier sonne par le son de son registre, l’arme à pied sans son déclaré reste muette', () => {
    expect(Object.keys(IMPACT_SOUND_STEMS)).toEqual([])
    const ev = grenadeSoundEvents(doc(), [])
    expect(ev.map((e) => e.stem)).toEqual(['impact_wraith_mortar'])
  })
})

describe('garde-rail : la forme de rendu ne décide pas d’une explosion', () => {
  it('shotImpactFx.ts ne lit ni `fx` ni la valeur explosive dans son code', () => {
    const src = readFileSync(resolve(featureRoot(), 'model', 'shotImpactFx.ts'), 'utf8')
      .replace(/\/\*[\s\S]*?\*\//g, '')
      .replace(/^\s*\/\/.*$/gm, '')
    expect(src).not.toMatch(/\.fx\b/)
    expect(src).not.toMatch(/['"]explosive['"]/)
  })
})
