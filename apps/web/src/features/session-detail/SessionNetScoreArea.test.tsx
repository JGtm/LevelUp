/**
 * Tests buildSessionNetScoreOption — fonction pure (pas de montage ECharts/canvas).
 *
 * Régression ciblée : la série DOIT exposer des paires [index, cumul] (2D) et non des
 * scalaires, sinon `visualMap.dimension: 1` pointe hors-portée et la courbe devient
 * invisible (le bug "Net score n'affiche rien"). On vérifie aussi la couleur de repli.
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
  series: Array<{ data: unknown[]; lineStyle?: { color?: string }; areaStyle?: { color?: string } }>
  visualMap: { dimension: number; pieces: unknown[] }
  xAxis: { data: string[] }
}

describe('buildSessionNetScoreOption', () => {
  it('série vide → option de fond minimale', () => {
    const opt = buildSessionNetScoreOption([], { seriesLabel: 'Net' }) as { backgroundColor: string }
    expect(opt.backgroundColor).toBeTruthy()
    expect((opt as unknown as { series?: unknown }).series).toBeUndefined()
  })

  it('expose des paires [index, cumul] (2D) — pas des scalaires', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net cumulé' }) as unknown as OptShape
    const data = opt.series[0].data
    expect(data).toHaveLength(3)
    // Chaque point est une paire [i, cumul].
    expect(data[0]).toEqual([0, 5])
    expect(data[1]).toEqual([1, 2])
    expect(data[2]).toEqual([2, -3])
  })

  it('visualMap colore selon le cumul (dimension 1)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.visualMap.dimension).toBe(1)
    expect(opt.visualMap.pieces).toHaveLength(2)
  })

  it('couleur de repli explicite sur la ligne + l’aire (visibilité garantie)', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    // La clé `color` est posée (valeur résolue via token en prod ; vide en jsdom sans
    // palette). On vérifie la présence structurelle de la couleur de repli, pas sa valeur.
    expect(opt.series[0].lineStyle).toHaveProperty('color')
    expect(opt.series[0].areaStyle).toHaveProperty('color')
  })

  it('axe X = labels #N\\nCarte fournis', () => {
    const opt = buildSessionNetScoreOption(SERIES, { seriesLabel: 'Net' }) as unknown as OptShape
    expect(opt.xAxis.data).toEqual(['#1\nBazaar', '#2\nStreets', '#3\nRecharge'])
  })
})
