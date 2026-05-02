/**
 * squadMapHeatmapChart.test.ts — teammates.03.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadMapHeatmapOption } from './squadMapHeatmapChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
}))

const OPTS = {
  mapLabelOf: (m: string) => m.toUpperCase(),
  pieceLabels: { tier1: 'T1', tier2: 'T2', tier3: 'T3', tier4: 'T4', tier5: 'T5' },
  noScoreLabel: '-',
}

function makeData(): SquadMapHeatmap {
  return {
    players: ['Me', 'Friend1'],
    maps_topn: ['Aquarius', 'Bazaar'],
    cells: [
      { player: 'Me', map_ui: 'Aquarius', perf_avg: 80, match_count: 4 },
      { player: 'Me', map_ui: 'Bazaar', perf_avg: 45, match_count: 2 },
      { player: 'Friend1', map_ui: 'Aquarius', perf_avg: undefined, match_count: 0 },
      { player: 'Friend1', map_ui: 'Bazaar', perf_avg: 60, match_count: 1 },
    ],
  }
}

function makeSeries(d: SquadMapHeatmap | null): ChartSeries<SquadMapHeatmap>[] {
  return d ? [{ key: 'k', datapoints: [d] }] : []
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadMapHeatmapOption', () => {
  it('série vide → option minimale', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(null), OPTS)
    expect(opt).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('aucun joueur ou aucune carte → option minimale', () => {
    const empty: SquadMapHeatmap = { players: [], maps_topn: [], cells: [] }
    expect(buildSquadMapHeatmapOption(makeSeries(empty), OPTS)).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('mapLabelOf appliqué sur xAxis', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(['AQUARIUS', 'BAZAAR'])
  })

  it('yAxis = liste des joueurs avec inverse', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const yAxis = opt.yAxis as { data: string[]; inverse: boolean }
    expect(yAxis.data).toEqual(['Me', 'Friend1'])
    expect(yAxis.inverse).toBe(true)
  })

  it('data heatmap = matrice (xi, yi, value) avec value=null pour cellule sans score', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const series = opt.series as Array<{ data: Array<[number, number, number | null]> }>
    expect(series[0].data).toHaveLength(4)
    // Friend1 × Aquarius (xi=0, yi=1) doit être null.
    const cellNull = series[0].data.find((d) => d[0] === 0 && d[1] === 1)
    expect(cellNull?.[2]).toBeNull()
    // Me × Aquarius (xi=0, yi=0) doit valoir 80.
    const cellMe = series[0].data.find((d) => d[0] === 0 && d[1] === 0)
    expect(cellMe?.[2]).toBe(80)
  })

  it('visualMap discret 5 paliers avec tokens perf-tier', () => {
    const opt = buildSquadMapHeatmapOption(makeSeries(makeData()), OPTS)
    const vm = opt.visualMap as { type: string; pieces: Array<{ color: string; label: string }> }
    expect(vm.type).toBe('piecewise')
    expect(vm.pieces).toHaveLength(5)
    expect(vm.pieces[0].color).toBe('color:perf-tier-5') // <30
    expect(vm.pieces[4].color).toBe('color:perf-tier-1') // ≥75
  })
})
