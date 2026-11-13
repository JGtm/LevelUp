/**
 * Ordre des lignes des deux tableaux de ApiHaloSection (par appel, par joueur) après chaque clic
 * d'en-tête : figé avant et après le passage du tri sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { AdminPerfStats, PerfCallStats, PerfPlayerCallStats } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { ApiHaloSection } from './ApiHaloSection'

function call(name: string, count: number, avg: number, max: number, sum: number, errors?: number): PerfCallStats {
  return { name, count, avg_ms: avg, max_ms: max, sum_ms: sum, errors }
}

function byPlayer(player: string, c: string, count: number, avg: number, max: number, errors: number): PerfPlayerCallStats {
  return { player, call: c, count, avg_ms: avg, max_ms: max, errors }
}

const PERF: AdminPerfStats = {
  api_calls: [
    call('delta', 5, 12, 30, 60, 2),
    call('alpha', 20, 5, 8, 100),
    call('Charlie', 5, 12, 9, 60, 2),
    call('bravo', 1, 90, 90, 90, 7),
  ],
  api_by_player: [
    byPlayer('zed', 'stats', 4, 10, 20, 1),
    byPlayer('amy', 'films', 9, 3, 5, 0),
    byPlayer('amy', 'stats', 4, 10, 40, 1),
    byPlayer('Bob', 'films', 9, 8, 5, 2),
  ],
  api_buckets: { rate_limited_429: 0, auth: 0, server_5xx: 0, network: 0, other: 0 },
  blocked_window: call('blocked_window', 0, 0, 0, 0),
  generated_at: '2026-07-24T00:00:00Z',
  persist_phases: [],
  postsync_steps: [],
  postsync_total: call('postsync_total', 0, 0, 0, 0),
}

describe('ApiHaloSection — ordre des lignes par en-tête', () => {
  it('tableau par appel : ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<ApiHaloSection perf={PERF} />)
    const table = container.querySelectorAll('table')[0] as HTMLElement
    const recording = recordSortOrders(table, [
      { name: 'Appel' },
      { name: 'Appels' },
      { name: 'Moy.' },
      { name: 'Max' },
      { name: 'Cumul' },
      { name: 'Erreurs' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Appel": "1: alpha > bravo > Charlie > delta | 2: delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > delta | aria-sort: ascending,none,none,none,none,none",
        "Appels": "1: alpha > delta > Charlie > bravo | 2: bravo > delta > Charlie > alpha | 3: alpha > delta > Charlie > bravo | aria-sort: none,descending,none,none,none,none",
        "Cumul": "1: alpha > bravo > delta > Charlie | 2: delta > Charlie > bravo > alpha | 3: alpha > bravo > delta > Charlie | aria-sort: none,none,none,none,descending,none",
        "Erreurs": "1: bravo > delta > Charlie > alpha | 2: alpha > delta > Charlie > bravo | 3: bravo > delta > Charlie > alpha | aria-sort: none,none,none,none,none,descending",
        "Max": "1: bravo > delta > Charlie > alpha | 2: alpha > Charlie > delta > bravo | 3: bravo > delta > Charlie > alpha | aria-sort: none,none,none,descending,none,none",
        "Moy.": "1: bravo > delta > Charlie > alpha | 2: alpha > delta > Charlie > bravo | 3: bravo > delta > Charlie > alpha | aria-sort: none,none,descending,none,none,none",
        "initial": "delta > alpha > Charlie > bravo",
      }
    `)
  })

  it('tableau par joueur : ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<ApiHaloSection perf={PERF} />)
    const table = container.querySelectorAll('table')[1] as HTMLElement
    const recording = recordSortOrders(table, [
      { name: 'Joueur' },
      { name: 'Appel' },
      { name: 'Appels' },
      { name: 'Moy.' },
      { name: 'Max' },
      { name: 'Erreurs' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Appel": "1: amy > Bob > zed > amy | 2: zed > amy > amy > Bob | 3: amy > Bob > zed > amy | aria-sort: none,ascending,none,none,none,none",
        "Appels": "1: amy > Bob > zed > amy | 2: zed > amy > amy > Bob | 3: amy > Bob > zed > amy | aria-sort: none,none,descending,none,none,none",
        "Erreurs": "1: Bob > zed > amy > amy | 2: amy > zed > amy > Bob | 3: Bob > zed > amy > amy | aria-sort: none,none,none,none,none,descending",
        "Joueur": "1: amy > amy > Bob > zed | 2: zed > Bob > amy > amy | 3: amy > amy > Bob > zed | aria-sort: ascending,none,none,none,none,none",
        "Max": "1: amy > zed > amy > Bob | 2: amy > Bob > zed > amy | 3: amy > zed > amy > Bob | aria-sort: none,none,none,none,descending,none",
        "Moy.": "1: zed > amy > Bob > amy | 2: amy > Bob > zed > amy | 3: zed > amy > Bob > amy | aria-sort: none,none,none,descending,none,none",
        "initial": "zed > amy > amy > Bob",
      }
    `)
  })
})
