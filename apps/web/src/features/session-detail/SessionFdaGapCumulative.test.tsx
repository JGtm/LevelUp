/**
 * Tests B2 (plan PLAN_EXPECTED_FDA) — « Écart cumulé au FDA attendu ».
 *
 * - `computeCumulativeFdaGap` (pur) : cumul chronologique + report D5 (un match
 *   sans attendu ne modifie pas le cumul, la courbe reporte la dernière valeur).
 * - `buildSessionFdaGapOption` (pur) : aire signée divergente ancrée à 0 + markLine 0,
 *   PLUS 2 courbes fines « FDA réel/attendu (cumulé) » sur l'axe secondaire
 *   (`yAxisIndex: 1`).
 * - `SessionFdaGapCumulative` (composant) : masquage par capability `expected_stats`.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import {
  SessionFdaGapCumulative,
  buildSessionFdaGapOption,
  computeCumulativeFdaGap,
  type FdaGapPoint,
} from './SessionFdaGapCumulative'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

function match(
  kda: number | undefined,
  kdaExpected: number | undefined,
  startTime: string,
  map = 'Bazaar',
): SessionDetailMatchRow {
  return {
    assists: 0,
    deaths: 0,
    is_ranked: false,
    kills: 0,
    match_id: `m-${startTime}`,
    pair_name: 'Slayer',
    playlist_name: 'pl',
    start_time: startTime,
    map_name: map,
    kda: kda,
    kda_expected: kdaExpected,
  }
}

interface OptShape {
  backgroundColor: string
  series?: Array<{
    type: string
    name?: string
    data: number[]
    yAxisIndex?: number
    areaStyle?: { origin?: number; color?: { type?: string } }
    lineStyle?: { color?: { type?: string } }
    markLine?: { data: Array<{ yAxis: number }> }
  }>
  legend?: { data: string[] }
  xAxis?: { data: string[]; boundaryGap?: boolean }
  yAxis?: unknown[]
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

describe('computeCumulativeFdaGap', () => {
  it('cumul chronologique du différentiel réel − attendu', () => {
    const pts = computeCumulativeFdaGap([
      match(1.5, 1.0, '2025-01-01T10:00:00Z'),
      match(0.8, 1.2, '2025-01-01T11:00:00Z'),
      match(2.0, 1.0, '2025-01-01T12:00:00Z'),
    ])
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.1, 1.1])
    expect(pts.map((p) => p.gap)).toEqual([0.5, -0.4, 1])
  })

  it('trie les matchs par start_time avant de cumuler', () => {
    const pts = computeCumulativeFdaGap([
      match(2.0, 1.0, '2025-01-01T12:00:00Z'),
      match(1.5, 1.0, '2025-01-01T10:00:00Z'),
    ])
    // Ordre chronologique : #1 = 10h (gap 0.5), #2 = 12h (gap 1.0) → cumul 0.5, 1.5.
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 1.5])
  })

  it('report D5 : un match sans attendu ne modifie pas le cumul (gap null)', () => {
    const pts = computeCumulativeFdaGap([
      match(1.5, 1.0, '2025-01-01T10:00:00Z'),
      match(0.8, undefined, '2025-01-01T11:00:00Z'),
      match(2.0, 1.0, '2025-01-01T12:00:00Z'),
    ])
    expect(pts[1].gap).toBeNull()
    // Le cumul reporte la dernière valeur (0.5), puis reprend (+1.0 = 1.5).
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.5, 1.5])
  })

  it('report D5 côté réel manquant également', () => {
    const pts = computeCumulativeFdaGap([
      match(1.5, 1.0, '2025-01-01T10:00:00Z'),
      match(undefined, 1.0, '2025-01-01T11:00:00Z'),
    ])
    expect(pts[1].gap).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([0.5, 0.5])
  })
})

describe('buildSessionFdaGapOption', () => {
  const opts = {
    seriesLabel: 'Écart cumulé',
    realLabel: 'Réel',
    expectedLabel: 'Attendu',
    gapLabel: 'Écart',
    realCumulativeLabel: 'FDA réel (cumulé)',
    expectedCumulativeLabel: 'FDA attendu (cumulé)',
  }

  it('série vide → option de fond minimale (pas de série)', () => {
    const opt = buildSessionFdaGapOption([], opts) as unknown as OptShape
    expect(opt.backgroundColor).toBeTruthy()
    expect(opt.series).toBeUndefined()
  })

  it('3 séries : cumul (aire), FDA réel cumulé + FDA attendu cumulé sur axe secondaire', () => {
    const series: ChartSeries<FdaGapPoint>[] = [
      {
        key: 'g',
        datapoints: computeCumulativeFdaGap([
          match(1.5, 1.0, '2025-01-01T10:00:00Z'),
          match(0.8, 1.2, '2025-01-01T11:00:00Z'),
        ]),
      },
    ]
    const opt = buildSessionFdaGapOption(series, opts) as unknown as OptShape
    expect(opt.series).toHaveLength(3)
    expect(opt.series![0].type).toBe('line')
    expect(opt.series![0].data).toEqual([0.5, 0.1])
    expect(opt.series![0].areaStyle?.origin).toBe(0)
    expect(opt.series![0].markLine?.data[0].yAxis).toBe(0)
    expect((opt.series![0].areaStyle?.color as { type?: string })?.type).toBe('linear')
    // Courbes FDA réel/attendu cumulés — axe SECONDAIRE (yAxisIndex 1).
    expect(opt.series![1].name).toBe('FDA réel (cumulé)')
    expect(opt.series![1].yAxisIndex).toBe(1)
    expect(opt.series![1].data).toEqual([1.5, 2.3])
    expect(opt.series![2].name).toBe('FDA attendu (cumulé)')
    expect(opt.series![2].yAxisIndex).toBe(1)
    expect(opt.series![2].data).toEqual([1, 2.2])
    // Axe Y en tableau (primaire + secondaire droit) + légende sur les 3 séries.
    expect(opt.yAxis).toHaveLength(2)
    expect(opt.legend?.data).toEqual(['Écart cumulé', 'FDA réel (cumulé)', 'FDA attendu (cumulé)'])
    expect(opt.xAxis?.boundaryGap).toBe(false)
  })
})

describe('SessionFdaGapCumulative — masquage capability', () => {
  const matches = [match(1.5, 1.0, '2025-01-01T10:00:00Z'), match(0.8, 1.2, '2025-01-01T11:00:00Z')]

  it('capability expected_stats présente → chart rendu', async () => {
    setTitleCaps(['expected_stats'])
    render(<SessionFdaGapCumulative title="Écart cumulé au FDA attendu" matches={matches} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('capability expected_stats absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <SessionFdaGapCumulative title="Écart cumulé au FDA attendu" matches={matches} />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})
