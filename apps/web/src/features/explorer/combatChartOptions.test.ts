import { describe, expect, it } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'
import {
  buildCombatFdaOption,
  buildCombatScoreOption,
  type CombatFdaPoint,
  type CombatScorePoint,
} from './combatChartOptions'

/** Forme minimale de l'option ECharts asserttée (évite `any`). */
interface OptShape {
  backgroundColor?: unknown
  yAxis?: Array<{ inverse?: boolean; min?: number }>
  series?: Array<{
    type?: string
    stack?: unknown
    yAxisIndex?: number
    connectNulls?: boolean
    data?: unknown[]
  }>
}

const FDA_LABELS = {
  kills: 'Frags',
  deaths: 'Morts',
  assists: 'Assists',
  fda: 'FDA',
  yAxisLeft: 'Nombre',
  yAxisRight: 'FDA',
}

const SCORE_LABELS = {
  score: 'Score',
  placement: 'Placement',
  yAxisLeft: 'Score',
  yAxisRight: 'Placement',
}

describe('buildCombatFdaOption (G1)', () => {
  it('rend 3 barres groupées (sans stack) + 1 ligne FDA sur l’axe Y secondaire', () => {
    const series: ChartSeries<CombatFdaPoint>[] = [
      {
        key: 'k',
        datapoints: [
          { x: '2026-05-01T10:00:00Z', kills: 10, deaths: 5, assists: 3, kda: 2.1, outcome: 2 },
          { x: '2026-05-02T10:00:00Z', kills: 8, deaths: 8, assists: 4, kda: 1.0, outcome: 3 },
        ],
      },
    ]
    const opt = buildCombatFdaOption(series, FDA_LABELS) as OptShape
    expect(opt.yAxis).toHaveLength(2)
    const bars = (opt.series ?? []).filter((s) => s.type === 'bar')
    const lines = (opt.series ?? []).filter((s) => s.type === 'line')
    expect(bars).toHaveLength(3)
    expect(bars.every((b) => b.stack === undefined)).toBe(true)
    expect(lines).toHaveLength(1)
    expect(lines[0]?.yAxisIndex).toBe(1)
  })

  it('retourne une option vide si aucun datapoint', () => {
    const opt = buildCombatFdaOption([], FDA_LABELS) as OptShape
    expect(opt).toHaveProperty('backgroundColor')
    expect(opt.series).toBeUndefined()
  })
})

describe('buildCombatScoreOption (G3)', () => {
  it('axe placement inversé (min 1) ; rank null → point null + connectNulls false', () => {
    const series: ChartSeries<CombatScorePoint>[] = [
      {
        key: 'k',
        datapoints: [
          { x: '2026-05-01T10:00:00Z', score: 1500, rank: 1 },
          { x: '2026-05-02T10:00:00Z', score: 1200, rank: null },
        ],
      },
    ]
    const opt = buildCombatScoreOption(series, SCORE_LABELS) as OptShape
    expect(opt.yAxis).toHaveLength(2)
    expect(opt.yAxis?.[1]?.inverse).toBe(true)
    expect(opt.yAxis?.[1]?.min).toBe(1)
    const line = (opt.series ?? []).find((s) => s.type === 'line')
    expect(line?.connectNulls).toBe(false)
    expect(line?.data?.[1]).toBeNull()
  })
})
