/**
 * SquadFdaGapCumulativeCard.test.tsx — Lot C (D3/D4), forme revue par le lot L1
 * (PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
 *
 * Masquage par capability `expected_stats` (self-gate, retour null) ; rendu du
 * graphe SEUL (aucune pastille « écart moyen par match » sous le graphe) ; valeur de
 * fin au bout des courbes formatée dans la locale de l'interface (« +0,1 » en FR).
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import { SquadFdaGapCumulativeCard } from './SquadFdaGapCumulativeCard'
import { getSquadText } from './i18n'

const options: Array<{ series?: Array<{ endLabel?: { formatter: (p: { value?: unknown }) => string } }> }> =
  []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: (typeof options)[number] }) => {
    options.push(props.option)
    return <div data-testid="echarts-mock" />
  },
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
  Me: [pt(0, 1.6, 1.0), pt(1, 1.4, 1.0)], // cumul final +1,0
  F1: [pt(0, 0.5, 1.0), pt(1, 0.5, 1.0)], // cumul final -1,0
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
  options.length = 0
  vi.restoreAllMocks()
})

describe('SquadFdaGapCumulativeCard', () => {
  it('capability expected_stats présente → titre + graphe, aucune pastille sous le graphe', async () => {
    setTitleCaps(['expected_stats'])
    render(
      <SquadFdaGapCumulativeCard rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(await screen.findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.getByText(T.fdaGap.title)).toBeInTheDocument()
    // L1.2 : plus de rangée « écart moyen par match » ni de valeur « /match ».
    expect(screen.queryByTestId('fda-gap-kpis')).toBeNull()
    expect(screen.queryByText(/\/match/)).toBeNull()
    expect(screen.queryByText('Me')).toBeNull()
  })

  it('valeur de fin au bout des courbes, dans la locale de l’interface (FR)', async () => {
    setTitleCaps(['expected_stats'])
    render(
      <SquadFdaGapCumulativeCard rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    await screen.findByTestId('echarts-mock')
    const series = options.at(-1)?.series ?? []
    expect(series).toHaveLength(2)
    // Cumul final Me : (1,6−1) + (1,4−1) = +1,0 ; F1 : −0,5 − 0,5 = −1,0.
    expect(series[0].endLabel?.formatter({ value: 1.0 })).toBe('+1,0')
    expect(series[1].endLabel?.formatter({ value: -1.0 })).toMatch(/^[-−]1,0$/)
  })

  it('capability expected_stats absente → non rendu (null)', () => {
    setTitleCaps(['ranked'])
    const { container } = render(
      <SquadFdaGapCumulativeCard rowsByPlayer={ROWS} playerOrder={ORDER} colorByPlayer={COLORS} t={T} />,
    )
    expect(container).toBeEmptyDOMElement()
    expect(screen.queryByTestId('echarts-mock')).toBeNull()
  })
})
