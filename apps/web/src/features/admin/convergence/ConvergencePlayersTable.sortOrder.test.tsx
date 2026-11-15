/**
 * Ordre des lignes de ConvergencePlayersTable après chaque clic d'en-tête : figé avant et après le
 * passage du tri sur TanStack Table (étape 7.7).
 */
import { QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { AdminConvergenceReport, PlayerConvergenceReport } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { createTestQueryClient } from '@/test/render-utils'
import { ConvergencePlayersTable } from './ConvergencePlayersTable'

vi.mock('../data-quality/mutations', () => ({
  useRunPlayerConvergence: () => ({ mutateAsync: vi.fn() }),
}))

function player(gamertag: string, enrich: number, psa: number, events: number): PlayerConvergenceReport {
  return { gamertag, xuid: gamertag, player_slug: gamertag, missing_enrichment: enrich, missing_psa: psa, missing_events: events }
}

const REPORT: AdminConvergenceReport = {
  generated_at: '2026-07-24T00:00:00Z',
  horizon: 100,
  players: [player('Delta', 5, 0, 7), player('alpha', 20, 3, 0), player('Charlie', 5, 9, 1), player('bravo', 0, 3, 7), player('Echo', 20, 0, 0)],
  title_slug: 'halo_infinite',
  totals_since_boot: { aliases_upserted: 0, events_processed: 0, psa_processed: 0 },
}

describe('ConvergencePlayersTable — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(
      <QueryClientProvider client={createTestQueryClient()}>
        <ConvergencePlayersTable report={REPORT} />
      </QueryClientProvider>,
    )
    const recording = recordSortOrders(container, [
      { name: 'Joueur' },
      { name: 'Enrichment manquant' },
      { name: 'PSA non tentés' },
      { name: 'Events incomplets' },
      { name: 'Total' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Enrichment manquant": "1: alpha > Echo > Delta > Charlie > bravo | 2: bravo > Delta > Charlie > alpha > Echo | 3: alpha > Echo > Delta > Charlie > bravo | aria-sort: none,descending,none,none,none,-",
        "Events incomplets": "1: Delta > bravo > Charlie > alpha > Echo | 2: alpha > Echo > Charlie > Delta > bravo | 3: Delta > bravo > Charlie > alpha > Echo | aria-sort: none,none,none,descending,none,-",
        "Joueur": "1: alpha > bravo > Charlie > Delta > Echo | 2: Echo > Delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > Delta > Echo | aria-sort: ascending,none,none,none,none,-",
        "PSA non tentés": "1: Charlie > alpha > bravo > Delta > Echo | 2: Delta > Echo > alpha > bravo > Charlie | 3: Charlie > alpha > bravo > Delta > Echo | aria-sort: none,none,descending,none,none,-",
        "Total": "1: alpha > Echo > Charlie > Delta > bravo | 2: bravo > Delta > Charlie > Echo > alpha | 3: alpha > Echo > Charlie > Delta > bravo | aria-sort: none,none,none,none,descending,-",
        "initial": "Delta > alpha > Charlie > bravo > Echo",
      }
    `)
  })
})
