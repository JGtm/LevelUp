/**
 * Tests P3 — « Balance des dégâts cumulée » (session).
 *
 * - `computeCumulativeNetLives` (pur) : cumul chronologique en vies + report D5.
 * - `buildSessionNetLivesOption` (pur) : aire signée divergente ancrée à 0 + markLine 0.
 * - `SessionNetLivesCumulative` (composant) : masquage capability `damage_taken`
 *   + pastille KPI « balance moyenne par match ».
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import {
  SessionNetLivesCumulative,
  buildSessionNetLivesOption,
  computeCumulativeNetLives,
  type NetLivesPoint,
} from './SessionNetLivesCumulative'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const HP = 225

function match(
  damageDealt: number | undefined,
  damageTaken: number | undefined,
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
    damage_dealt: damageDealt,
    damage_taken: damageTaken,
  } as SessionDetailMatchRow
}

interface OptShape {
  backgroundColor: string
  series?: Array<{
    type: string
    data: number[]
    areaStyle?: { origin?: number; color?: { type?: string } }
    markLine?: { data: Array<{ yAxis: number }> }
  }>
  xAxis?: { boundaryGap?: boolean }
}

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225, provides_damage_taken: true, provides_team_mmr: true, provides_max_killing_spree: true, offensive_conversion_p80: 0.9, defensive_resistance_p80: 1.65 },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('computeCumulativeNetLives', () => {
  it('cumul chronologique de la balance en vies', () => {
    const pts = computeCumulativeNetLives(
      [
        match(900, 450, '2025-01-01T10:00:00Z'), // +2
        match(450, 900, '2025-01-01T11:00:00Z'), // -2
        match(675, 225, '2025-01-01T12:00:00Z'), // +2
      ],
      HP,
    )
    expect(pts.map((p) => p.value)).toEqual([2, -2, 2])
    expect(pts.map((p) => p.cumulative)).toEqual([2, 0, 2])
  })

  it('trie les matchs par start_time avant de cumuler', () => {
    const pts = computeCumulativeNetLives(
      [match(675, 225, '2025-01-01T12:00:00Z'), match(900, 450, '2025-01-01T10:00:00Z')],
      HP,
    )
    // #1 = 10h (+2), #2 = 12h (+2) → cumul 2, 4.
    expect(pts.map((p) => p.cumulative)).toEqual([2, 4])
  })

  it('report D5 : un match sans dégâts subis ne modifie pas le cumul (value null)', () => {
    const pts = computeCumulativeNetLives(
      [
        match(900, 450, '2025-01-01T10:00:00Z'),
        match(450, undefined, '2025-01-01T11:00:00Z'),
        match(675, 225, '2025-01-01T12:00:00Z'),
      ],
      HP,
    )
    expect(pts[1].value).toBeNull()
    expect(pts.map((p) => p.cumulative)).toEqual([2, 2, 4])
  })
})

describe('buildSessionNetLivesOption', () => {
  const opts = { seriesLabel: 'Balance cumulée', matchLabel: 'Balance du match' }

  it('série vide → option de fond minimale (pas de série)', () => {
    const opt = buildSessionNetLivesOption([], opts) as unknown as OptShape
    expect(opt.backgroundColor).toBeTruthy()
    expect(opt.series).toBeUndefined()
  })

  it('une série ligne : data = cumul, aire ancrée à 0, markLine 0', () => {
    const series: ChartSeries<NetLivesPoint>[] = [
      {
        key: 'nl',
        datapoints: computeCumulativeNetLives(
          [match(900, 450, '2025-01-01T10:00:00Z'), match(450, 900, '2025-01-01T11:00:00Z')],
          HP,
        ),
      },
    ]
    const opt = buildSessionNetLivesOption(series, opts) as unknown as OptShape
    expect(opt.series).toHaveLength(1)
    expect(opt.series![0].type).toBe('line')
    expect(opt.series![0].data).toEqual([2, 0])
    expect(opt.series![0].areaStyle?.origin).toBe(0)
    expect(opt.series![0].markLine?.data[0].yAxis).toBe(0)
    expect((opt.series![0].areaStyle?.color as { type?: string })?.type).toBe('linear')
    expect(opt.xAxis?.boundaryGap).toBe(false)
  })
})

describe('SessionNetLivesCumulative — masquage capability + KPI', () => {
  const matches = [
    match(900, 450, '2025-01-01T10:00:00Z'), // +2
    match(450, 900, '2025-01-01T11:00:00Z'), // -2
  ]

  it('capability damage_taken présente → chart + pastille KPI rendus', async () => {
    setTitleCaps(['damage_taken'])
    render(<SessionNetLivesCumulative title="Balance des dégâts cumulée" matches={matches} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByTestId('net-lives-kpi')).toBeInTheDocument()
    // Moyenne (+2 et -2) = 0 → « +0,0 » (signDisplay always, séparateur FR/EN).
    expect(screen.getByText(/\+0[.,]0/)).toBeInTheDocument()
  })

  it('capability damage_taken absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <SessionNetLivesCumulative title="Balance des dégâts cumulée" matches={matches} />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})
