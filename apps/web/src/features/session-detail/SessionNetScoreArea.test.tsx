/**
 * Tests buildSessionNetScoreOption — fonction pure (pas de montage ECharts/canvas).
 *
 * Régression du bug "Net score n'affiche rien" : la série DOIT exposer des SCALAIRES
 * (données 1D, ECharts aligne par index sur l'axe catégoriel) + une couleur de ligne
 * EXPLICITE (pas de visualMap dont la défaillance rendait la courbe invisible).
 */
import { describe, expect, it } from 'vitest'

import type { ChartSeries } from '@/components/charts/ChartCard'

import { buildSessionNetScoreOption } from './SessionNetScoreArea'

interface NetPoint {
  label: string
  cumulative: number
}

const SERIES: ChartSeries<NetPoint>[] = [
  {
    key: 'net',
    datapoints: [
      { label: '#1\nBazaar', cumulative: 5 },
      { label: '#2\nStreets', cumulative: 2 },
      { label: '#3\nRecharge', cumulative: -3 },
    ],
  },
]

type OptShape = {
  series: Array<{ data: unknown[]; type: string; lineStyle?: { color?: string }; areaStyle?: { color?: string } }>
  visualMap?: unknown
  xAxis: { data: string[] }
}

describe('buildSessionNetScoreOption', () => {
  it('série vide → option de fond minimale', () => {
    const opt = buildSessionNetScoreOption([], { seriesLabel: 'Net' }) as { backgroundColor: string }
    expect(opt.backgroundColor).toBeTruthy()
    expect((opt as unknown as { series?: unknown }).series).toBeUndefined()
  })

  it('expose des SCALAIRES (données 1D) — pas de paires', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net cumulé' }) as unknown as OptShape
    // 1D : un scalaire par catégorie (ECharts aligne par index).
    expect(opt.series[0].data).toEqual([5, 2, -3])
    expect(opt.series[0].type).toBe('line')
  })

  it('pas de visualMap (source des rendus vides) ; couleur de ligne + aire explicite', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.visualMap).toBeUndefined()
    // La clé `color` est posée (valeur résolue via token en prod ; vide en jsdom).
    expect(opt.series[0].lineStyle).toHaveProperty('color')
    expect(opt.series[0].areaStyle).toHaveProperty('color')
  })

  it('axe X = labels #N\\nCarte fournis', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.xAxis.data).toEqual(['#1\nBazaar', '#2\nStreets', '#3\nRecharge'])
  })
})
