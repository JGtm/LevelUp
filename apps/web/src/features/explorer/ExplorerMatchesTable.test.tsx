/**
 * Tests ExplorerMatchesTable — pagination par defaultPageSize (mode Joueur).
 *
 * Mode compact (defaultPageSize=10) : 10 lignes/page + navigation par page,
 * SANS expander (retiré — redondant avec la pagination, cf. retour user).
 * Mode legacy (defaultPageSize undefined) : PAGE_SIZE=20 par page.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchRow } from '@/lib/api/types'

import { ExplorerMatchesTable } from './ExplorerMatchesTable'

function makeRow(i: number): ExplorerMatchRow {
  return {
    match_id: `match-${i}`,
    start_time: '2026-05-26T12:00:00Z',
    start_time_label: `2026-05-26 12:0${i}`,
    map_ui: `Map${i}`,
    mode_ui: 'Slayer',
    playlist_label: 'Quick Play',
    outcome_label: 'Victoire',
    outcome_code: 2,
    score_label: '50-30',
    is_with_friends: false,
    experience_type_label: 'PvP',
    kills: 10,
    deaths: 5,
    assists: 3,
    kda: 2.1,
  }
}

function makeRows(n: number): ExplorerMatchRow[] {
  return Array.from({ length: n }, (_, i) => makeRow(i + 1))
}

describe('ExplorerMatchesTable — pagination (defaultPageSize)', () => {
  it('compact mode (defaultPageSize=10) affiche 10 lignes/page, sans expander', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    // 10 rows visibles dans le tbody (page 1).
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    expect(tbody?.querySelectorAll('tr').length).toBe(10)
    // Expander retiré ; pagination présente (2 pages pour 15 lignes).
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
    expect(screen.getByText(/Page 1 \/ 2/)).toBeInTheDocument()
  })

  it('pagination : "Suivant" affiche les lignes restantes (page 2)', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    fireEvent.click(screen.getByText('Suivant'))
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    // Page 2 = 5 lignes restantes (15 - 10).
    expect(tbody?.querySelectorAll('tr').length).toBe(5)
  })

  it('legacy mode (defaultPageSize undefined) affiche 15 lignes sans expander', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable rows={rows} playerSlug="me" alwaysShowPagination />,
    )
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    // PAGE_SIZE=20 ≥ 15 → tout est visible.
    expect(tbody?.querySelectorAll('tr').length).toBe(15)
    // Pas de bouton expander.
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
  })

  it('compact mode avec ≤ defaultPageSize lignes : pas d\'expander', () => {
    const rows = makeRows(8)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    // Pas d'expander (rows.length=8 ≤ defaultPageSize=10).
    expect(screen.queryByTestId('explorer-matches-table-expander')).not.toBeInTheDocument()
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    expect(tbody?.querySelectorAll('tr').length).toBe(8)
  })

  it('sans extraColumns → aucune colonne « Δ rang » (réservée à la vue session)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(2)} playerSlug="me" />)
    expect(screen.getByTestId('explorer-matches-table')).toBeInTheDocument()
    expect(screen.queryByText('Δ rang')).not.toBeInTheDocument()
    expect(screen.queryByText('Δ rank')).not.toBeInTheDocument()
  })
})
