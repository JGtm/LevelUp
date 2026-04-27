/**
 * Tests — buildKdaBarsOption (Phase 3 P3.G).
 */
import { describe, it, expect } from 'vitest'
import { buildKdaBarsOption } from './TimeseriesKdaBars'
import type { ChartSeries } from '@/components/charts/ChartCard'

interface KdaBarPoint {
  x: string
  kills: number
  deaths: number
  kdRatio: number
  outcome: number | null
}

const labels = {
  killsLabel: 'Kills',
  deathsLabel: 'Deaths',
  kdRatioLabel: 'K/D',
  yAxisLeft: 'Kills / Deaths',
  yAxisRight: 'K/D',
}

function makeSeries(points: KdaBarPoint[]): ChartSeries<KdaBarPoint>[] {
  return [
    {
      key: 'timeseries.kda_bars',
      meta: { gamertag: 'kda' },
      datapoints: points,
    },
  ]
}

interface OptionShape {
  series?: Array<{
    type?: string
    name?: string
    stack?: string
    data?: unknown[]
    yAxisIndex?: number
  }>
  yAxis?: Array<{ type?: string; position?: string; min?: number }>
  xAxis?: { type?: string; data?: string[] }
  legend?: { data?: string[] }
  backgroundColor?: string
}

describe('buildKdaBarsOption', () => {
  it('retourne option vide quand aucune série', () => {
    const opt = buildKdaBarsOption([], labels) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('retourne option vide si série sans datapoints', () => {
    const opt = buildKdaBarsOption(makeSeries([]), labels) as OptionShape
    expect(opt.series).toBeUndefined()
  })

  it('génère 3 séries (kills bar + deaths bar + K/D line) dans l\'ordre', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
        { x: '2025-01-02', kills: 4, deaths: 8, kdRatio: 0.5, outcome: 3 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.series).toHaveLength(3)
    expect(opt.series?.[0].name).toBe('Kills')
    expect(opt.series?.[0].type).toBe('bar')
    expect(opt.series?.[1].name).toBe('Deaths')
    expect(opt.series?.[1].type).toBe('bar')
    expect(opt.series?.[2].name).toBe('K/D')
    expect(opt.series?.[2].type).toBe('line')
  })

  it('deaths sont rendus en valeurs négatives (axe Y inversé)', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.series?.[1].data).toEqual([-5])
  })

  it('K/D line utilise yAxisIndex 1 (axe droit)', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.series?.[2].yAxisIndex).toBe(1)
    expect(opt.series?.[2].data).toEqual([2.0])
  })

  it('yAxis dual : gauche kills/deaths + droite K/D positionnée right', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.yAxis).toHaveLength(2)
    expect(opt.yAxis?.[0].position).toBeUndefined() // default left
    expect(opt.yAxis?.[1].position).toBe('right')
    expect(opt.yAxis?.[1].min).toBe(0)
  })

  it('xAxis = category avec timestamps comme data', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
        { x: '2025-01-02', kills: 4, deaths: 8, kdRatio: 0.5, outcome: 3 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.xAxis?.type).toBe('category')
    expect(opt.xAxis?.data).toEqual(['2025-01-01', '2025-01-02'])
  })

  it('kills + deaths empilés sur stack "kda"', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.series?.[0].stack).toBe('kda')
    expect(opt.series?.[1].stack).toBe('kda')
  })

  it('legend liste les 3 noms de séries', () => {
    const opt = buildKdaBarsOption(
      makeSeries([
        { x: '2025-01-01', kills: 10, deaths: 5, kdRatio: 2.0, outcome: 2 },
      ]),
      labels,
    ) as OptionShape
    expect(opt.legend?.data).toEqual(['Kills', 'Deaths', 'K/D'])
  })
})
