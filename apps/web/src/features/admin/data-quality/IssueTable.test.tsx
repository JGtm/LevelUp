/**
 * IssueTable.test.tsx — C7(b) : la colonne « Exemples » rend jusqu'à 3 match_id
 * cliquables (nouvel onglet) vers la vue de match player-scopée, dégrade en texte
 * simple sans joueur cible, et reste absente sans matchLinkTitleSlug.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { AdminDataQualityIssue, PlayerSummary } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { IssueTable, type IssueColumn } from './IssueTable'

const columns: IssueColumn[] = [{ header: 'ID', cell: (i) => i.id }]

function player(slug: string): PlayerSummary {
  return {
    gamertag: slug,
    is_demo: false,
    player_slug: slug,
    sync_enabled: true,
    waypoint_player: slug,
    xuid: '1',
  }
}

function issue(exampleMatchIDs: string[]): AdminDataQualityIssue {
  return {
    kind: 'raw_uuid',
    id: 'pl-x',
    occurrences: exampleMatchIDs.length,
    example_match_ids: exampleMatchIDs,
  }
}

afterEach(() => {
  useAppShellStore.setState({ currentPlayer: null, availablePlayers: [], locale: 'fr' })
})

describe('IssueTable — colonne Exemples (C7b)', () => {
  it('rend jusqu\'à 3 liens de match cliquables (nouvel onglet, href player-scopé)', () => {
    useAppShellStore.setState({ currentPlayer: player('JGtm'), availablePlayers: [], locale: 'fr' })
    render(
      <IssueTable
        issues={[issue(['m-1', 'm-2', 'm-3', 'm-4'])]}
        columns={columns}
        matchLinkTitleSlug="halo_5"
      />,
    )
    const links = screen.getAllByRole('link')
    expect(links).toHaveLength(3)
    expect(links[0]).toHaveAttribute('href', '/t/halo_5/players/JGtm/matches/m-1')
    expect(links[0]).toHaveAttribute('target', '_blank')
    expect(links[0]).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('utilise le premier profil disponible si aucun joueur courant', () => {
    useAppShellStore.setState({ currentPlayer: null, availablePlayers: [player('Alt')], locale: 'fr' })
    render(<IssueTable issues={[issue(['m-9'])]} columns={columns} matchLinkTitleSlug="halo_5" />)
    expect(screen.getByRole('link')).toHaveAttribute('href', '/t/halo_5/players/Alt/matches/m-9')
  })

  it('dégrade en texte simple (non cliquable) sans joueur cible', () => {
    useAppShellStore.setState({ currentPlayer: null, availablePlayers: [], locale: 'fr' })
    render(<IssueTable issues={[issue(['m-1'])]} columns={columns} matchLinkTitleSlug="halo_5" />)
    expect(screen.queryByRole('link')).toBeNull()
  })

  it('omet la colonne Exemples sans matchLinkTitleSlug', () => {
    useAppShellStore.setState({ currentPlayer: player('JGtm'), availablePlayers: [], locale: 'fr' })
    render(<IssueTable issues={[issue(['m-1'])]} columns={columns} />)
    expect(screen.queryByRole('link')).toBeNull()
  })
})

/** Ordre des lignes du tbody, identifiées par l'id qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

function issueWithId(id: string, occurrences: number): AdminDataQualityIssue {
  return { kind: 'raw_uuid', id, occurrences }
}

describe('IssueTable — tri CLIENT par en-têtes (I16)', () => {
  const sortableColumns: IssueColumn[] = [{ header: 'ID', cell: (i) => i.id, sortValue: (i) => i.id }]
  const names = ['alpha', 'bravo', 'charlie']

  it('sans `sortable` : ordre serveur conservé, en-têtes non triables', () => {
    render(
      <IssueTable
        issues={[issueWithId('charlie', 1), issueWithId('alpha', 2), issueWithId('bravo', 3)]}
        columns={sortableColumns}
      />,
    )
    expect(rowOrder(names)).toEqual(['charlie', 'alpha', 'bravo'])
    expect(screen.getByText('ID').closest('th')?.querySelector('button')).not.toBeInTheDocument()
  })

  it('avec `sortable` : clic sur la colonne ID trie alphabétiquement (asc au 1er clic)', () => {
    render(
      <IssueTable
        issues={[issueWithId('charlie', 1), issueWithId('alpha', 2), issueWithId('bravo', 3)]}
        columns={sortableColumns}
        sortable
      />,
    )
    const header = screen.getByText('ID')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['alpha', 'bravo', 'charlie'])
  })

  it('avec `sortable` : clic sur « Matchs » (occurrences) trie numériquement (desc au 1er clic)', () => {
    render(
      <IssueTable
        issues={[issueWithId('charlie', 1), issueWithId('alpha', 2), issueWithId('bravo', 3)]}
        columns={sortableColumns}
        sortable
      />,
    )
    const header = screen.getByText('Matchs')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['bravo', 'alpha', 'charlie'])
  })

  it('`sortable` + `pagination` : le tri reste désactivé (garde défensive)', () => {
    render(
      <IssueTable
        issues={[issueWithId('charlie', 1), issueWithId('alpha', 2), issueWithId('bravo', 3)]}
        columns={sortableColumns}
        sortable
        pagination={{ pageIndex: 0, pageSize: 25, total: 3, onPageChange: () => {} }}
      />,
    )
    expect(rowOrder(names)).toEqual(['charlie', 'alpha', 'bravo'])
    expect(screen.getByText('ID').closest('th')?.querySelector('button')).not.toBeInTheDocument()
  })
})
