/**
 * SquadSynergiesPage.test.tsx — 2 empty states + rendu sans erreur avec données.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as squadContextModule from './SquadContext'
import type { TeammateRow } from '@/lib/api/types'
import { SquadSynergiesPage } from './SquadSynergiesPage'

const ROW = (gamertag: string): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 5,
  last_seen_at: undefined,
  with_kpis: {
    match_count: 5,
    wins: 3,
    kd_ratio: 1.5,
    win_rate: 0.6,
    accuracy: 0.45,
    kills_per_game: 12,
    assists_per_game: 4,
    headshot_kills_per_game: 3,
    perfect_kills_per_game: 1,
  },
  without_kpis: undefined,
})

function mockSquadContext(opts: {
  selectedRows: TeammateRow[]
  confirmedGamertags: string[]
}) {
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: opts.selectedRows,
    confirmedGamertags: opts.confirmedGamertags,
    pageData: null,
    playerSlug: 'test',
  })
}

function setTitleCaps(caps: string[]) {
  useAppShellStore.setState({
    currentTitleSlug: 'test_title',
    availableTitles: [
      { slug: 'test_title', name: 'Test', status: 'active', capabilities: caps, is_default: true, effective_hp_to_kill: 225 },
    ],
  })
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
  useAppShellStore.setState({ currentTitleSlug: 'halo_infinite', availableTitles: [] })
})

describe('SquadSynergiesPage — empty states', () => {
  it('no_selection : wording analyse, pas de contenu', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: [] })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Choisis 1 à 3 coéquipiers/)).toBeInTheDocument()
  })

  it('invalid_selection : message dédié', () => {
    mockSquadContext({ selectedRows: [], confirmedGamertags: ['ghost'] })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText(/Aucune donnée commune/)).toBeInTheDocument()
  })

  it('avec rows : rend sans erreur', () => {
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    expect(() => renderWithProviders(<SquadSynergiesPage />)).not.toThrow()
  })

  it('capability expected_stats présente → « Écart cumulé au FDA attendu » à gauche de Répartition', () => {
    setTitleCaps(['expected_stats'])
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.getByText('Écart cumulé au FDA attendu')).toBeInTheDocument()
  })

  it('capability expected_stats absente (Halo 5) → card Écart FDA masqué', () => {
    setTitleCaps(['ranked'])
    mockSquadContext({
      selectedRows: [ROW('A'), ROW('B')],
      confirmedGamertags: ['A', 'B'],
    })
    renderWithProviders(<SquadSynergiesPage />)
    expect(screen.queryByText('Écart cumulé au FDA attendu')).toBeNull()
  })
})
