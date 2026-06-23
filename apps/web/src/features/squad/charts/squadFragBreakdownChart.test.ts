/**
 * squadFragBreakdownChart.test.ts — « Répartition des frags » par joueur (barres empilées).
 */
import { describe, it, expect, vi } from 'vitest'
import { buildFragBreakdownOption, type FragBreakdownLabels } from './squadFragBreakdownChart'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `hex(${token})`,
}))

function pt(order: number, overrides: Partial<SquadPerformanceSeriesPoint> = {}): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    ...overrides,
  }
}

const LABELS: FragBreakdownLabels = {
  melee: 'Mêlée',
  powerWeapon: 'Arme lourde',
  grenade: 'Grenade',
  other: 'Autres',
}
const ORDER = ['Me', 'F1']

type Serie = { name: string; type: string; stack: string; itemStyle: { color: string }; data: number[] }

describe('buildFragBreakdownOption', () => {
  it('vide → option minimale (aucune série)', () => {
    const opt = buildFragBreakdownOption({}, { labels: LABELS })
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
    expect(opt.series).toBeUndefined()
  })

  it('agrège les types par joueur sur tous les matchs + dérive « Autres »', () => {
    const rows = {
      // melee 1+2=3, pw 0+1=1, gren 1+0=1, kills 10+10=20 → autres = 20-5 = 15
      Me: [
        pt(0, { kills: 10, melee_kills: 1, power_weapon_kills: 0, grenade_kills: 1 }),
        pt(1, { kills: 10, melee_kills: 2, power_weapon_kills: 1, grenade_kills: 0 }),
      ],
      // melee 0, pw 2, gren 0, kills 8 → autres = 8-2 = 6
      F1: [pt(0, { kills: 8, melee_kills: 0, power_weapon_kills: 2, grenade_kills: 0 })],
    }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ORDER, labels: LABELS })
    const series = opt.series as Serie[]
    // 4 segments mutuellement exclusifs, dans l'ordre melee/pw/grenade/other.
    expect(series).toHaveLength(4)
    expect(series.map((s) => s.name)).toEqual(['Mêlée', 'Arme lourde', 'Grenade', 'Autres'])
    expect(series.every((s) => s.type === 'bar' && s.stack === 'frags')).toBe(true)
    // data alignée sur l'ordre des joueurs [Me, F1].
    expect(series[0].data).toEqual([3, 0]) // mêlée
    expect(series[1].data).toEqual([1, 2]) // arme lourde
    expect(series[2].data).toEqual([1, 0]) // grenade
    expect(series[3].data).toEqual([15, 6]) // autres = kills − typés
  })

  it('clampe « Autres » à 0 si les types dépassent les kills', () => {
    const rows = { Me: [pt(0, { kills: 3, melee_kills: 2, power_weapon_kills: 2, grenade_kills: 0 })] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ['Me'], labels: LABELS })
    const series = opt.series as Serie[]
    expect(series[3].data).toEqual([0]) // max(0, 3 − 4)
  })

  it('champs kill-type absents → traités comme 0, « Autres » absorbe les frags', () => {
    const rows = { Me: [pt(0, { kills: 7 })] } // pas de melee/pw/grenade
    const opt = buildFragBreakdownOption(rows, { playerOrder: ['Me'], labels: LABELS })
    const series = opt.series as Serie[]
    expect(series[0].data).toEqual([0])
    expect(series[1].data).toEqual([0])
    expect(series[2].data).toEqual([0])
    expect(series[3].data).toEqual([7]) // tout en « Autres »
  })

  it('couleurs PAR TYPE via tokens chart-series 1/6/7/8 (alignées sur le donut)', () => {
    const rows = { Me: [pt(0, { melee_kills: 1 })] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ['Me'], labels: LABELS })
    const series = opt.series as Serie[]
    expect(series[0].itemStyle.color).toBe('hex(chart-series-1)')
    expect(series[1].itemStyle.color).toBe('hex(chart-series-6)')
    expect(series[2].itemStyle.color).toBe('hex(chart-series-7)')
    expect(series[3].itemStyle.color).toBe('hex(chart-series-8)')
  })

  it('axe Y = joueurs dans l’ordre, inversé (main en haut)', () => {
    const rows = { Me: [pt(0)], F1: [pt(0)] }
    const opt = buildFragBreakdownOption(rows, { playerOrder: ORDER, labels: LABELS })
    const yAxis = opt.yAxis as { type: string; data: string[]; inverse: boolean }
    expect(yAxis.type).toBe('category')
    expect(yAxis.data).toEqual(['Me', 'F1'])
    expect(yAxis.inverse).toBe(true)
  })
})
