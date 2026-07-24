/**
 * Tests V72-13 — « XP de carrière (estimée) » sur la page Sessions.
 *
 * - `buildSessionCareerXpOption` (pur) : barres XP/match (axe secondaire) +
 *   ligne XP cumulée (axe primaire), MIROIR de TimeseriesCareerXP.
 * - `SessionCareerXP` (composant) : auto-gate DATA-DRIVEN — masqué (null) sans
 *   aucune estimation sur la session, rendu sinon.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { SessionCareerXP, buildSessionCareerXpOption } from './SessionCareerXP'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

// `as unknown as SessionDetailMatchRow` : career_xp_estimated est ajouté à
// openapi.yaml (édition manuelle) mais generated.ts n'est régénéré que par le
// superviseur (contrat de cette tâche) — cast nécessaire tant que le champ
// n'existe pas encore sur le type généré.
function match(
  careerXpEstimated: number | undefined,
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
    career_xp_estimated: careerXpEstimated,
  } as unknown as SessionDetailMatchRow
}

interface OptShape {
  backgroundColor: string
  series?: Array<{
    type: string
    name?: string
    data: Array<number | null>
    yAxisIndex?: number
  }>
  legend?: { data: string[] }
  yAxis?: unknown[]
  xAxis?: { data: string[] }
}

interface CareerXpPoint {
  label: string
  perMatch: number | null
  cumulative: number | null
}

function makeSeries(points: CareerXpPoint[]): ChartSeries<CareerXpPoint>[] {
  return [{ key: 'career_xp', datapoints: points }]
}

describe('buildSessionCareerXpOption', () => {
  it('série vide → option de fond minimale (pas de série)', () => {
    const opt = buildSessionCareerXpOption([], {
      cumulativeLabel: 'Cumul',
      perMatchLabel: 'Par match',
    }) as unknown as OptShape
    expect(opt.backgroundColor).toBeTruthy()
    expect(opt.series).toBeUndefined()
  })

  it('2 séries : barres XP/match (axe 1) + ligne cumul (axe 0)', () => {
    const points: CareerXpPoint[] = [
      { label: '#1\nBazaar', perMatch: 100, cumulative: 100 },
      { label: '#2\nBazaar', perMatch: null, cumulative: 100 },
      { label: '#3\nBazaar', perMatch: 200, cumulative: 300 },
    ]
    const opt = buildSessionCareerXpOption(makeSeries(points), {
      cumulativeLabel: 'XP cumulée',
      perMatchLabel: 'XP / match',
    }) as unknown as OptShape

    expect(opt.series).toHaveLength(2)
    const bar = opt.series!.find((s) => s.type === 'bar')
    const line = opt.series!.find((s) => s.type === 'line')
    expect(bar?.yAxisIndex).toBe(1)
    expect(bar?.data).toEqual([100, null, 200])
    expect(line?.yAxisIndex).toBe(0)
    expect(line?.data).toEqual([100, 100, 300])
    // La légende ECharts se déduit des noms de séries (pas de data explicite).
    expect(opt.legend).toBeDefined()
    expect(opt.series!.map((s) => s.name)).toEqual(
      expect.arrayContaining(['XP / match', 'XP cumulée']),
    )
    expect(opt.yAxis).toHaveLength(2)
    expect(opt.xAxis?.data).toEqual(['#1\nBazaar', '#2\nBazaar', '#3\nBazaar'])
  })
})

describe('SessionCareerXP — auto-gate data-driven', () => {
  it('aucune estimation sur la session → non rendu (null)', () => {
    const matches = [match(undefined, '2025-01-01T10:00:00Z'), match(undefined, '2025-01-01T11:00:00Z')]
    const { container } = render(<SessionCareerXP title="XP de carrière (estimée)" matches={matches} />)
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('au moins un match avec estimation → chart rendu', async () => {
    const matches = [match(1000, '2025-01-01T10:00:00Z'), match(undefined, '2025-01-01T11:00:00Z')]
    render(<SessionCareerXP title="XP de carrière (estimée)" matches={matches} />)
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })
})
