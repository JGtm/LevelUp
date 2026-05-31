/**
 * Tests buildSessionNetScoreOption — fonction pure (pas de montage ECharts/canvas).
 *
 * Le chart est DIVERGENT via un DÉGRADÉ (vert au-dessus de 0, rouge en dessous), SANS
 * visualMap ni série scatter (tous deux à l'origine du bug "Net score n'affiche rien").
 * Une seule série ligne : data = scalaires (alignés par index), ligne/aire en dégradé,
 * symboles colorés par OUTCOME via itemStyle par point.
 */
import { describe, expect, it } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'

import { buildSessionNetScoreOption } from './SessionNetScoreArea'

type OutcomeKey = 'win' | 'loss' | 'tie' | 'dnf'

interface NetPoint {
  label: string
  cumulative: number
  outcomeKey: OutcomeKey | null
  outcomeLabel: string
}

const SERIES: ChartSeries<NetPoint>[] = [
  {
    key: 'net',
    datapoints: [
      { label: '#1\nBazaar', cumulative: 5, outcomeKey: 'win', outcomeLabel: 'Victoire' },
      { label: '#2\nStreets', cumulative: 2, outcomeKey: 'loss', outcomeLabel: 'Défaite' },
      { label: '#3\nRecharge', cumulative: -3, outcomeKey: 'loss', outcomeLabel: 'Défaite' },
    ],
  },
]

type GradientColor = { type?: string; colorStops?: unknown[] }
type DataItem = { value: number; itemStyle?: { color?: string } }
type OptShape = {
  series: Array<{
    data: unknown[]
    type: string
    lineStyle?: { color?: GradientColor | string }
    areaStyle?: { color?: GradientColor | string; origin?: number }
  }>
  visualMap?: unknown
  xAxis: { data: string[]; boundaryGap?: boolean }
}

describe('buildSessionNetScoreOption', () => {
  it('série vide → option de fond minimale', () => {
    const opt = buildSessionNetScoreOption([], { seriesLabel: 'Net' }) as { backgroundColor: string }
    expect(opt.backgroundColor).toBeTruthy()
    expect((opt as unknown as { series?: unknown }).series).toBeUndefined()
  })

  it('une SEULE série ligne dont les data exposent les scalaires (alignés par index)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net cumulé' }) as unknown as OptShape
    expect(opt.series).toHaveLength(1)
    expect(opt.series[0].type).toBe('line')
    expect((opt.series[0].data as DataItem[]).map((d) => d.value)).toEqual([5, 2, -3])
  })

  it('PAS de visualMap (source des rendus vides)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.visualMap).toBeUndefined()
  })

  it('ligne + aire en DÉGRADÉ divergent (colorStops), aire ancrée à 0', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    const lineColor = opt.series[0].lineStyle?.color as GradientColor
    const areaColor = opt.series[0].areaStyle?.color as GradientColor
    expect(lineColor?.type).toBe('linear')
    expect(lineColor?.colorStops).toHaveLength(4)
    expect(areaColor?.type).toBe('linear')
    expect(opt.series[0].areaStyle?.origin).toBe(0)
  })

  it('chaque point porte un itemStyle (couleur OUTCOME du symbole)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    for (const item of opt.series[0].data as DataItem[]) {
      expect(item).toHaveProperty('itemStyle')
    }
  })

  it('axe X = labels #N\\nCarte fournis, sans boundaryGap (points sur les vertices)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.xAxis.data).toEqual(['#1\nBazaar', '#2\nStreets', '#3\nRecharge'])
    expect(opt.xAxis.boundaryGap).toBe(false)
  })
})
