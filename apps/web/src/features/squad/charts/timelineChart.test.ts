/**
 * timelineChart.test.ts — buildTimelineSeries produit 2 ChartSeries<ChartPoint2D>
 * sans libellé hardcodé.
 */
import { describe, it, expect } from 'vitest'
import { buildTimelineSeries } from './timelineChart'
import type { SquadTimeseriesPoint } from '@/lib/api/types'

const POINT = (period_label: string, win_rate: number): SquadTimeseriesPoint => ({
  period_label,
  win_rate,
  avg_performance: 65,
  match_count: 3,
})

describe('buildTimelineSeries', () => {
  it('retourne [] pour points vide', () => {
    expect(
      buildTimelineSeries({ points: [], perfName: 'P', winRateName: 'W' }),
    ).toEqual([])
  })

  it('retourne 2 séries (perf + winrate)', () => {
    const result = buildTimelineSeries({
      points: [POINT('S1', 0.6), POINT('S2', 0.55)],
      perfName: 'T_PERF',
      winRateName: 'T_WIN',
    })
    expect(result).toHaveLength(2)
    expect(result[0].key).toBe('perf')
    expect(result[1].key).toBe('winrate')
  })

  it('reporte le nom de série dans meta.name', () => {
    const result = buildTimelineSeries({
      points: [POINT('S1', 0.6)],
      perfName: 'T_PERF',
      winRateName: 'T_WIN',
    })
    expect(result[0].meta?.name).toBe('T_PERF')
    expect(result[1].meta?.name).toBe('T_WIN')
  })

  it('mappe period_label → x et avg_performance → y pour perf', () => {
    const result = buildTimelineSeries({
      points: [POINT('S1', 0.6), POINT('S2', 0.55)],
      perfName: 'P',
      winRateName: 'W',
    })
    expect(result[0].datapoints[0]).toMatchObject({ x: 'S1', y: 65 })
    expect(result[0].datapoints[1]).toMatchObject({ x: 'S2', y: 65 })
  })

  it('convertit win_rate × 100 pour winrate', () => {
    const result = buildTimelineSeries({
      points: [POINT('S1', 0.6)],
      perfName: 'P',
      winRateName: 'W',
    })
    expect(result[1].datapoints[0]).toMatchObject({ x: 'S1', y: 60 })
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const result = buildTimelineSeries({
      points: [POINT('S1', 0.6)],
      perfName: 'P',
      winRateName: 'W',
    })
    const json = JSON.stringify(result)
    expect(json).not.toMatch(/Perf\. moy\./)
    expect(json).not.toMatch(/Win rate \(%\)/)
    expect(json).not.toMatch(/Évolution des performances/)
  })
})
