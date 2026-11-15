/**
 * Ordre des lignes de CareerEncountersSection après chaque clic d'en-tête : figé avant et après le
 * passage du tri sur TanStack Table (étape 7.7).
 */
import { describe, expect, it, vi } from 'vitest'

import type { EncounterDTO } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { renderWithProviders } from '@/test/render-utils'
import { CareerEncountersSection } from './CareerEncountersSection'

function enc(gamertag: string, count: number, team: number, enemy: number, kda: number | null): EncounterDTO {
  return { xuid: gamertag, gamertag, match_count: count, as_teammate: team, as_enemy: enemy, avg_kda: kda } as EncounterDTO
}

const DATA = {
  teammates: [enc('delta', 5, 5, 0, 1.2), enc('Alpha', 9, 4, 5, null), enc('charlie', 5, 0, 5, 1.2)],
  enemies: [enc('bravo', 12, 2, 10, 0.4), enc('Echo', 9, 1, 8, 2.5)],
  total: 5,
}

vi.mock('./queries', () => ({
  useCareerEncounters: () => ({ data: DATA, isLoading: false }),
}))

describe('CareerEncountersSection — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = renderWithProviders(<CareerEncountersSection playerSlug="p" />)
    const recording = recordSortOrders(container, [
      { name: 'Joueur' },
      { name: /total_matches_played|Matchs/ },
      { name: 'En équipe' },
      { name: 'Adverse' },
      { name: /^kda$|FDA|KDA/ },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "/^kda$|FDA|KDA/": "1: Echo > delta > charlie > bravo > Alpha | 2: bravo > delta > charlie > Echo > Alpha | 3: Echo > delta > charlie > bravo > Alpha | aria-sort: none,none,none,none,descending",
        "/total_matches_played|Matchs/": "1: bravo > Alpha > Echo > delta > charlie | 2: delta > charlie > Alpha > Echo > bravo | 3: bravo > Alpha > Echo > delta > charlie | aria-sort: none,descending,none,none,none",
        "Adverse": "1: bravo > Echo > Alpha > charlie > delta | 2: delta > Alpha > charlie > Echo > bravo | 3: bravo > Echo > Alpha > charlie > delta | aria-sort: none,none,none,descending,none",
        "En équipe": "1: delta > Alpha > bravo > Echo > charlie | 2: charlie > Echo > bravo > Alpha > delta | 3: delta > Alpha > bravo > Echo > charlie | aria-sort: none,none,descending,none,none",
        "Joueur": "1: Alpha > bravo > charlie > delta > Echo | 2: Echo > delta > charlie > bravo > Alpha | 3: Alpha > bravo > charlie > delta > Echo | aria-sort: ascending,none,none,none,none",
        "initial": "delta > Alpha > charlie > bravo > Echo",
      }
    `)
  })
})
