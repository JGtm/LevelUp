/**
 * Tests — buildHistogramOption (Phase 3 P3.B).
 */
import { describe, it, expect } from 'vitest'
import { buildHistogramOption, type ChartPointHistogram } from './HistogramChart'
import type { ChartSeries } from './ChartCard'

function makeSeries(points: ChartPointHistogram[]): ChartSeries<ChartPointHistogram>[] {
  return [
    {
      key: 'test.histogram',
      meta: { gamertag: 'test' },
      datapoints: points,
    },
  ]
}

interface OptionShape {
  series?: Array<{ type?: string; data?: number[]; itemStyle?: { color?: string } }>
  xAxis?: { type?: string; data?: string[]; name?: string }
  yAxis?: { type?: string; name?: string }
  backgroundColor?: string
}

describe('buildHistogramOption', () => {
  it('retourne option vide si aucune série', () => {
    const opt = buildHistogramOption([]) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('retourne option vide si série sans datapoints', () => {
    const opt = buildHistogramOption(makeSeries([])) as OptionShape
    expect(opt.series).toBeUndefined()
  })

  it('génère une série bar avec les counts dans l\'ordre', () => {
    const opt = buildHistogramOption(
      makeSeries([
        { binStart: 0, binEnd: 1, count: 3 },
        { binStart: 1, binEnd: 2, count: 7 },
        { binStart: 2, binEnd: 3, count: 5 },
      ]),
    ) as OptionShape
    expect(opt.series?.[0].type).toBe('bar')
    expect(opt.series?.[0].data).toEqual([3, 7, 5])
  })

  it('mappe binStart/binEnd en categories format integer', () => {
    const opt = buildHistogramOption(
      makeSeries([
        { binStart: 0, binEnd: 5, count: 1 },
        { binStart: 5, binEnd: 10, count: 2 },
      ]),
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['0–5', '5–10'])
  })

  it('formate les bornes float avec 2 décimales', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0.5, binEnd: 1.25, count: 1 }]),
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['0.50–1.25'])
  })

  it('respecte un formatBin custom', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
      { formatBin: (p) => `bucket-${p.count}` },
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['bucket-1'])
  })

  it('expose xAxisLabel + yAxisLabel sur les axes', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
      { xAxisLabel: 'K/D', yAxisLabel: 'Matchs' },
    ) as OptionShape
    expect(opt.xAxis?.name).toBe('K/D')
    expect(opt.yAxis?.name).toBe('Matchs')
  })

  it('par défaut yAxisLabel = "Matchs"', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
    ) as OptionShape
    expect(opt.yAxis?.name).toBe('Matchs')
  })
})
