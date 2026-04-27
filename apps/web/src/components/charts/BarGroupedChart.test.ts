import { describe, it, expect, vi } from 'vitest'

import { buildBarGroupedOption } from './BarGroupedChart'
import type { ChartPointStacked } from './BarStackedChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildBarGroupedOption', () => {
  const series: ChartSeries<ChartPointStacked>[] = [
    {
      key: 'test.grouped',
      datapoints: [
        { category: 'main', components: { kpm: 1.0, dpm: 0.5, apm: 0.8 } },
        { category: 'f1', components: { kpm: 1.2, dpm: 0.6, apm: 0.0 } },
      ],
    },
  ]

  it('génère 1 series par component', () => {
    const opt = buildBarGroupedOption(series) as { series: { name: string }[] }
    expect(opt.series).toHaveLength(3)
    expect(opt.series.map((s) => s.name).sort()).toEqual(['apm', 'dpm', 'kpm'])
  })

  it('aucun stack (groupé)', () => {
    const opt = buildBarGroupedOption(series) as { series: Record<string, unknown>[] }
    for (const s of opt.series) {
      expect(s.stack).toBeUndefined()
    }
  })

  it('xAxis = categories', () => {
    const opt = buildBarGroupedOption(series) as { xAxis: { data: string[] } }
    expect(opt.xAxis.data).toEqual(['main', 'f1'])
  })

  it('series vide retourne option minimal', () => {
    expect(buildBarGroupedOption([])).toEqual({ backgroundColor: 'transparent' })
  })
})
