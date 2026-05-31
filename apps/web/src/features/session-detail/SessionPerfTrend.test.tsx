/**
 * Tests buildSessionPerfOption — option ECharts du graphe "Score de performance" :
 * barres par match (itemStyle par tier) + markLine de moyenne + axe X catégoriel.
 * (Les couleurs résolues ne sont pas testées : resolveToken renvoie '' hors runtime CSS.)
 */
import { describe, expect, it } from 'vitest'

import { buildSessionPerfOption } from './SessionPerfTrend'

interface PerfOptShape {
  series?: Array<{
    type: string
    data: Array<{ value: number; itemStyle: { color: string } }>
    markLine: { data: Array<{ yAxis: number }>; label: { formatter: string } }
  }>
  xAxis?: { type: string; data: unknown[] }
  yAxis?: { min?: number; max?: number }
}

function makeSeries(points: Array<{ score: number; tier: number }>) {
  return [
    {
      key: 'perf',
      datapoints: points.map((p, i) => ({ label: `#${i + 1}\nMap`, score: p.score, tier: p.tier })),
    },
  ]
}

describe('buildSessionPerfOption', () => {
  it('produit des barres par match + une markLine de moyenne en couleur grading', () => {
    const opt = buildSessionPerfOption(
      makeSeries([
        { score: 80, tier: 1 },
        { score: 40, tier: 4 },
        { score: 60, tier: 3 },
      ]),
      { scoreLabel: 'Score', meanLabel: 'Moyenne' },
    ) as unknown as PerfOptShape

    const bar = opt.series![0]
    expect(bar.type).toBe('bar')
    expect(bar.data).toHaveLength(3)
    // Chaque barre porte un itemStyle (couleur résolue par tier).
    expect(bar.data.every((d) => 'color' in d.itemStyle)).toBe(true)
    // Moyenne = (80 + 40 + 60) / 3 = 60.
    expect(bar.markLine.data[0].yAxis).toBe(60)
    expect(bar.markLine.label.formatter).toContain('60')
    expect(bar.markLine.label.formatter).toContain('Moyenne')
    // Axe X catégoriel, 1 étiquette par match.
    expect(opt.xAxis!.type).toBe('category')
    expect(opt.xAxis!.data).toHaveLength(3)
    // Échelle Y FIXE 0..100 (le max 100 doit toujours être affiché).
    expect(opt.yAxis!.min).toBe(0)
    expect(opt.yAxis!.max).toBe(100)
  })

  it('option vide (pas de série) si aucun point', () => {
    const opt = buildSessionPerfOption([], { scoreLabel: 'S', meanLabel: 'M' }) as unknown as PerfOptShape
    expect(opt.series).toBeUndefined()
  })
})
