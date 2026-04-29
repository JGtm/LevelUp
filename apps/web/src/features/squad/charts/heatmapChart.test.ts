/**
 * heatmapChart.test.ts — buildHeatmapSeries produit un ChartSeries<ChartPointHeatmap>
 * correctement trié et sans libellé hardcodé.
 */
import { describe, it, expect, vi } from 'vitest'
import { buildHeatmapSeries } from './heatmapChart'
import type { MapBreakdownRow } from '@/lib/api/types'

const ROWS: MapBreakdownRow[] = [
  { map_ui: 'Aquarius', match_count: 4, win_rate: 0.75 },
  { map_ui: 'Live Fire', match_count: 6, win_rate: 0.5 },
]

const id = (s: string) => s

describe('buildHeatmapSeries', () => {
  it('retourne [] pour rows vide', () => {
    expect(
      buildHeatmapSeries({ rows: [], winAxisLabel: 'WR', mapLabelOf: id }),
    ).toEqual([])
  })

  it('retourne 1 série avec 1 datapoint par carte', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf: id })
    expect(result).toHaveLength(1)
    expect(result[0].datapoints).toHaveLength(2)
  })

  it('trie les cartes par win_rate décroissant', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf: id })
    const xs = result[0].datapoints.map((d) => d.x)
    expect(xs).toEqual(['Aquarius', 'Live Fire'])
  })

  it('utilise winAxisLabel comme y pour tous les datapoints', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'H_AXIS', mapLabelOf: id })
    for (const dp of result[0].datapoints) {
      expect(dp.y).toBe('H_AXIS')
    }
  })

  it('mappe win_rate comme value', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf: id })
    expect(result[0].datapoints[0].value).toBe(75)
    expect(result[0].datapoints[1].value).toBe(50)
  })

  it('applique mapLabelOf pour chaque carte (axe x)', () => {
    const mapLabelOf = vi.fn((s: string) => `LOC_${s}`)
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf })
    const xs = result[0].datapoints.map((d) => d.x)
    expect(xs).toEqual(['LOC_Aquarius', 'LOC_Live Fire'])
    expect(mapLabelOf).toHaveBeenCalledWith('Aquarius')
    expect(mapLabelOf).toHaveBeenCalledWith('Live Fire')
  })

  it('mapLabelOf identité = fallback (ID brut inchangé)', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf: id })
    const xs = result[0].datapoints.map((d) => d.x)
    expect(xs).toEqual(['Aquarius', 'Live Fire'])
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const result = buildHeatmapSeries({ rows: ROWS, winAxisLabel: 'WR', mapLabelOf: id })
    const json = JSON.stringify(result)
    expect(json).not.toMatch(/Win rate par carte/)
    expect(json).not.toMatch(/Win %/)
    expect(json).not.toMatch(/Win rate \(%\)/)
  })
})
