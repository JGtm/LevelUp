/**
 * Ordre des lignes de AdminJobsTable après chaque clic d'en-tête : figé avant et après le passage du
 * tri sur TanStack Table (étape 7.7). Un instantané qui change = un tri qui a changé.
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { AsyncJobStatus } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { AdminJobsTable } from './AdminJobsTable'

function job(id: string, over: Partial<AsyncJobStatus>): AsyncJobStatus {
  return { job_id: id, job_type: id, status: 'succeeded', ...over }
}

const JOBS: AsyncJobStatus[] = [
  job('delta', { status: 'running', player_slug: 'zed', started_at: '2026-05-02T10:00:00Z' }),
  job('alpha', { status: 'failed', player_slug: 'amy', started_at: '2026-05-01T10:00:00Z', finished_at: '2026-05-01T10:05:00Z' }),
  job('charlie', { status: 'succeeded', player_slug: '_all', started_at: '2026-05-03T10:00:00Z', finished_at: '2026-05-03T10:00:30Z' }),
  job('bravo', { status: 'queued' }),
  job('echo', { status: 'succeeded', player_slug: 'amy', started_at: '2026-05-01T10:00:00Z', finished_at: '2026-05-01T10:05:00Z' }),
]

describe('AdminJobsTable — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<AdminJobsTable jobs={JOBS} />)
    const recording = recordSortOrders(container, [
      { name: 'Type' },
      { name: 'Statut' },
      { name: 'Joueur' },
      { name: 'Démarré' },
      { name: 'Durée' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Durée": "1: alpha > echo > charlie > delta > bravo | 2: charlie > alpha > echo > delta > bravo | 3: alpha > echo > charlie > delta > bravo | aria-sort: none,none,none,none,descending",
        "Démarré": "1: charlie > delta > alpha > echo > bravo | 2: alpha > echo > delta > charlie > bravo | 3: charlie > delta > alpha > echo > bravo | aria-sort: none,none,none,descending,none",
        "Joueur": "1: alpha > echo > delta > charlie > bravo | 2: delta > alpha > echo > charlie > bravo | 3: alpha > echo > delta > charlie > bravo | aria-sort: none,none,ascending,none,none",
        "Statut": "1: alpha > bravo > delta > charlie > echo | 2: charlie > echo > delta > bravo > alpha | 3: alpha > bravo > delta > charlie > echo | aria-sort: none,ascending,none,none,none",
        "Type": "1: alpha > bravo > charlie > delta > echo | 2: echo > delta > charlie > bravo > alpha | 3: alpha > bravo > charlie > delta > echo | aria-sort: ascending,none,none,none,none",
        "initial": "delta > alpha > charlie > bravo > echo",
      }
    `)
  })
})
