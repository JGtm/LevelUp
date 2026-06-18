/**
 * SquadMatchHistoryTable.test.tsx — TanStack Table teammates.11.
 *
 * Couvre :
 *  - Rendu des colonnes + valeurs formatées (date, outcome, K/D/A, perf, MMR).
 *  - Pagination 20/page (boutons prev/next, indicateur, masquée si <=20).
 *  - Empty rendering (rows vides → null).
 *  - Navigation au clic ligne.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import * as useFieldMappingsModule from '@/lib/i18n/fieldMappings'
import * as router from '@tanstack/react-router'
import type { SquadMatchHistoryRow } from '@/lib/api/types'
import { SquadMatchHistoryTable } from './SquadMatchHistoryTable'

const navigateMock = vi.fn()

vi.mock('@tanstack/react-router', async () => {
  const actual = await vi.importActual<typeof router>('@tanstack/react-router')
  return { ...actual, useNavigate: () => navigateMock }
})

function row(idx: number, overrides: Partial<SquadMatchHistoryRow> = {}): SquadMatchHistoryRow {
  return {
    match_id: `match-${idx}`,
    start_time: `2026-04-${String(30 - (idx % 30)).padStart(2, '0')}T15:30:00Z`,
    map_ui: 'Aquarius',
    playlist_name: 'Ranked Arena',
    pair_name: 'Slayer',
    mode_ui: 'Assassin',
    outcome: 2,
    kills: 18,
    deaths: 7,
    assists: 4,
    accuracy: 45.0, // pourcentage 0..100 (match_participants), affiché tel quel
    performance_score: 72.3,
    team_mmr_avg: 1530,
    session_label: 'Session 04-30',
    ...overrides,
  }
}

const MAPPINGS: useFieldMappingsModule.FieldMappingsResponse = {
  title_slug: 'halo_infinite',
  schema_version: 1,
  locale: 'fr',
  fields: {},
  assets: {
    map: {
      Aquarius: { label: 'Aquarius FR', kind: 'map', storage_value: 'Aquarius' } as never,
    } as never,
    playlist: {
      'Ranked Arena': { label: 'Arène classée', kind: 'playlist', storage_value: 'Ranked Arena' } as never,
    } as never,
  } as never,
}

function mockMappings(value: useFieldMappingsModule.FieldMappingsResponse | undefined) {
  vi.spyOn(useFieldMappingsModule, 'useFieldMappings').mockReturnValue({
    data: value,
    isLoading: false,
    isError: false,
    error: null,
    isSuccess: !!value,
    isPending: !value,
    isFetching: false,
    isStale: false,
    refetch: vi.fn(),
  } as unknown as ReturnType<typeof useFieldMappingsModule.useFieldMappings>)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  navigateMock.mockClear()
  mockMappings(MAPPINGS)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('SquadMatchHistoryTable', () => {
  it('rows vides → ne rend rien (null)', () => {
    const { container } = renderWithProviders(
      <SquadMatchHistoryTable rows={[]} playerSlug="me" />,
    )
    expect(container.querySelector('[data-testid="squad-match-history-table"]')).toBeNull()
  })

  it('rend le tableau avec entêtes localisées + valeurs formatées', () => {
    renderWithProviders(<SquadMatchHistoryTable rows={[row(1)]} playerSlug="me" />)
    const table = screen.getByTestId('squad-match-history-table')
    expect(within(table).getByText('Date')).toBeInTheDocument()
    expect(within(table).getByText('Carte')).toBeInTheDocument()
    expect(within(table).getByText('K/D/A')).toBeInTheDocument()
    expect(within(table).getByText('Aquarius FR')).toBeInTheDocument()
    expect(within(table).getByText('Arène classée')).toBeInTheDocument()
    expect(within(table).getByText('Victoire')).toBeInTheDocument()
    expect(within(table).getByText('18/7/4')).toBeInTheDocument()
    expect(within(table).getByText('45.0%')).toBeInTheDocument()
    expect(within(table).getByText('72.3')).toBeInTheDocument()
    expect(within(table).getByText('1530')).toBeInTheDocument()
    // Colonne Mode : affiche mode_ui (résolu), pas pair_name brut.
    expect(within(table).getByText('Assassin')).toBeInTheDocument()
  })

  it('colonne Mode : fallback sur pair_name si mode_ui absent', () => {
    renderWithProviders(
      <SquadMatchHistoryTable rows={[row(1, { mode_ui: undefined })]} playerSlug="me" />,
    )
    const table = screen.getByTestId('squad-match-history-table')
    expect(within(table).getByText('Slayer')).toBeInTheDocument()
  })

  it('outcome=3 (loss) affiche libellé Défaite', () => {
    renderWithProviders(
      <SquadMatchHistoryTable rows={[row(1, { outcome: 3 })]} playerSlug="me" />,
    )
    expect(screen.getByText('Défaite')).toBeInTheDocument()
  })

  it('valeurs nulles → "-" (accuracy + perf + session_label)', () => {
    const r = row(1, { accuracy: undefined, performance_score: undefined, session_label: undefined })
    renderWithProviders(<SquadMatchHistoryTable rows={[r]} playerSlug="me" />)
    const table = screen.getByTestId('squad-match-history-table')
    const dashCells = within(table).getAllByText('-')
    expect(dashCells.length).toBeGreaterThanOrEqual(3)
  })

  it('≤20 rows → pas de pagination affichée', () => {
    const rows = Array.from({ length: 15 }, (_, i) => row(i))
    renderWithProviders(<SquadMatchHistoryTable rows={rows} playerSlug="me" />)
    expect(screen.queryByText(/Page 1/)).toBeNull()
    expect(screen.queryByText(/Précédent/)).toBeNull()
  })

  it('>20 rows → pagination affichée + navigation prev/next', () => {
    const rows = Array.from({ length: 47 }, (_, i) => row(i))
    renderWithProviders(<SquadMatchHistoryTable rows={rows} playerSlug="me" />)
    expect(screen.getByText('Page 1 / 3')).toBeInTheDocument()
    expect(screen.getByText('47 matchs')).toBeInTheDocument()

    const next = screen.getByRole('button', { name: /Suivant/ })
    fireEvent.click(next)
    expect(screen.getByText('Page 2 / 3')).toBeInTheDocument()

    const prev = screen.getByRole('button', { name: /Précédent/ })
    fireEvent.click(prev)
    expect(screen.getByText('Page 1 / 3')).toBeInTheDocument()
  })

  it('clic sur une ligne → navigate vers /players/$playerSlug/matches/$matchId', () => {
    // Phase 2a : navigate est désormais appelé avec un `state` qui pousse
    // le MatchNavContext (source: 'session', matchIds: [...]) dans le router.
    renderWithProviders(<SquadMatchHistoryTable rows={[row(1)]} playerSlug="me" />)
    const tr = screen.getByTestId('squad-match-history-table').querySelector('tbody tr')
    if (!tr) throw new Error('row not found')
    fireEvent.click(tr)
    expect(navigateMock).toHaveBeenCalledWith(
      expect.objectContaining({
        to: '/players/$playerSlug/matches/$matchId',
        params: { playerSlug: 'me', matchId: 'match-1' },
        state: expect.any(Function),
      }),
    )
  })

  it('21 rows → exactement 20 sur page 1, 1 sur page 2', () => {
    const rows = Array.from({ length: 21 }, (_, i) => row(i))
    renderWithProviders(<SquadMatchHistoryTable rows={rows} playerSlug="me" />)
    const tableEl = screen.getByTestId('squad-match-history-table')
    expect(tableEl.querySelectorAll('tbody tr')).toHaveLength(20)

    fireEvent.click(screen.getByRole('button', { name: /Suivant/ }))
    const tableEl2 = screen.getByTestId('squad-match-history-table')
    expect(tableEl2.querySelectorAll('tbody tr')).toHaveLength(1)
  })
})
