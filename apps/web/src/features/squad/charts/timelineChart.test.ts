/**
 * timelineChart.test.ts — Le builder accepte les libellés en argument,
 * aucune string FR libre.
 */
import { describe, it, expect } from 'vitest'
import { buildTimelineChart } from './timelineChart'
import type { SquadTimeseriesPoint } from '@/lib/api/types'

const POINT = (period_label: string, win_rate: number): SquadTimeseriesPoint => ({
  period_label,
  win_rate,
  avg_performance: 65,
  match_count: 3,
})

describe('buildTimelineChart', () => {
  const labels = {
    title: 'T_TITLE',
    perfName: 'T_PERF',
    winRateName: 'T_WIN',
    perfAxis: 'T_PERF_AXIS',
    winRateAxis: 'T_WIN_AXIS',
  }

  it('retourne null pour points vide', () => {
    expect(buildTimelineChart({ points: [], ...labels })).toBeNull()
  })

  it('reporte tous les libellés fournis dans le layout', () => {
    const fig = buildTimelineChart({
      points: [POINT('S1', 60), POINT('S2', 55)],
      ...labels,
    })
    expect(fig!.layout.title).toEqual({ text: 'T_TITLE', font: { size: 13 } })
    expect(fig!.layout.yaxis).toMatchObject({ title: 'T_PERF_AXIS' })
    expect(fig!.layout.yaxis2).toMatchObject({ title: 'T_WIN_AXIS' })
    const names = fig!.data.map((t) => (t as { name?: string }).name)
    expect(names).toContain('T_PERF')
    expect(names).toContain('T_WIN')
  })

  it('aucun libellé hardcodé en français résiduel', () => {
    const fig = buildTimelineChart({
      points: [POINT('S1', 60)],
      ...labels,
    })
    const json = JSON.stringify(fig)
    expect(json).not.toMatch(/Perf\. moy\./)
    expect(json).not.toMatch(/Win rate \(%\)/)
    expect(json).not.toMatch(/Score perf\./)
    expect(json).not.toMatch(/Évolution des performances/)
  })
})
