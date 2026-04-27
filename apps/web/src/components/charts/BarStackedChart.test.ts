import { describe, it, expect, vi } from 'vitest'

import {
  buildBarStackedOption,
  type ChartPointStacked,
} from './BarStackedChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildBarStackedOption', () => {
  const series: ChartSeries<ChartPointStacked>[] = [
    {
      key: 'test.stack',
      datapoints: [
        { category: 'Aquarius', components: { win: 5, loss: 2 } },
        { category: 'Recharge', components: { win: 3, loss: 4 } },
      ],
    },
  ]

  it('génère 1 ECharts series par component', () => {
    const opt = buildBarStackedOption(series) as { series: { name: string }[] }
    expect(opt.series).toHaveLength(2)
    expect(opt.series.map((s) => s.name).sort()).toEqual(['loss', 'win'])
  })

  it('catégories sur xAxis en orientation vertical (default)', () => {
    const opt = buildBarStackedOption(series) as { xAxis: { data: string[] } }
    expect(opt.xAxis.data).toEqual(['Aquarius', 'Recharge'])
  })

  it('catégories sur yAxis en orientation horizontal', () => {
    const opt = buildBarStackedOption(series, { orientation: 'horizontal' }) as {
      yAxis: { data: string[] }
    }
    expect(opt.yAxis.data).toEqual(['Aquarius', 'Recharge'])
  })

  it('toutes les bars partagent stack="total"', () => {
    const opt = buildBarStackedOption(series) as { series: { stack: string }[] }
    for (const s of opt.series) {
      expect(s.stack).toBe('total')
    }
  })

  it('respecte componentOrder', () => {
    const opt = buildBarStackedOption(series, {
      componentOrder: ['win', 'loss'],
    }) as { series: { name: string }[] }
    expect(opt.series.map((s) => s.name)).toEqual(['win', 'loss'])
  })

  it('applique componentColors via resolveToken', () => {
    const opt = buildBarStackedOption(series, {
      componentColors: { win: 'outcome-win', loss: 'outcome-loss' },
      componentOrder: ['win', 'loss'],
    }) as { series: { itemStyle: { color: string } }[] }
    expect(opt.series[0].itemStyle.color).toBe('var(outcome-win)')
    expect(opt.series[1].itemStyle.color).toBe('var(outcome-loss)')
  })

  it('series vide retourne option minimal', () => {
    const opt = buildBarStackedOption([])
    expect(opt).toEqual({ backgroundColor: 'transparent' })
  })

  it('component absent d\'un dp → 0 (pas crash)', () => {
    const partial: ChartSeries<ChartPointStacked>[] = [
      {
        key: 'test',
        datapoints: [
          { category: 'A', components: { win: 5 } },
          { category: 'B', components: { loss: 3 } },
        ],
      },
    ]
    const opt = buildBarStackedOption(partial) as { series: { name: string; data: number[] }[] }
    const winSeries = opt.series.find((s) => s.name === 'win')!
    expect(winSeries.data).toEqual([5, 0]) // B sans win → 0
  })
})
