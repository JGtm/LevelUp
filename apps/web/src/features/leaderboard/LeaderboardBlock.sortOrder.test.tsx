/**
 * Ordre des lignes du Classement CSR mondial après chaque clic d'en-tête : figé avant et après le
 * passage du tri sur TanStack Table (étape 7.7). Les égalités (Frags, Morts) et le tri initial par
 * rang font partie de ce qui est figé.
 */
import { waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it } from 'vitest'

import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { LeaderboardBlock } from './LeaderboardBlock'

function row(rank: number, gamertag: string, over: Record<string, unknown>) {
  return {
    rank,
    xuid: `x${rank}`,
    gamertag,
    csr_value: 1900 - rank * 10,
    tier: 'Onyx',
    sub_tier: 0,
    is_local: false,
    match_count: 10,
    kills: 100,
    deaths: 50,
    assists: 20,
    win_rate: 0.5,
    kda: 10,
    accuracy: 400,
    ...over,
  }
}

const ENTRIES = [
  row(1, 'Ace', { csr_value: 1800, kills: 300, deaths: 90, assists: 10, win_rate: 0.7, kda: 20, accuracy: 600, match_count: 12 }),
  row(2, 'Bob', { csr_value: 1900, kills: 100, deaths: 90, assists: 30, win_rate: 0.5, kda: 8, accuracy: 300, match_count: 30 }),
  row(3, 'Cid', { csr_value: 1700, kills: 100, deaths: 40, assists: 30, win_rate: 0.5, kda: 8, accuracy: 300, match_count: 30 }),
  row(4, 'Dan', { csr_value: 1700, kills: 250, deaths: 40, assists: 5, win_rate: 0.9, kda: 40, accuracy: 700, match_count: 5 }),
  row(5, 'Eve', { csr_value: 1600, kills: 100, deaths: 60, assists: 30, win_rate: 0.1, kda: 2, accuracy: 100, match_count: 8 }),
]

describe('LeaderboardBlock — ordre des lignes par en-tête', () => {
  it('ordre initial (rang), puis trois clics par en-tête', async () => {
    server.use(
      http.get('/api/v1/players/:playerSlug/pages/leaderboard', () =>
        HttpResponse.json({ entries: ENTRIES, category: 'csr-world', season_id: 'csrseason13-2', playlist_id: 'p', title_slug: 'halo_infinite', total: ENTRIES.length }),
      ),
    )
    const { container, findByText } = renderWithProviders(<LeaderboardBlock playerSlug="p" />)
    await findByText('Ace')
    await waitFor(() => expect(container.querySelectorAll('tbody tr').length).toBe(5))
    const recording = recordSortOrders(
      container,
      [
        { name: '#' },
        { name: 'CSR' },
        { name: 'FDA' },
        { name: 'Frags' },
        { name: 'Morts' },
        { name: 'Assistances' },
        { name: 'Victoires' },
        { name: 'Matchs' },
        { name: 'Précision' },
      ],
      1,
    )
    expect(recording).toMatchInlineSnapshot(`
      {
        "#": "1: Eve > Dan > Cid > Bob > Ace | 2: Ace > Bob > Cid > Dan > Eve | 3: Eve > Dan > Cid > Bob > Ace | aria-sort: descending,-,-,none,none,none,none,none,none,none,none,none,none,-",
        "Assistances": "1: Eve > Cid > Bob > Ace > Dan | 2: Dan > Ace > Bob > Cid > Eve | 3: Eve > Cid > Bob > Ace > Dan | aria-sort: none,-,-,none,none,none,none,descending,none,none,none,none,none,-",
        "CSR": "1: Bob > Ace > Dan > Cid > Eve | 2: Eve > Cid > Dan > Ace > Bob | 3: Bob > Ace > Dan > Cid > Eve | aria-sort: none,-,-,descending,none,none,none,none,none,none,none,none,none,-",
        "FDA": "1: Dan > Ace > Cid > Bob > Eve | 2: Eve > Bob > Cid > Ace > Dan | 3: Dan > Ace > Cid > Bob > Eve | aria-sort: none,-,-,none,descending,none,none,none,none,none,none,none,none,-",
        "Frags": "1: Ace > Dan > Eve > Cid > Bob | 2: Bob > Cid > Eve > Dan > Ace | 3: Ace > Dan > Eve > Cid > Bob | aria-sort: none,-,-,none,none,descending,none,none,none,none,none,none,none,-",
        "Matchs": "1: Cid > Bob > Ace > Eve > Dan | 2: Dan > Eve > Ace > Bob > Cid | 3: Cid > Bob > Ace > Eve > Dan | aria-sort: none,-,-,none,none,none,none,none,none,descending,none,none,none,-",
        "Morts": "1: Bob > Ace > Eve > Dan > Cid | 2: Cid > Dan > Eve > Ace > Bob | 3: Bob > Ace > Eve > Dan > Cid | aria-sort: none,-,-,none,none,none,descending,none,none,none,none,none,none,-",
        "Précision": "1: Dan > Ace > Eve > Cid > Bob | 2: Bob > Cid > Eve > Ace > Dan | 3: Dan > Ace > Eve > Cid > Bob | aria-sort: none,-,-,none,none,none,none,none,none,none,descending,none,none,-",
        "Victoires": "1: Dan > Ace > Cid > Bob > Eve | 2: Eve > Bob > Cid > Ace > Dan | 3: Dan > Ace > Cid > Bob > Eve | aria-sort: none,-,-,none,none,none,none,none,descending,none,none,none,none,-",
        "initial": "Ace > Bob > Cid > Dan > Eve",
      }
    `)
  })
})
