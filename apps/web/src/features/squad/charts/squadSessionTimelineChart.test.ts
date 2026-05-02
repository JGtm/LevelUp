/**
 * squadSessionTimelineChart.test.ts — teammates.04.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadSessionTimelineOption } from './squadSessionTimelineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadSessionPoint } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

vi.mock('@/components/charts/_utils', async () => {
  const actual = await vi.importActual<typeof import('@/components/charts/_utils')>(
    '@/components/charts/_utils',
  )
  return { ...actual, seriesColor: (i: number) => `series-color-${i}` }
})

const OPTS = {
  perfLabel: 'Perf',
  winRateLabel: 'WR',
  mmrLabel: 'MMR',
  perfAxisLabel: 'Y1',
  mmrAxisLabel: 'Y2',
}

function makeSeries(rows: SquadSessionPoint[]): ChartSeries<SquadSessionPoint>[] {
  return [{ key: 'k', datapoints: rows }]
}

function point(label: string, perf: number, extras: Partial<SquadSessionPoint> = {}): SquadSessionPoint {
  return { session_label: label, squad_perf: perf, match_count: 5, wins: 3, losses: 2, ...extras }
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadSessionTimelineOption', () => {
  it('points vides → option minimale', () => {
    expect(buildSquadSessionTimelineOption([], OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('1 série bars perf si pas de winrate ni MMR', () => {
    const opt = buildSquadSessionTimelineOption(makeSeries([point('A', 50)]), OPTS)
    const series = opt.series as Array<{ name: string; type: string }>
    expect(series).toHaveLength(1)
    expect(series[0].name).toBe('Perf')
    expect(series[0].type).toBe('bar')
  })

  it('2 séries (bars perf + line winrate) si winrate présent et pas de MMR', () => {
    const opt = buildSquadSessionTimelineOption(
      makeSeries([point('A', 50, { win_rate: 0.6 })]),
      OPTS,
    )
    const series = opt.series as Array<{ name: string }>
    expect(series.map((s) => s.name)).toEqual(['Perf', 'WR'])
    expect((opt.yAxis as unknown as Array<unknown>)).toHaveLength(1)
  })

  it('3 séries (bars perf + line winrate + line MMR) avec yAxis dual quand MMR dispo', () => {
    const opt = buildSquadSessionTimelineOption(
      makeSeries([point('A', 60, { win_rate: 0.6, team_mmr_avg: 1500 })]),
      OPTS,
    )
    const series = opt.series as Array<{ name: string }>
    expect(series.map((s) => s.name)).toEqual(['Perf', 'WR', 'MMR'])
    expect((opt.yAxis as unknown as Array<unknown>)).toHaveLength(2)
  })

  it('couleur perf par tier (≥75 → perf-tier-1, <30 → perf-tier-5)', () => {
    const opt = buildSquadSessionTimelineOption(
      makeSeries([point('A', 80), point('B', 25)]),
      OPTS,
    )
    const series = opt.series as Array<{ data: Array<{ itemStyle: { color: string } }> }>
    expect(series[0].data[0].itemStyle.color).toBe('color:perf-tier-1')
    expect(series[0].data[1].itemStyle.color).toBe('color:perf-tier-5')
  })

  it('winrate × 100 pour affichage en pourcentage', () => {
    const opt = buildSquadSessionTimelineOption(
      makeSeries([point('A', 50, { win_rate: 0.625 })]),
      OPTS,
    )
    const series = opt.series as Array<{ name: string; data: Array<unknown> }>
    const wr = series.find((s) => s.name === 'WR')
    expect(wr?.data[0]).toBe(62.5)
  })
})
