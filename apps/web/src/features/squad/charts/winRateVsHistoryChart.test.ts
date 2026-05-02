/**
 * winRateVsHistoryChart.test.ts — builder option ECharts win rate vs historique.
 *
 * tokenCssVar est stubbed pour retourner le nom du token comme valeur de couleur —
 * permet d'asserter le token sans CSS var réelle.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildWinRateVsHistoryOption } from './winRateVsHistoryChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MapBreakdownRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

function row(mapUI: string, winRate: number, historicalWinRate?: number): MapBreakdownRow {
  return { map_ui: mapUI, match_count: 10, win_rate: winRate, historical_win_rate: historicalWinRate }
}

const OPTS = {
  mapLabelOf: (m: string) => m.toUpperCase(),
  sessionLabel: 'Session',
  historyLabel: 'Historique',
}

function makeSeries(rows: MapBreakdownRow[]): ChartSeries<MapBreakdownRow>[] {
  return [{ key: 'win-rate-vs-history', datapoints: rows }]
}

describe('buildWinRateVsHistoryOption', () => {
  beforeEach(() => { vi.clearAllMocks() })

  it('série vide → option minimale', () => {
    const opt = buildWinRateVsHistoryOption([], OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('série sans datapoints → option minimale', () => {
    const opt = buildWinRateVsHistoryOption([{ key: 'k', datapoints: [] }], OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('applique mapLabelOf sur les noms de cartes', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('aquarius', 0.5, 0.5)]), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toContain('AQUARIUS')
  })

  it('trie par win_rate session asc', () => {
    const rows = [row('b', 0.8, 0.5), row('a', 0.3, 0.5), row('c', 0.6, 0.5)]
    const opt = buildWinRateVsHistoryOption(makeSeries(rows), OPTS)
    const yAxis = opt.yAxis as { data: string[] }
    expect(yAxis.data).toEqual(['A', 'C', 'B'])
  })

  it('session au-dessus du seuil → couleur divergent-pos', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('x', 0.8, 0.5)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle?: { color: string } }> }>
    const sessionSeries = series.find((s) => s.data[0]?.itemStyle !== undefined)
    expect(sessionSeries?.data[0].itemStyle?.color).toBe('color:divergent-pos')
  })

  it('session en dessous du seuil → couleur divergent-neg', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('x', 0.3, 0.6)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle?: { color: string } }> }>
    const sessionSeries = series.find((s) => s.data[0]?.itemStyle !== undefined)
    expect(sessionSeries?.data[0].itemStyle?.color).toBe('color:divergent-neg')
  })

  it('session dans le seuil → couleur divergent-neutral', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('x', 0.52, 0.5)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle?: { color: string } }> }>
    const sessionSeries = series.find((s) => s.data[0]?.itemStyle !== undefined)
    expect(sessionSeries?.data[0].itemStyle?.color).toBe('color:divergent-neutral')
  })

  it('sans historique → couleur divergent-neutral', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('x', 0.7)]), OPTS)
    const series = opt.series as Array<{ data: Array<{ itemStyle?: { color: string } }> }>
    const sessionSeries = series.find((s) => s.data[0]?.itemStyle !== undefined)
    expect(sessionSeries?.data[0].itemStyle?.color).toBe('color:divergent-neutral')
  })

  it('valeurs converties en pourcentage (×100, 1 décimale)', () => {
    const opt = buildWinRateVsHistoryOption(makeSeries([row('x', 0.625, 0.5)]), OPTS)
    const series = opt.series as Array<{ data: unknown[] }>
    const sessionData = series.find((s) => {
      const d = s.data[0]
      return typeof d === 'object' && d !== null && 'value' in d
    })?.data[0] as { value: number }
    expect(sessionData.value).toBe(62.5)
  })
})
