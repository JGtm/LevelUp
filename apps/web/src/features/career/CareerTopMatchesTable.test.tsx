/**
 * CareerTopMatchesTable.test.tsx — colonne « Ouvrir sur Halo Waypoint » (I19),
 * insérée en 2e colonne (juste après le rang #).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { TopMatchDTO } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { CareerTopMatchesTable } from './CareerTopMatchesTable'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player' }),
  }
})

function makeItem(overrides: Partial<TopMatchDTO> = {}): TopMatchDTO {
  return {
    match_id: 'm-best-1',
    start_time: '2026-06-01T20:00:00Z',
    performance_score: 92.5,
    map_ui: 'Aquarius',
    mode_ui: 'Slayer',
    outcome_code: 2,
    outcome_label: 'Victoire',
    kills: 25,
    deaths: 8,
    kda: 3.1,
    ...overrides,
  }
}

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
      },
    ],
  })
}

function resetTitleAndWaypointPref() {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  useSettingsDraftStore.setState((s) => ({
    localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: true },
  }))
}

beforeEach(resetTitleAndWaypointPref)
afterEach(resetTitleAndWaypointPref)

const WAYPOINT_LABEL = 'Ouvrir sur Halo Waypoint'

describe('CareerTopMatchesTable — colonne « Ouvrir sur Halo Waypoint » (I19)', () => {
  it('visible par défaut, en 2e colonne (juste après #), avec l\'URL attendue', () => {
    renderWithProviders(
      <CareerTopMatchesTable items={[makeItem()]} playerSlug="Chocoboflor" />,
    )
    const link = screen.getByRole('link', { name: WAYPOINT_LABEL })
    expect(link).toHaveAttribute(
      'href',
      'https://www.halowaypoint.com/halo-infinite/players/Chocoboflor/matches/m-best-1',
    )
    // 2e colonne : juste après le "#" (idx 0 => "1"), avant la Date.
    const headerCells = screen.getAllByRole('columnheader')
    expect(headerCells[0]).toHaveTextContent('#')
    expect(headerCells[2]).toHaveTextContent('Date')
  })

  it('masquée quand le titre courant ne déclare pas waypoint_match_url', () => {
    setTitleCaps(['team_mmr'])
    renderWithProviders(<CareerTopMatchesTable items={[makeItem()]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })

  it('masquée quand la préférence locale showWaypointColumn est désactivée', () => {
    useSettingsDraftStore.setState((s) => ({
      localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: false },
    }))
    renderWithProviders(<CareerTopMatchesTable items={[makeItem()]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })
})
