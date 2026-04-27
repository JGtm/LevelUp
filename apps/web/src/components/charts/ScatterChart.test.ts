/**
 * Tests — buildScatterOption (Phase 3 P3.B).
 */
import { describe, it, expect } from 'vitest'
import { buildScatterOption, type ChartPointScatter } from './ScatterChart'
import type { ChartSeries } from './ChartCard'

function makeSeries(
  key: string,
  name: string,
  points: ChartPointScatter[],
): ChartSeries<ChartPointScatter> {
  return {
    key,
    meta: { gamertag: name },
    datapoints: points,
  }
}

interface OptionShape {
  series?: Array<{
    type?: string
    name?: string
    data?: Array<[number, number]>
    symbolSize?: number
  }>
  xAxis?: { type?: string; name?: string }
  yAxis?: { type?: string; name?: string }
  legend?: { data?: string[] }
  backgroundColor?: string
}

describe('buildScatterOption', () => {
  it('retourne option vide quand aucune série', () => {
    const opt = buildScatterOption([]) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('génère une série scatter par ChartSeries', () => {
    const opt = buildScatterOption([
      makeSeries('win', 'Victoires', [
        { x: 5, y: 2 },
        { x: 8, y: 1 },
      ]),
      makeSeries('loss', 'Défaites', [{ x: 3, y: 6 }]),
    ]) as OptionShape
    expect(opt.series).toHaveLength(2)
    expect(opt.series?.[0].type).toBe('scatter')
  })

  it('mappe les points {x,y} en tuples [x,y]', () => {
    const opt = buildScatterOption([
      makeSeries('s', 'S', [
        { x: 1, y: 2 },
        { x: 3, y: 4 },
      ]),
    ]) as OptionShape
    expect(opt.series?.[0].data).toEqual([
      [1, 2],
      [3, 4],
    ])
  })

  it('utilise meta.gamertag comme nom de série', () => {
    const opt = buildScatterOption([makeSeries('s', 'Mon-GT', [{ x: 0, y: 0 }])]) as OptionShape
    expect(opt.series?.[0].name).toBe('Mon-GT')
    expect(opt.legend?.data).toEqual(['Mon-GT'])
  })

  it('respecte un seriesNameResolver custom', () => {
    const opt = buildScatterOption(
      [makeSeries('outcome.win', 'Win', [{ x: 0, y: 0 }])],
      { seriesNameResolver: (s) => s.key.toUpperCase() },
    ) as OptionShape
    expect(opt.series?.[0].name).toBe('OUTCOME.WIN')
  })

  it('expose xAxisLabel + yAxisLabel sur les axes', () => {
    const opt = buildScatterOption(
      [makeSeries('s', 'S', [{ x: 0, y: 0 }])],
      { xAxisLabel: 'Kills', yAxisLabel: 'K/D' },
    ) as OptionShape
    expect(opt.xAxis?.name).toBe('Kills')
    expect(opt.yAxis?.name).toBe('K/D')
  })

  it('respecte symbolSize custom', () => {
    const opt = buildScatterOption(
      [makeSeries('s', 'S', [{ x: 0, y: 0 }])],
      { symbolSize: 10 },
    ) as OptionShape
    expect(opt.series?.[0].symbolSize).toBe(10)
  })

  it('symbolSize default = 5', () => {
    const opt = buildScatterOption([makeSeries('s', 'S', [{ x: 0, y: 0 }])]) as OptionShape
    expect(opt.series?.[0].symbolSize).toBe(5)
  })

  it('axes X et Y sont type value (numérique)', () => {
    const opt = buildScatterOption([makeSeries('s', 'S', [{ x: 0, y: 0 }])]) as OptionShape
    expect(opt.xAxis?.type).toBe('value')
    expect(opt.yAxis?.type).toBe('value')
  })
})
