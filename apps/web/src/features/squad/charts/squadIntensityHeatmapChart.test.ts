/**
 * squadIntensityHeatmapChart.test.ts — teammates.15.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildSquadIntensityHeatmapOption, SQUAD_INTENSITY_PHASE_LABELS } from './squadIntensityHeatmapChart'
import type { SquadIntensityMatchRow } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

function row(matchID: string, label: string, phases: number[]): SquadIntensityMatchRow {
  return { match_id: matchID, label, phases }
}

beforeEach(() => { vi.clearAllMocks() })

describe('buildSquadIntensityHeatmapOption', () => {
  it('vide → option minimale', () => {
    expect(buildSquadIntensityHeatmapOption([], { zLabel: 'Cadence' })).toMatchObject({ backgroundColor: 'transparent' })
  })

  it('xAxis = 10 phases canoniques 0-10..90-100%', () => {
    const opt = buildSquadIntensityHeatmapOption(
      [row('m1', 'Aquarius — 30/04', [0, 0.5, 1, 0.3, 0, 0, 0, 0, 0, 0])],
      { zLabel: 'Cadence' },
    )
    const xAxis = opt.xAxis as { data: string[] }
    expect(xAxis.data).toEqual(SQUAD_INTENSITY_PHASE_LABELS)
    expect(xAxis.data[0]).toBe('0-10%')
    expect(xAxis.data[9]).toBe('90-100%')
  })

  it('yAxis = labels « #N Carte » (carte extraite du label) avec inverse', () => {
    const opt = buildSquadIntensityHeatmapOption(
      [row('m1', 'Aquarius — 30/04', new Array(10).fill(0)), row('m2', 'Bazaar — 01/05', new Array(10).fill(0))],
      { zLabel: 'Cadence' },
    )
    const yAxis = opt.yAxis as { data: string[]; inverse: boolean }
    expect(yAxis.data).toEqual(['#1 Aquarius', '#2 Bazaar'])
    expect(yAxis.inverse).toBe(true)
  })

  it('data heatmap = matrice (xi, yi, value) pour chaque cellule', () => {
    const opt = buildSquadIntensityHeatmapOption(
      [row('m1', 'A', [0, 0.5, 1, 0.3, 0, 0, 0, 0, 0, 0])],
      { zLabel: 'Cadence' },
    )
    const series = opt.series as Array<{ data: Array<[number, number, number]> }>
    expect(series[0].data).toHaveLength(10)
    const cellPeak = series[0].data.find((d) => d[0] === 2) // phase 2 = 1.0
    expect(cellPeak?.[2]).toBe(1)
  })

  it('visualMap continu rampe fréquence (mono-teinte CVD-safe, via heatmapRampTokens)', () => {
    const opt = buildSquadIntensityHeatmapOption(
      [row('m1', 'A', [0.5, 0, 0, 0, 0, 0, 0, 0, 0, 0])],
      { zLabel: 'Cadence' },
    )
    const vm = opt.visualMap as { type: string; min: number; max: number; inRange: { color: string[] } }
    expect(vm.type).toBe('continuous')
    expect(vm.min).toBe(0)
    expect(vm.max).toBe(1)
    // Rampe NEUTRE de fréquence : heatmap-freq-low → heatmap-freq-high (2 stops).
    expect(vm.inRange.color).toEqual(['color:heatmap-freq-low', 'color:heatmap-freq-high'])
  })

  it('phases manquantes (length<10) → 0 par défaut', () => {
    const opt = buildSquadIntensityHeatmapOption(
      [row('m1', 'A', [0.5, 0.3])], // seulement 2 phases
      { zLabel: 'Cadence' },
    )
    const series = opt.series as Array<{ data: Array<[number, number, number]> }>
    expect(series[0].data).toHaveLength(10)
    expect(series[0].data[5][2]).toBe(0)
  })
})
