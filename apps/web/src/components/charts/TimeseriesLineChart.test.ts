import { describe, it, expect, vi } from 'vitest'

import {
  buildTimeseriesLineOption,
  type ChartPoint2D,
} from './TimeseriesLineChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildTimeseriesLineOption', () => {
  const t0 = '2026-04-01T10:00:00Z'
  const t1 = '2026-04-01T11:00:00Z'

  const series: ChartSeries<ChartPoint2D>[] = [
    {
      key: 'squad.contrib.assists.main',
      meta: { gamertag: 'main' },
      datapoints: [
        { x: t0, y: 5, label: 'win' },
        { x: t1, y: 3, label: 'loss' },
      ],
    },
    {
      key: 'squad.contrib.assists.f1',
      meta: { gamertag: 'f1' },
      datapoints: [{ x: t0, y: 2 }],
    },
  ]

  it('génère 1 ECharts series par ChartSeries', () => {
    const opt = buildTimeseriesLineOption(series) as { series: { name: string }[] }
    expect(opt.series).toHaveLength(2)
    expect(opt.series[0].name).toBe('main')
    expect(opt.series[1].name).toBe('f1')
  })

  it('xAxis time par défaut', () => {
    const opt = buildTimeseriesLineOption(series) as { xAxis: { type: string } }
    expect(opt.xAxis.type).toBe('time')
  })

  it('xAxis category si timeAxis=false', () => {
    const opt = buildTimeseriesLineOption(series, { timeAxis: false }) as {
      xAxis: { type: string }
    }
    expect(opt.xAxis.type).toBe('category')
  })

  it('marker outcome appliqué quand outcomeMarkers=true (default)', () => {
    const opt = buildTimeseriesLineOption(series) as {
      series: { data: { itemStyle?: { color: string } }[] }[]
    }
    const dpWin = opt.series[0].data[0]
    expect(dpWin.itemStyle?.color).toBe('var(outcome-win)')
  })

  it('marker outcome désactivé si outcomeMarkers=false', () => {
    const opt = buildTimeseriesLineOption(series, { outcomeMarkers: false }) as {
      series: { data: { itemStyle?: unknown }[] }[]
    }
    expect(opt.series[0].data[0].itemStyle).toBeUndefined()
  })

  it('seriesNameResolver override le nom', () => {
    const opt = buildTimeseriesLineOption(series, {
      seriesNameResolver: (s) => `Trace-${s.key}`,
    }) as { series: { name: string }[] }
    expect(opt.series[0].name).toBe('Trace-squad.contrib.assists.main')
  })

  it('series vide retourne option minimal', () => {
    expect(buildTimeseriesLineOption([])).toEqual({ backgroundColor: 'transparent' })
  })
})
