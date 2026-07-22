/**
 * squadWeaponAccuracyBarsChart.test.ts — « Précision par rôle » Escouade : barres groupées
 * horizontales, 1 barre/joueur/rôle, longueur = précision % (axe borné 0..100).
 */
import { describe, it, expect } from 'vitest'
import { buildSquadWeaponAccuracyBarsOption } from './squadWeaponAccuracyBarsChart'
import type { SquadWeaponAccuracy, SquadWeaponAccuracyBar } from '@/lib/api/types'

const opts = {
  colorByPlayer: { Me: '#111111', F1: '#222222' },
  roleLabel: (r: string) => `R:${r}`,
  shotsLabel: 'Tirs',
}

function bar(
  role: string,
  acc: Record<string, number>,
  shots: Record<string, number>,
): SquadWeaponAccuracyBar {
  return {
    role,
    accuracy_by_player: acc,
    shots_fired_by_player: shots,
    total_shots_squad: Object.values(shots).reduce((a, b) => a + b, 0),
  }
}

type Serie = { name: string; type: string; data: (number | null)[]; itemStyle: { color: string } }

describe('buildSquadWeaponAccuracyBarsOption', () => {
  it('vide / null → option minimale (aucune série)', () => {
    expect(buildSquadWeaponAccuracyBarsOption(null, opts)).toMatchObject({ backgroundColor: 'transparent' })
    expect(buildSquadWeaponAccuracyBarsOption({ players: [], bars: [] }, opts).series).toBeUndefined()
  })

  it('1 série par joueur, précision 0..1 → %, joueur sans tir sur le rôle = null', () => {
    const data: SquadWeaponAccuracy = {
      players: ['Me', 'F1'],
      bars: [
        bar('precision', { Me: 0.4, F1: 0.55 }, { Me: 100, F1: 40 }),
        bar('sniper', { Me: 0.7 }, { Me: 10 }), // F1 n’a pas tiré → absent
      ],
    }
    const opt = buildSquadWeaponAccuracyBarsOption(data, opts)
    const series = opt.series as Serie[]
    expect(series.map((s) => s.name)).toEqual(['Me', 'F1'])
    expect(series[0].data).toEqual([40, 70]) // Me : 0.4→40 %, 0.7→70 %
    expect(series[1].data).toEqual([55, null]) // F1 : 0.55→55 %, absent→null
    expect(series[0].itemStyle.color).toBe('#111111')
    expect(series[1].itemStyle.color).toBe('#222222')
  })

  it('axe Y = libellés de rôle (roleLabel), un rôle vide est ignoré', () => {
    const data: SquadWeaponAccuracy = {
      players: ['Me'],
      bars: [
        bar('precision', { Me: 0.5 }, { Me: 20 }),
        bar('', { Me: 0.9 }, { Me: 5 }), // rôle vide → ignoré
      ],
    }
    const opt = buildSquadWeaponAccuracyBarsOption(data, opts)
    expect((opt.yAxis as { data: string[] }).data).toEqual(['R:precision'])
    expect((opt.series as Serie[])[0].data).toEqual([50])
  })

  it('xAxis borné 0..100 (longueur = précision honnête)', () => {
    const data: SquadWeaponAccuracy = { players: ['Me'], bars: [bar('precision', { Me: 0.4 }, { Me: 10 })] }
    const xAxis = buildSquadWeaponAccuracyBarsOption(data, opts).xAxis as { min: number; max: number }
    expect(xAxis.min).toBe(0)
    expect(xAxis.max).toBe(100)
  })
})
