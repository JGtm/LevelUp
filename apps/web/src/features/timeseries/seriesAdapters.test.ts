/**
 * Tests — seriesAdapters (Phase 2 P2.E).
 */
import { describe, it, expect } from 'vitest'
import {
  cumulativePointsToSeries,
  heatmapCellsToSeries,
  DOW_LABELS_FR,
  DOW_LABELS_EN,
} from './seriesAdapters'
import type { CumulativePoint, IntensityHeatmapPoint } from '@/lib/api/types'

describe('cumulativePointsToSeries', () => {
  it('retourne 1 série mono-trace avec key/name', () => {
    const points: CumulativePoint[] = [
      { index: 0, start_time: '2025-01-01T10:00:00Z', value: 1.5 },
      { index: 1, start_time: '2025-01-02T10:00:00Z', value: 1.7 },
    ]
    const series = cumulativePointsToSeries(points, { key: 'k', name: 'KD cumulé' })
    expect(series).toHaveLength(1)
    expect(series[0].key).toBe('k')
    expect(series[0].meta).toEqual({ gamertag: 'KD cumulé' })
    expect(series[0].datapoints).toEqual([
      { x: '2025-01-01T10:00:00Z', y: 1.5 },
      { x: '2025-01-02T10:00:00Z', y: 1.7 },
    ])
  })

  it('retourne une série vide quand aucun point', () => {
    const series = cumulativePointsToSeries([], { key: 'k', name: 'n' })
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([])
  })

  it('préserve l\'ordre des points', () => {
    const points: CumulativePoint[] = [
      { index: 2, start_time: '2025-01-03T00:00:00Z', value: 3 },
      { index: 0, start_time: '2025-01-01T00:00:00Z', value: 1 },
      { index: 1, start_time: '2025-01-02T00:00:00Z', value: 2 },
    ]
    const dps = cumulativePointsToSeries(points, { key: 'k', name: 'n' })[0].datapoints
    expect(dps.map((d) => d.y)).toEqual([3, 1, 2])
  })
})

describe('heatmapCellsToSeries', () => {
  const cells: IntensityHeatmapPoint[] = [
    { day_of_week: 0, hour: 9, count: 3, avg_kd: 1.4 },
    { day_of_week: 6, hour: 22, count: 7, avg_kd: 0.8 },
  ]

  it('résout day_of_week via dowLabels FR', () => {
    const series = heatmapCellsToSeries(cells, {
      key: 'k',
      name: 'n',
      dowLabels: DOW_LABELS_FR,
    })
    const dps = series[0].datapoints
    expect(dps[0].y).toBe('Lun')
    expect(dps[1].y).toBe('Dim')
  })

  it('résout day_of_week via dowLabels EN', () => {
    const series = heatmapCellsToSeries(cells, {
      key: 'k',
      name: 'n',
      dowLabels: DOW_LABELS_EN,
    })
    const dps = series[0].datapoints
    expect(dps[0].y).toBe('Mon')
    expect(dps[1].y).toBe('Sun')
  })

  it('formate l\'heure en chaîne 2 chiffres', () => {
    const series = heatmapCellsToSeries(cells, {
      key: 'k',
      name: 'n',
      dowLabels: DOW_LABELS_FR,
    })
    expect(series[0].datapoints[0].x).toBe('09')
    expect(series[0].datapoints[1].x).toBe('22')
  })

  it('expose count comme value et avg_kd dans detail', () => {
    const series = heatmapCellsToSeries(cells, {
      key: 'k',
      name: 'n',
      dowLabels: DOW_LABELS_FR,
    })
    expect(series[0].datapoints[0].value).toBe(3)
    expect(series[0].datapoints[0].detail).toEqual({ avg_kd: 1.4 })
  })

  it('retourne une série vide quand aucune cellule', () => {
    const series = heatmapCellsToSeries([], {
      key: 'k',
      name: 'n',
      dowLabels: DOW_LABELS_FR,
    })
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([])
  })

  it('retombe sur le numéro brut si dow hors plage', () => {
    const series = heatmapCellsToSeries(
      [{ day_of_week: 99, hour: 0, count: 1, avg_kd: 0 }],
      { key: 'k', name: 'n', dowLabels: DOW_LABELS_FR },
    )
    expect(series[0].datapoints[0].y).toBe('99')
  })
})
