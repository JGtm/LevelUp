/**
 * WatcherSection.test.tsx — tri CLIENT par clic sur les en-têtes (I16), table
 * des joueurs surveillés (WatcherPlayersTable).
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { WatcherPlayerStatus } from '@/lib/api/types'
import { WatcherSection } from './WatcherSection'

function makePlayer(overrides: Partial<WatcherPlayerStatus>): WatcherPlayerStatus {
  return {
    gamertag: 'Player',
    xuid: 'x',
    state: 'Watching',
    in_game: false,
    state_since: '2026-07-24T00:00:00Z',
    state_duration: '1h',
    ...overrides,
  }
}

const dataRef: { current: unknown } = { current: undefined }

vi.mock('@/features/settings/watcher-queries', () => ({
  useWatcherStatus: () => ({ data: dataRef.current, isError: false }),
}))

/** Ordre des lignes du tbody, identifiées par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('WatcherSection — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function setup() {
    dataRef.current = {
      daemon_running: true,
      rta_connected: true,
      token_valid: true,
      players: [
        makePlayer({ xuid: 'x1', gamertag: 'Charlie' }),
        makePlayer({ xuid: 'x2', gamertag: 'Alpha' }),
        makePlayer({ xuid: 'x3', gamertag: 'Bravo' }),
      ],
    }
  }

  it('sans clic : ordre serveur conservé', () => {
    setup()
    render(<WatcherSection />)
    expect(rowOrder(names)).toEqual(['Charlie', 'Alpha', 'Bravo'])
  })

  it('clic sur « Joueur » trie alphabétiquement (asc au 1er clic)', () => {
    setup()
    render(<WatcherSection />)
    const header = screen.getAllByText('Joueur')[0]
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })
})
