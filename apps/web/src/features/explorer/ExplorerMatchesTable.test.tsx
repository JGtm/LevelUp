/**
 * Tests ExplorerMatchesTable — couverture de la prop defaultPageSize + expander.
 *
 * Mode compact (defaultPageSize=10) :
 *  - 10 lignes affichées par défaut sur un sample de 15 rows
 *  - Bouton "Voir tout (20 par page)" présent
 *  - Click → 15 lignes (PAGE_SIZE=20 contient tout)
 *  - Label switch vers "Réduire (10 lignes)"
 *
 * Mode legacy (defaultPageSize undefined) :
 *  - PAGE_SIZE=20 dès le départ, pas de bouton expander
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

describe('ExplorerMatchesTable — defaultPageSize + expander', () => {
  it('compact mode (defaultPageSize=10) affiche 10 lignes + bouton "Voir tout"', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    // 10 rows visibles dans le tbody.
    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    expect(tbody?.querySelectorAll('tr').length).toBe(10)
    // Bouton "Voir tout" présent.
    const expander = screen.getByTestId('explorer-matches-table-expander')
    expect(expander).toBeInTheDocument()
    expect(expander).toHaveTextContent(/Voir tout/i)
  })

  it('click sur l\'expander passe à 15 lignes et label devient "Réduire"', () => {
    const rows = makeRows(15)
    renderWithProviders(
      <ExplorerMatchesTable
        rows={rows}
        playerSlug="me"
        defaultPageSize={10}
        alwaysShowPagination
      />,
    )
    const expander = screen.getByTestId('explorer-matches-table-expander')
    fireEvent.click(expander)

    const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
    // 15 < PAGE_SIZE=20 → tout est visible.
    expect(tbody?.querySelectorAll('tr').length).toBe(15)
    expect(expander).toHaveTextContent(/Réduire/i)
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
})
