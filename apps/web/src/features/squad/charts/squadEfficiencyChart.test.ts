/**
 * squadEfficiencyChart.test.ts — builder multi-joueurs paramétré par métrique.
 *
 * `buildSquadEfficiencyMultiOption` : 1 courbe/joueur, couleur par joueur,
 * valeurs = dégâts/frag (damagePerKill) ou dégâts/mort (damagePerDeath) selon la
 * métrique, repère « 1 vie » (série fantôme hors légende), bornes d'axe incluant
 * le repère, robustesse au joueur sans donnée.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { buildSquadEfficiencyMultiOption } from './squadEfficiencyChart'

function pt(
  order: number,
  fields: Partial<SquadPerformanceSeriesPoint>,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 0,
    deaths: 0,
    assists: 0,
    ...fields,
  } as SquadPerformanceSeriesPoint
}

interface LineSeries {
  name: string
  type: string
  data: Array<number | null>
  lineStyle?: { color: string }
  markLine?: { data: Array<{ yAxis: number }> }
  silent?: boolean
}

const COLORS = { Me: '#aaa', F1: '#bbb' }
const PLAYERS = ['Me', 'F1']
const ONE_LIFE = 225
const REF = '1 vie (225)'

function baseOpts(metric: 'damagePerKill' | 'damagePerDeath') {
  return { metric, refLabel: REF, oneLife: ONE_LIFE, colorByPlayer: COLORS, showXAxis: true } as const
}

describe('buildSquadEfficiencyMultiOption', () => {
  it('vide (aucune donnée) → fond minimal, pas de série', () => {
    const opt = buildSquadEfficiencyMultiOption({}, PLAYERS, baseOpts('damagePerKill'))
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('metric damagePerKill : 1 courbe/joueur = dégâts infligés / frags, arrondi', () => {
    const rows = {
      Me: [
        pt(0, { damage_dealt: 900, kills: 4 }), // 225
        pt(1, { damage_dealt: 1000, kills: 5 }), // 200
      ],
      F1: [pt(0, { damage_dealt: 600, kills: 3 })], // 200
    }
    const opt = buildSquadEfficiencyMultiOption(rows, PLAYERS, baseOpts('damagePerKill'))
    const series = opt.series as unknown as LineSeries[]
    // 2 joueurs + 1 série fantôme repère
    expect(series).toHaveLength(3)
    expect(series[0].name).toBe('Me')
    expect(series[0].data).toEqual([225, 200])
    expect(series[1].name).toBe('F1')
    expect(series[1].data).toEqual([200, null])
    expect(series[0].lineStyle?.color).toBe('#aaa')
    expect(series[1].lineStyle?.color).toBe('#bbb')
  })

  it('metric damagePerDeath : 1 courbe/joueur = dégâts subis / morts, arrondi', () => {
    const rows = {
      Me: [
        pt(0, { damage_taken: 1000, deaths: 5 }), // 200
        pt(1, { damage_taken: 900, deaths: 3 }), // 300
      ],
      F1: [pt(0, { damage_taken: 800, deaths: 4 })], // 200
    }
    const opt = buildSquadEfficiencyMultiOption(rows, PLAYERS, baseOpts('damagePerDeath'))
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].data).toEqual([200, 300])
    expect(series[1].data).toEqual([200, null])
  })

  it('repère « 1 vie » : série fantôme hors légende, markLine à oneLife', () => {
    const rows = { Me: [pt(0, { damage_dealt: 900, kills: 4 })] }
    const opt = buildSquadEfficiencyMultiOption(rows, PLAYERS, baseOpts('damagePerKill'))
    const series = opt.series as unknown as LineSeries[]
    const ghost = series.at(-1) as LineSeries
    expect(ghost.name).toBe(REF)
    expect(ghost.data).toEqual([])
    expect(ghost.silent).toBe(true)
    expect(ghost.markLine?.data[0].yAxis).toBe(ONE_LIFE)
    // La légende ne liste que les joueurs (le repère reste toujours visible).
    const legend = opt.legend as unknown as { data: string[] }
    expect(legend.data).toEqual(PLAYERS)
  })

  it('bornes d\'axe Y : le repère « 1 vie » reste dans [min,max]', () => {
    const rows = {
      Me: [pt(0, { damage_dealt: 800, kills: 4 })], // 200 (sous 225)
      F1: [pt(0, { damage_dealt: 600, kills: 3 })], // 200
    }
    const opt = buildSquadEfficiencyMultiOption(rows, PLAYERS, baseOpts('damagePerKill'))
    const yAxis = opt.yAxis as unknown as { min: number; max: number }
    expect(yAxis.min).toBeLessThanOrEqual(ONE_LIFE)
    expect(yAxis.max).toBeGreaterThanOrEqual(ONE_LIFE)
  })

  it('joueur sans donnée → série présente, entièrement null', () => {
    const rows = {
      Me: [pt(0, { damage_dealt: 900, kills: 4 })], // 225
      F1: [] as SquadPerformanceSeriesPoint[],
    }
    const opt = buildSquadEfficiencyMultiOption(rows, PLAYERS, baseOpts('damagePerKill'))
    const series = opt.series as unknown as LineSeries[]
    expect(series[0].data).toEqual([225])
    expect(series[1].name).toBe('F1')
    expect(series[1].data).toEqual([null])
  })
})
