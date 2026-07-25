/**
 * SquadNetLivesChart.test.tsx — « Balance des dégâts cumulée » (onglet Dynamique).
 *
 * Masquage par capability `damage_taken` (self-gate, retour null) : un titre sans
 * dégâts subis (ex. Halo 5) ne peut pas calculer la balance → carte masquée.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import { SquadNetLivesChart } from './SquadNetLivesChart'
import { getSquadText } from './i18n'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

const T = getSquadText('fr')

function pt(
  order: number,
  damageDealt: number | undefined,
  damageTaken: number | undefined,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 10,
    deaths: 5,
    assists: 3,
    damage_dealt: damageDealt,
    damage_taken: damageTaken,
  } as SquadPerformanceSeriesPoint
}

const ROWS: Record<string, SquadPerformanceSeriesPoint[]> = {
  Me: [pt(0, 900, 450), pt(1, 450, 900)],
  F1: [pt(0, 675, 225), pt(1, 900, 675)],
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
      },
    ],
  })
}

afterEach(() => {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  vi.restoreAllMocks()
})

describe('SquadNetLivesChart', () => {
  it('capability damage_taken présente → chart rendu', async () => {
    setTitleCaps(['damage_taken'])
    render(
      <SquadNetLivesChart rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('capability damage_taken absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <SquadNetLivesChart rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})
