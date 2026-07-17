import { describe, it, expect, vi } from 'vitest'

import {
  buildHeatmap2DOption,
  type ChartPointHeatmap,
} from './Heatmap2DChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildHeatmap2DOption', () => {
  const series: ChartSeries<ChartPointHeatmap>[] = [
    {
      key: 'heatmap.test',
      datapoints: [
        { x: 'Aquarius', y: 'main', value: 75 },
        { x: 'Aquarius', y: 'f1', value: 60 },
        { x: 'Recharge', y: 'main', value: 80 },
      ],
    },
  ]

  it('extrait les axes uniques', () => {
    const opt = buildHeatmap2DOption(series) as {
      xAxis: { data: string[] }
      yAxis: { data: string[] }
    }
    expect(opt.xAxis.data).toEqual(['Aquarius', 'Recharge'])
    expect(opt.yAxis.data).toEqual(['main', 'f1'])
  })

  it('génère data au format [xIdx, yIdx, value, detail?]', () => {
    const opt = buildHeatmap2DOption(series) as {
      series: { data: unknown[][] }[]
    }
    expect(opt.series[0].data).toHaveLength(3)
    // Aquarius/main = [0, 0, 75, undefined] — 4e élément `detail` ajouté par
    // synthesis-kpi-grid (refonte chart : payload optionnel pour tooltip riche).
    expect(opt.series[0].data[0].slice(0, 3)).toEqual([0, 0, 75])
    // Recharge/main = [1, 0, 80, undefined]
    expect(opt.series[0].data[2].slice(0, 3)).toEqual([1, 0, 80])
  })

  it('palette sequential par défaut', () => {
    const opt = buildHeatmap2DOption(series) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-cold)',
      'var(heatmap-hot)',
    ])
  })

  it('palette divergent si paletteMode=divergent', () => {
    const opt = buildHeatmap2DOption(series, { paletteMode: 'divergent' }) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-divergent-low)',
      'var(divergent-neutral)',
      'var(heatmap-divergent-high)',
    ])
  })

  it('palette CVD : une heatmap séquentielle bascule sur la rampe fréquence (CVD-safe)', () => {
    const opt = buildHeatmap2DOption(series, { colorPalette: 'cividis' }) as {
      visualMap: { inRange: { color: string[] } }
    }
    expect(opt.visualMap.inRange.color).toEqual([
      'var(heatmap-freq-low)',
      'var(heatmap-freq-high)',
    ])
  })

  it('valueRange override min/max', () => {
    const opt = buildHeatmap2DOption(series, { valueRange: [0, 100] }) as {
      visualMap: { min: number; max: number }
    }
    expect(opt.visualMap.min).toBe(0)
    expect(opt.visualMap.max).toBe(100)
  })

  it('series vide retourne option minimal', () => {
    expect(buildHeatmap2DOption([])).toEqual({ backgroundColor: 'transparent' })
  })
})
