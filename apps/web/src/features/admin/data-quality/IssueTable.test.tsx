/**
 * IssueTable.test.tsx — C7(b) : la colonne « Exemples » rend jusqu'à 3 match_id
 * cliquables (nouvel onglet) vers la vue de match player-scopée, dégrade en texte
 * simple sans joueur cible, et reste absente sans matchLinkTitleSlug.
 */
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'

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
