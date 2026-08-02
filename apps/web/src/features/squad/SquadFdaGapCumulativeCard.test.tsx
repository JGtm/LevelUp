/**
 * SquadFdaGapCumulativeCard.test.tsx — Lot C (D3/D4).
 *
 * Masquage par capability `expected_stats` (self-gate, retour null) + rendu des
 * pastilles KPI « écart moyen par match » (format signé + suffixe i18n).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import { SquadFdaGapCumulativeCard } from './SquadFdaGapCumulativeCard'
import { getSquadText } from './i18n'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const T = getSquadText('fr')

function pt(
  order: number,
  kda: number | undefined,
  kdaExpected: number | undefined,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    kda,
    kda_expected: kdaExpected,
  }
}

const ROWS: Record<string, SquadPerformanceSeriesPoint[]> = {
  Me: [pt(0, 1.6, 1.0), pt(1, 1.4, 1.0)], // écart moyen +0,5
  F1: [pt(0, 0.5, 1.0), pt(1, 0.5, 1.0)], // écart moyen -0,5
}
const ORDER = ['Me', 'F1']
const COLORS = { Me: '#aaa', F1: '#bbb' }

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      {
        slug: 'test_title',
        name: 'Test',
        status: 'active',
        capabilities: caps,
        is_default: true,
        effective_hp_to_kill: 225,
        provides_damage_taken: true,
        provides_team_mmr: true,
        provides_max_killing_spree: true,
        offensive_conversion_p80: 0.9,
        defensive_resistance_p80: 1.65,
      },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  vi.restoreAllMocks()
})

describe('SquadFdaGapCumulativeCard', () => {
  it('capability expected_stats présente → chart + pastilles KPI rendus', async () => {
    setTitleCaps(['expected_stats'])
    render(
      <SquadFdaGapCumulativeCard rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByTestId('fda-gap-kpis')).toBeInTheDocument()
    expect(screen.getByText(T.fdaGap.averageCaption)).toBeInTheDocument()
    expect(screen.getByText('Me')).toBeInTheDocument()
    expect(screen.getByText('F1')).toBeInTheDocument()
    // Pastille signée + suffixe i18n (« /match »), séparateur décimal FR ou EN.
    expect(screen.getByText(/\+0[.,]5\/match/)).toBeInTheDocument()
    expect(screen.getByText(/-0[.,]5\/match/)).toBeInTheDocument()
  })

  it('capability expected_stats absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <SquadFdaGapCumulativeCard rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })

  it('pastille « — » pour un joueur sans match avec attendu (D5)', () => {
    setTitleCaps(['expected_stats'])
    const rows = { Me: [pt(0, 1.0, undefined), pt(1, 2.0, undefined)] }
    render(
      <SquadFdaGapCumulativeCard
        rowsByPlayer={rows}
        playerOrder={['Me']}
        colorByPlayer={{ Me: '#aaa' }}
        t={T}
      />,
    )
    expect(screen.getByTestId('fda-gap-kpis')).toBeInTheDocument()
    expect(screen.getByText('—')).toBeInTheDocument()
  })
})
