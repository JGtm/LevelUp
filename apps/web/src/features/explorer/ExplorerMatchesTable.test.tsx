/**
 * Tests ExplorerMatchesTable — pagination par defaultPageSize (mode Joueur).
 *
 * Mode compact (defaultPageSize=10) : 10 lignes/page + navigation par page,
 * SANS expander (retiré — redondant avec la pagination, cf. retour user).
 * Mode legacy (defaultPageSize undefined) : PAGE_SIZE=20 par page.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchRow } from '@/lib/api/types'

import { ExplorerMatchesTable } from './ExplorerMatchesTable'

function makeRow(i: number, overrides: Partial<ExplorerMatchRow> = {}): ExplorerMatchRow {
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
    ...overrides,
  }
}

function makeRows(n: number): ExplorerMatchRow[] {
  return Array.from({ length: n }, (_, i) => makeRow(i + 1))
}

/** Ordre des lignes du tbody, identifiées par le nom de carte qu'elles contiennent. */
function bodyMapOrder(names: string[]): string[] {
  const tbody = screen.getByTestId('explorer-matches-table').querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
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

describe('ExplorerMatchesTable — tri CLIENT par en-têtes (mode Matchs)', () => {
  it('sans `sortable` → en-têtes statiques (aucun bouton de tri, pas d’aria-sort)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(2)} playerSlug="me" />)
    const dateHeader = screen.getByRole('columnheader', { name: 'Date' })
    expect(within(dateHeader).queryByRole('button')).not.toBeInTheDocument()
    expect(dateHeader).not.toHaveAttribute('aria-sort')
  })

  it('avec `sortable` → Date triée descendante par défaut (aria-sort + ▼) ; colonnes non actives = aria-sort none', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(3)} playerSlug="me" sortable />)
    const dateBtn = screen.getByRole('button', { name: 'Trier par Date' })
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'descending')
    expect(within(dateBtn).getByText('▼')).toBeInTheDocument()
    // Résultat est désormais triable (tri client) mais inactif → aria-sort none.
    const outcomeBtn = screen.getByRole('button', { name: 'Trier par Résultat' })
    expect(outcomeBtn.closest('th')).toHaveAttribute('aria-sort', 'none')
  })

  it('colonne numérique (Frags) trie NUMÉRIQUEMENT, asc/desc (pas lexicographique)', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', kills: 2 }),
      makeRow(2, { map_ui: 'Bravo', kills: 30 }),
      makeRow(3, { map_ui: 'Charlie', kills: 10 }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    const btn = screen.getByRole('button', { name: 'Trier par Frags' })
    fireEvent.click(btn) // desc (numérique) → 30, 10, 2
    expect(bodyMapOrder(names)).toEqual(['Bravo', 'Charlie', 'Alpha'])
    fireEvent.click(btn) // asc → 2, 10, 30 (lexicographique donnerait 10,2,30)
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
  })

  it('colonne texte (Carte) trie ALPHABÉTIQUEMENT, ascendant au 1er clic', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Charlie' }),
      makeRow(2, { map_ui: 'Alpha' }),
      makeRow(3, { map_ui: 'Bravo' }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    fireEvent.click(screen.getByRole('button', { name: 'Trier par Carte' }))
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('colonne dérivée (Résultat) trie sur outcome_code BRUT, pas le libellé traduit', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', outcome_code: 3 }), // Défaite
      makeRow(2, { map_ui: 'Bravo', outcome_code: 1 }), // Égalité
      makeRow(3, { map_ui: 'Charlie', outcome_code: 2 }), // Victoire
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    fireEvent.click(screen.getByRole('button', { name: 'Trier par Résultat' }))
    // desc sur le code : 3, 2, 1 → Alpha, Charlie, Bravo (un tri alpha du libellé
    // donnerait Défaite < Égalité < Victoire = Alpha, Bravo, Charlie).
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
  })

  it('valeurs nulles rangées EN BAS dans les deux sens (Frags null)', () => {
    const names = ['Alpha', 'Bravo', 'Charlie']
    const rows = [
      makeRow(1, { map_ui: 'Alpha', kills: 10 }),
      makeRow(2, { map_ui: 'Bravo', kills: null }),
      makeRow(3, { map_ui: 'Charlie', kills: 5 }),
    ]
    renderWithProviders(<ExplorerMatchesTable rows={rows} playerSlug="me" sortable />)
    const btn = screen.getByRole('button', { name: 'Trier par Frags' })
    fireEvent.click(btn) // desc → 10, 5, null(bas)
    expect(bodyMapOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
    fireEvent.click(btn) // asc → 5, 10, null(toujours bas)
    expect(bodyMapOrder(names)).toEqual(['Charlie', 'Alpha', 'Bravo'])
  })

  it('bascule la direction sur la colonne active (▼ desc → ▲ asc + aria-sort)', () => {
    renderWithProviders(<ExplorerMatchesTable rows={makeRows(3)} playerSlug="me" sortable />)
    const dateBtn = screen.getByRole('button', { name: 'Trier par Date' })
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'descending')
    fireEvent.click(dateBtn)
    expect(dateBtn.closest('th')).toHaveAttribute('aria-sort', 'ascending')
    expect(within(dateBtn).getByText('▲')).toBeInTheDocument()
  })
})
