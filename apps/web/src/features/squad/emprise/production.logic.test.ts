/**
 * production.logic.test.ts — « Frags obtenus avec les ressources » et « Rendement face à
 * l'adversaire » sur la soirée témoin du 22/09 (maquette de l'onglet, L4.6), puis le chemin
 * sans film (Halo 5, D10).
 */
import { describe, expect, it } from 'vitest'

import type { SquadEmpriseBlock } from '@/lib/api/types'

import { EMPRISE_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildProductionRows, buildYieldRows, yieldGeometry } from './production.logic'

const FR = EMPRISE_TEXT.fr

describe('buildProductionRows', () => {
  it('22/09 : frags pendant l’effet 8 / 5 sur 2 min 39 / 1 min 53 ; frags aux armes spéciales 47 / 54 sur 23 / 29 prises', () => {
    const rows = buildProductionRows(EMPRISE_2209)
    expect(rows.map((r) => r.resource)).toEqual(['powerup', 'power_weapon'])
    const [bonus, armes] = rows
    expect(bonus.kills).toEqual({ us: 8, them: 5 })
    expect(bonus.exposure?.kind).toBe('effect_ms')
    const fx = FR.production.exposure.effect_ms
    expect([fx.fmt(bonus.exposure!.value.us), fx.fmt(bonus.exposure!.value.them)]).toEqual(['2 min 39', '1 min 53'])
    expect(FR.pctFmt((159 / 272) * 100)).toBe('58,5 %')
    expect(armes.kills).toEqual({ us: 47, them: 54 })
    expect(armes.exposure).toEqual({ kind: 'pickups', value: { us: 23, them: 29 } })
    expect(FR.production.exposure.pickups.fmt(23)).toBe('23 prises')
  })

  it('une ressource sans frag n’a pas de ligne ; une exposition nulle n’a pas de barre fine', () => {
    const block: SquadEmpriseBlock = {
      ...EMPRISE_2209,
      production: [
        { resource: 'powerup', kills: { us: 0, them: 0 }, exposure: { kind: 'effect_ms', value: { us: 5000, them: 0 }, kills: { us: 0, them: 0 } } },
        { resource: 'power_weapon', kills: { us: 3, them: 1 }, exposure: { kind: 'pickups', value: { us: 0, them: 0 }, kills: { us: 0, them: 0 } } },
      ],
    }
    const rows = buildProductionRows(block)
    expect(rows.map((r) => r.resource)).toEqual(['power_weapon'])
    expect(rows[0].exposure).toBeNull()
  })
})

describe('buildYieldRows', () => {
  it('22/09 : bonus 3,0 contre 2,7 (+14 %), armes spéciales 1,7 contre 1,4 (+17 %, périmètre des prises mesurées)', () => {
    const [bonus, armes] = buildYieldRows(EMPRISE_2209)
    expect(FR.yield.rawFmt(bonus.us, bonus.them)).toBe('3,0 contre 2,7')
    expect(FR.yield.gapFmt(bonus.gap)).toBe('+14 %')
    expect(FR.yield.rawFmt(armes.us, armes.them)).toBe('1,7 contre 1,4')
    expect(FR.yield.gapFmt(armes.gap)).toBe('+17 %')
    expect(EMPRISE_TEXT.en.yield.gapFmt(-0.08)).toBe('−8%')
    expect(FR.yield.gapFmt(0.001)).toBe('0 %')
  })

  it('sans rendement publié (pas d’exposition), aucune ligne', () => {
    const block: SquadEmpriseBlock = { ...EMPRISE_2209, production: [{ resource: 'power_weapon', kills: { us: 9, them: 13 } }] }
    expect(buildYieldRows(block)).toEqual([])
  })
})

describe('yieldGeometry', () => {
  it('le zéro au milieu, ±50 % aux bords, au-delà borné', () => {
    expect(yieldGeometry(0.25)).toEqual({ left: 50, width: 25, x: 75, clamped: false })
    expect(yieldGeometry(-0.5)).toEqual({ left: 0, width: 50, x: 0, clamped: false })
    expect(yieldGeometry(1.2)).toMatchObject({ left: 50, width: 50, x: 100, clamped: true })
  })
})
