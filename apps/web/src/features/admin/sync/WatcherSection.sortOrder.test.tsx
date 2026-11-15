/**
 * Ordre des lignes de la table des joueurs surveillés (WatcherSection) après chaque clic d'en-tête :
 * figé avant et après le passage du tri sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { WatcherPlayerStatus } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { WatcherSection } from './WatcherSection'

function player(gamertag: string, over: Partial<WatcherPlayerStatus>): WatcherPlayerStatus {
  return { gamertag, xuid: gamertag, state: 'Watching', in_game: false, state_since: '2026-07-24T00:00:00Z', state_duration: '1h', ...over }
}

const DATA = {
  daemon_running: true,
  rta_connected: true,
  token_valid: true,
  players: [
    player('delta', { state: 'Syncing', presence_state: 'online', state_since: '2026-07-24T05:00:00Z', last_event_at: '2026-07-24T06:00:00Z' }),
    player('alpha', { state: 'Watching', state_since: '2026-07-24T01:00:00Z' }),
    player('Charlie', { state: 'Cooling', presence_state: 'away', state_since: '2026-07-24T05:00:00Z', last_event_at: '2026-07-24T07:00:00Z' }),
    player('bravo', { state: 'Watching', presence_state: 'online', state_since: '2026-07-24T03:00:00Z' }),
  ],
}

vi.mock('@/features/settings/watcher-queries', () => ({
  useWatcherStatus: () => ({ data: DATA, isError: false }),
}))

describe('WatcherSection — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<WatcherSection />)
    const recording = recordSortOrders(container, [
      { name: 'Joueur' },
      { name: 'État' },
      { name: 'Présence Xbox' },
      { name: 'Depuis' },
      { name: 'Dernier event' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Depuis": "1: delta > Charlie > bravo > alpha | 2: alpha > bravo > delta > Charlie | 3: delta > Charlie > bravo > alpha | aria-sort: none,none,none,descending,none",
        "Dernier event": "1: Charlie > delta > alpha > bravo | 2: delta > Charlie > alpha > bravo | 3: Charlie > delta > alpha > bravo | aria-sort: none,none,none,none,descending",
        "Joueur": "1: alpha > bravo > Charlie > delta | 2: delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > delta | aria-sort: ascending,none,none,none,none",
        "Présence Xbox": "1: Charlie > delta > bravo > alpha | 2: delta > bravo > Charlie > alpha | 3: Charlie > delta > bravo > alpha | aria-sort: none,none,ascending,none,none",
        "initial": "delta > alpha > Charlie > bravo",
        "État": "1: Charlie > delta > alpha > bravo | 2: alpha > bravo > delta > Charlie | 3: Charlie > delta > alpha > bravo | aria-sort: none,ascending,none,none,none",
      }
    `)
  })
})
