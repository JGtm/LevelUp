/**
 * Tests B1 (plan PLAN_EXPECTED_FDA) — « Écart au FDA attendu ».
 *
 * - `buildFdaGapDiffOption` (pur) : série nominale + trous D5 (match sans attendu
 *   = null, jamais 0). `resolveToken` renvoie '' hors runtime CSS → on teste la
 *   structure/les données, pas les couleurs.
 * - `TimeseriesFdaGapTrend` (composant) : masquage par capability `expected_stats`
 *   (absente → non rendu). echarts-for-react mocké (canvas jsdom instable).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { TimeseriesMatchRow } from '@/lib/api/types'

import { TimeseriesFdaGapTrend, buildFdaGapDiffOption, type FdaGapDiffLabels } from './TimeseriesFdaGapTrend'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const LABELS: FdaGapDiffLabels = { gap: 'Écart', real: 'Réel', expected: 'Attendu', smoothing: 'Tendance' }

function row(kda: number | undefined, kdaExpected: number | undefined, map = 'Bazaar'): TimeseriesMatchRow {
  return {
    accuracy: null,
    assists: 0,
    damage_dealt: null,
    damage_taken: null,
    deaths: 0,
    index: 0,
    kills: 0,
    match_id: 'm',
    outcome: null,
    perf_score: null,
    personal_score: null,
    playlist_name: 'pl',
    rank: null,
    start_time: '2025-01-01T00:00:00Z',
    time_played_seconds: null,
    map_name: map,
    kda: kda,
    kda_expected: kdaExpected,
  }
}

interface OptShape {
  series: Array<{
    type: string
    name: string
    data: (number | null)[]
    areaStyle?: { origin?: number; color?: { type?: string } }
    lineStyle?: { color?: { type?: string } | string }
    markLine?: { data: Array<{ yAxis: number }> }
    connectNulls?: boolean
  }>
  xAxis: { data: string[] }
}

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225 },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('buildFdaGapDiffOption', () => {
  it('série vide → null', () => {
    expect(buildFdaGapDiffOption([], LABELS)).toBeNull()
  })

  it('série nominale : aire différentielle brute + ligne lissée, markLine 0, aire ancrée à 0', () => {
    const opt = buildFdaGapDiffOption(
      [row(1.5, 1.0), row(0.8, 1.2), row(2.0, 1.0)],
      LABELS,
    ) as unknown as OptShape
    // 2 séries : aire (différentiel) + tendance lissée.
    expect(opt.series).toHaveLength(2)
    expect(opt.series[0].name).toBe('Écart')
    expect(opt.series[1].name).toBe('Tendance')
    // Différentiel brut kda - kda_expected, arrondi 2 décimales.
    expect(opt.series[0].data).toEqual([0.5, -0.4, 1])
    // Aire ancrée à 0 + markLine 0 + dégradé divergent (linéaire).
    expect(opt.series[0].areaStyle?.origin).toBe(0)
    expect(opt.series[0].markLine?.data[0].yAxis).toBe(0)
    expect((opt.series[0].areaStyle?.color as { type?: string })?.type).toBe('linear')
    expect(opt.xAxis.data).toHaveLength(3)
  })

  it('trous D5 : un match sans attendu → null (jamais 0), lissage préserve le trou', () => {
    const opt = buildFdaGapDiffOption(
      [row(1.5, 1.0), row(0.8, undefined), row(2.0, 1.0)],
      LABELS,
    ) as unknown as OptShape
    // Différentiel : trou au milieu (pas 0).
    expect(opt.series[0].data[1]).toBeNull()
    expect(opt.series[0].data[0]).toBe(0.5)
    expect(opt.series[0].data[2]).toBe(1)
    // La tendance lissée saute aussi le trou (préservation).
    expect(opt.series[1].data[1]).toBeNull()
    // L'aire brute ne relie pas les trous.
    expect(opt.series[0].connectNulls).toBe(false)
  })

  it('trou D5 côté réel manquant → null également', () => {
    const opt = buildFdaGapDiffOption([row(undefined, 1.0), row(2.0, 1.0)], LABELS) as unknown as OptShape
    expect(opt.series[0].data[0]).toBeNull()
    expect(opt.series[0].data[1]).toBe(1)
  })
})

describe('TimeseriesFdaGapTrend — masquage capability', () => {
  it('capability expected_stats présente → chart rendu', async () => {
    setTitleCaps(['expected_stats'])
    render(<TimeseriesFdaGapTrend rows={[row(1.5, 1.0), row(0.8, 1.2)]} labels={LABELS} title="Écart au FDA attendu" />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('capability expected_stats absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <TimeseriesFdaGapTrend rows={[row(1.5, 1.0), row(0.8, 1.2)]} labels={LABELS} title="Écart au FDA attendu" />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})
