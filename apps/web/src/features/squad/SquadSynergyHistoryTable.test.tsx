/**
 * SquadSynergyHistoryTable.test.tsx — non-régression colonnes + lien
 * « Ouvrir sur Halo Waypoint » (I19, remplace l'ancien texte « ↗ wp »).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SquadMatchHistoryRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

import { SquadSynergyHistoryTable } from './SquadSynergyHistoryTable'

const navigateMock = vi.fn()
vi.mock('@/lib/match-nav/useNavigateToMatch', () => ({
  useNavigateToMatch: () => navigateMock,
}))

function makeRow(overrides: Partial<SquadMatchHistoryRow> = {}): SquadMatchHistoryRow {
  return {
    match_id: 'match-1',
    start_time: '2026-05-26T12:00:00Z',
    map_ui: 'Live Fire',
    outcome: 2,
    kills: 10,
    deaths: 5,
    assists: 3,
    team_mmr_avg: 1500,
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

/** État par défaut déterministe (n'assume rien de la valeur hydratée depuis
 *  localStorage) : titre par défaut fail-open + préférence Waypoint ON. */
function resetTitleAndWaypointPref() {
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
  useSettingsDraftStore.setState((s) => ({
    localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: true },
  }))
}

beforeEach(resetTitleAndWaypointPref)
afterEach(() => {
  resetTitleAndWaypointPref()
  navigateMock.mockClear()
})

const WAYPOINT_LABEL = 'Ouvrir sur Halo Waypoint'

describe('SquadSynergyHistoryTable — non-régression', () => {
  it('rend les colonnes de base (date, carte, résultat)', () => {
    renderWithProviders(
      <SquadSynergyHistoryTable rows={[makeRow()]} playerSlug="Chocoboflor" />,
    )
    expect(screen.getByText('Live Fire')).toBeInTheDocument()
    expect(screen.getByText('Victoire')).toBeInTheDocument()
  })

  it('bloc vide : message noBlockData, pas de crash', () => {
    renderWithProviders(<SquadSynergyHistoryTable rows={[]} playerSlug="me" />)
    expect(screen.getByTestId('squad-synergy-history-table')).toBeInTheDocument()
  })
})

describe('SquadSynergyHistoryTable — lien « Ouvrir sur Halo Waypoint » (I19)', () => {
  it('visible par défaut (capability fail-open + préférence locale ON)', () => {
    renderWithProviders(
      <SquadSynergyHistoryTable rows={[makeRow()]} playerSlug="Chocoboflor" />,
    )
    const link = screen.getByRole('link', { name: WAYPOINT_LABEL })
    expect(link).toHaveAttribute(
      'href',
      'https://www.halowaypoint.com/halo-infinite/players/Chocoboflor/matches/match-1',
    )
  })

  it('masqué quand le titre courant ne déclare pas waypoint_match_url', () => {
    setTitleCaps(['team_mmr'])
    renderWithProviders(<SquadSynergyHistoryTable rows={[makeRow()]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })

  it('masqué quand la préférence locale showWaypointColumn est désactivée', () => {
    useSettingsDraftStore.setState((s) => ({
      localUiPrefs: { ...s.localUiPrefs, showWaypointColumn: false },
    }))
    renderWithProviders(<SquadSynergyHistoryTable rows={[makeRow()]} playerSlug="me" />)
    expect(screen.queryByRole('link', { name: WAYPOINT_LABEL })).not.toBeInTheDocument()
  })

  it('cliquer le lien Waypoint ne déclenche PAS la navigation interne de la ligne (stopPropagation)', () => {
    renderWithProviders(
      <SquadSynergyHistoryTable rows={[makeRow()]} playerSlug="Chocoboflor" />,
    )
    fireEvent.click(screen.getByRole('link', { name: WAYPOINT_LABEL }))
    expect(navigateMock).not.toHaveBeenCalled()
  })
})
