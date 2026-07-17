import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { RelationInsight } from '@/lib/api/types'

import { getPalmaresText } from './i18n'
import { RelationsTable } from './RelationsTable'

const labels = getPalmaresText('fr').relations

// mk — fabrique une relation minimale (tous les champs requis du DTO) ; on ne
// surcharge que ce qui compte pour le tri sous test.
function mk(partial: Partial<RelationInsight> & { gamertag: string }): RelationInsight {
  return {
    xuid: partial.gamertag,
    total_matches: 10,
    teammate_matches: 5,
    teammate_wins: 3,
    teammate_win_rate: 0.6,
    enemy_matches: 5,
    enemy_wins: 2,
    enemy_win_rate: 0.4,
    avg_kda_with: 1,
    avg_kda_against: 1,
    kills_dealt: 10,
    deaths_suffered: 10,
    duel_ratio: 1,
    first_seen_at: '2026-01-01T00:00:00Z',
    last_seen_at: '2026-06-01T00:00:00Z',
    category: 'mixed',
    is_core: false,
    is_revived: false,
    badges: [],
    ...partial,
  }
}

// Jeu de 4 lignes dont une à ratio null (relève le cas « null en dernier »).
const rows: RelationInsight[] = [
  mk({ gamertag: 'Bravo', duel_ratio: 1.0 }),
  mk({ gamertag: 'Alpha', duel_ratio: 2.5 }),
  mk({ gamertag: 'Delta', duel_ratio: null }),
  mk({ gamertag: 'Charlie', duel_ratio: 0.5 }),
]

function renderTable() {
  return render(
    <RelationsTable
      rows={rows}
      labels={labels}
      locale="fr"
      onPlayerClick={vi.fn()}
      emptyMessage="vide"
    />,
  )
}

// rowGamertags — gamertags dans l'ordre d'affichage du tbody (1re cellule).
function rowGamertags(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('tbody tr')).map(
    (tr) => tr.querySelector('td button')?.textContent ?? '',
  )
}

describe('RelationsTable — tri client', () => {
  it("clic sur « Ratio » trie en décroissant, null en dernier (A5a)", () => {
    const { container } = renderTable()
    // Ordre serveur initial préservé (aucun tri).
    expect(rowGamertags(container)).toEqual(['Bravo', 'Alpha', 'Delta', 'Charlie'])

    fireEvent.click(screen.getByRole('button', { name: 'Ratio' }))
    // Colonnes numériques : premier clic = décroissant (2.5, 1.0, 0.5, null).
    expect(rowGamertags(container)).toEqual(['Alpha', 'Bravo', 'Charlie', 'Delta'])
  })

  it('second clic inverse le tri, null toujours en dernier (A5b)', () => {
    const { container } = renderTable()
    const ratioHeader = screen.getByRole('button', { name: 'Ratio' })
    fireEvent.click(ratioHeader)
    fireEvent.click(ratioHeader)
    // Croissant (0.5, 1.0, 2.5) et le null (Delta) reste relégué en dernier.
    expect(rowGamertags(container)).toEqual(['Charlie', 'Bravo', 'Alpha', 'Delta'])
  })

  it("pose aria-sort sur l'en-tête actif (A5c)", () => {
    renderTable()
    const ratioCol = screen.getByRole('columnheader', { name: 'Ratio' })
    // Avant clic : aucun tri actif → 'none'.
    expect(ratioCol).toHaveAttribute('aria-sort', 'none')

    fireEvent.click(within(ratioCol).getByRole('button', { name: 'Ratio' }))
    expect(ratioCol).toHaveAttribute('aria-sort', 'descending')
  })

  it("tri alpha du joueur insensible à la casse", () => {
    const mixed: RelationInsight[] = [
      mk({ gamertag: 'zeta' }),
      mk({ gamertag: 'Alpha' }),
      mk({ gamertag: 'beta' }),
    ]
    const { container } = render(
      <RelationsTable rows={mixed} labels={labels} locale="fr" onPlayerClick={vi.fn()} emptyMessage="vide" />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Joueur' }))
    // Alpha, beta, zeta — insensible à la casse (asc au premier clic pour l'alpha).
    expect(rowGamertags(container)).toEqual(['Alpha', 'beta', 'zeta'])
  })

  it('un changement de tri ramène en page 1 (A4)', () => {
    // > 25 lignes (RELATIONS_PAGE_SIZE) pour forcer la pagination.
    const many: RelationInsight[] = Array.from({ length: 30 }, (_, i) =>
      mk({ gamertag: `P${String(i).padStart(2, '0')}`, total_matches: i }),
    )
    render(
      <RelationsTable rows={many} labels={labels} locale="fr" onPlayerClick={vi.fn()} emptyMessage="vide" />,
    )
    // Aller en page 2.
    fireEvent.click(screen.getByRole('button', { name: 'Suivant' }))
    expect(screen.getByText('2 / 2')).toBeInTheDocument()
    // Un clic de tri ramène en page 1 (reset piloté par onSortingChange).
    fireEvent.click(screen.getByRole('button', { name: 'Rencontres' }))
    expect(screen.getByText('1 / 2')).toBeInTheDocument()
  })

  it("la colonne « Lien » n'est pas triable (A1)", () => {
    renderTable()
    const linkCol = screen.getByRole('columnheader', { name: 'Lien' })
    expect(linkCol).not.toHaveAttribute('aria-sort')
    expect(within(linkCol).queryByRole('button')).toBeNull()
  })
})
