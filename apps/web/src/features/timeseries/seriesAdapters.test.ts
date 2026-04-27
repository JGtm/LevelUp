/**
 * Tests — seriesAdapters (Phase 2 P2.E).
 */
import { describe, it, expect } from 'vitest'
import {
  cumulativePointsToSeries,
  heatmapCellsToSeries,
  distributionBucketsToSeries,
  correlationPointsToSeries,
  DOW_LABELS_FR,
  DOW_LABELS_EN,
} from './seriesAdapters'
import type {
  CorrelationDataPair,
  CumulativePoint,
  DistributionBucket,
  IntensityHeatmapPoint,
} from '@/lib/api/types'

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

describe('distributionBucketsToSeries', () => {
  const buckets: DistributionBucket[] = [
    { bin_start: 0, bin_end: 1, count: 5 },
    { bin_start: 1, bin_end: 2, count: 12 },
  ]

  it('retourne une série mono-trace avec key/name', () => {
    const series = distributionBucketsToSeries(buckets, { key: 'k', name: 'KD' })
    expect(series).toHaveLength(1)
    expect(series[0].key).toBe('k')
    expect(series[0].meta).toEqual({ gamertag: 'KD' })
  })

  it('mappe bin_start/bin_end/count → binStart/binEnd/count', () => {
    const dps = distributionBucketsToSeries(buckets, { key: 'k', name: 'n' })[0].datapoints
    expect(dps).toEqual([
      { binStart: 0, binEnd: 1, count: 5 },
      { binStart: 1, binEnd: 2, count: 12 },
    ])
  })

  it('retourne une série vide quand aucun bucket', () => {
    const series = distributionBucketsToSeries([], { key: 'k', name: 'n' })
    expect(series).toHaveLength(1)
    expect(series[0].datapoints).toEqual([])
  })
})

describe('correlationPointsToSeries', () => {
  const labels = { win: 'Victoires', loss: 'Défaites', unknown: 'Inconnu' }
  const points: CorrelationDataPair[] = [
    { label: 'kills_vs_kd', x: 5, y: 1.5, outcome: 2 },
    { label: 'kills_vs_kd', x: 3, y: 0.7, outcome: 3 },
    { label: 'kills_vs_kd', x: 0, y: 0, outcome: null },
    { label: 'other_pair', x: 9, y: 9, outcome: 2 },
  ]

  it('filtre par label actif', () => {
    const series = correlationPointsToSeries(points, 'kills_vs_kd', labels)
    const all = series.flatMap((s) => s.datapoints)
    expect(all).toHaveLength(3)
  })

  it('découpe en 3 séries win/loss/unknown', () => {
    const series = correlationPointsToSeries(points, 'kills_vs_kd', labels)
    expect(series.map((s) => s.key)).toEqual(['outcome.win', 'outcome.loss', 'outcome.unknown'])
    expect(series[0].datapoints).toEqual([{ x: 5, y: 1.5 }])
    expect(series[1].datapoints).toEqual([{ x: 3, y: 0.7 }])
    expect(series[2].datapoints).toEqual([{ x: 0, y: 0 }])
  })

  it('utilise les labels i18n pour meta.gamertag', () => {
    const series = correlationPointsToSeries(points, 'kills_vs_kd', labels)
    expect(series.map((s) => (s.meta as { gamertag: string }).gamertag)).toEqual([
      'Victoires',
      'Défaites',
      'Inconnu',
    ])
  })

  it('omet une série vide (pas de losses → 2 séries)', () => {
    const winsOnly: CorrelationDataPair[] = [
      { label: 'kills_vs_kd', x: 5, y: 1.5, outcome: 2 },
      { label: 'kills_vs_kd', x: 8, y: 2.0, outcome: 2 },
    ]
    const series = correlationPointsToSeries(winsOnly, 'kills_vs_kd', labels)
    expect(series).toHaveLength(1)
    expect(series[0].key).toBe('outcome.win')
  })

  it('retourne [] si label inconnu', () => {
    const series = correlationPointsToSeries(points, 'unknown_label', labels)
    expect(series).toEqual([])
  })
})
