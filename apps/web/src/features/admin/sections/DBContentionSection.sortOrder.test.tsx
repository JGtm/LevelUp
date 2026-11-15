/**
 * Ordre des lignes de DBContentionSection (détenteurs) après chaque clic d'en-tête : figé avant et après
 * le passage du tri sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { DBContentionResponse } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { DBContentionSection } from './DBContentionSection'

function holder(label: string, count: number, total: number, avg: number, max: number, watchdog: number) {
  return { label, count, total_ms: total, avg_ms: avg, max_ms: max, watchdog_fired: watchdog }
}

const DATA: DBContentionResponse = {
  avg_acquire_ms: 0,
  avg_blocked_ms: 0,
  avg_release_ms: 0,
  avg_rw_window_ms: 0,
  drain_ms_total: 0,
  generated_at: '2026-07-24T00:00:00Z',
  holders: [
    holder('delta', 3, 30, 10, 12, 0),
    holder('alpha', 1, 90, 90, 90, 2),
    holder('Charlie', 3, 30, 5, 40, 1),
    holder('bravo', 8, 16, 2, 4, 0),
    holder('echo10', 3, 7, 7, 9, 2),
    holder('echo2', 3, 7, 7, 9, 2),
  ],
  max_blocked_ms: 0,
  max_rw_window_ms: 0,
  readers_in_use: 0,
  reads_rejected: 0,
  state: 'RO',
  swap_failures: 0,
  swaps: 0,
  watchdog_fired: 0,
}

vi.mock('../queries', () => ({
  useAdminDBContention: () => ({ data: DATA, isLoading: false, isError: false, refetch: vi.fn(), isFetching: false }),
}))

describe('DBContentionSection — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<DBContentionSection />)
    const recording = recordSortOrders(container, [
      { name: 'Détenteur' },
      { name: 'Nb' },
      { name: 'Total' },
      { name: 'Moy.' },
      { name: 'Max' },
      { name: /Watchdog/ },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "/Watchdog/": "1: alpha > echo10 > echo2 > Charlie > delta > bravo | 2: delta > bravo > Charlie > alpha > echo10 > echo2 | 3: alpha > echo10 > echo2 > Charlie > delta > bravo | aria-sort: none,none,none,none,none,descending",
        "Détenteur": "1: alpha > bravo > Charlie > delta > echo2 > echo10 | 2: echo10 > echo2 > delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > delta > echo2 > echo10 | aria-sort: ascending,none,none,none,none,none",
        "Max": "1: alpha > Charlie > delta > echo10 > echo2 > bravo | 2: bravo > echo10 > echo2 > delta > Charlie > alpha | 3: alpha > Charlie > delta > echo10 > echo2 > bravo | aria-sort: none,none,none,none,descending,none",
        "Moy.": "1: alpha > delta > echo10 > echo2 > Charlie > bravo | 2: bravo > Charlie > echo10 > echo2 > delta > alpha | 3: alpha > delta > echo10 > echo2 > Charlie > bravo | aria-sort: none,none,none,descending,none,none",
        "Nb": "1: bravo > delta > Charlie > echo10 > echo2 > alpha | 2: alpha > delta > Charlie > echo10 > echo2 > bravo | 3: bravo > delta > Charlie > echo10 > echo2 > alpha | aria-sort: none,descending,none,none,none,none",
        "Total": "1: alpha > delta > Charlie > bravo > echo10 > echo2 | 2: echo10 > echo2 > bravo > delta > Charlie > alpha | 3: alpha > delta > Charlie > bravo > echo10 > echo2 | aria-sort: none,none,descending,none,none,none",
        "initial": "delta > alpha > Charlie > bravo > echo10 > echo2",
      }
    `)
  })
})
