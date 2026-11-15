/**
 * Ordre des lignes de IssueTable après chaque clic d'en-tête : figé avant et après le passage du tri
 * sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { AdminDataQualityIssue } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { IssueTable, type IssueColumn } from './IssueTable'

const COLUMNS: IssueColumn[] = [
  { header: 'ID', cell: (i) => i.id, sortValue: (i) => i.id },
  { header: 'Poids', cell: (i) => String(i.occurrences), sortValue: (i) => (i.id === 'echo' ? null : i.occurrences % 3), sortDescFirst: true },
  { header: 'Sans tri', cell: () => '-' },
]

function issue(id: string, occurrences: number, lastSeen?: string): AdminDataQualityIssue {
  return { kind: 'raw_uuid', id, occurrences, last_seen: lastSeen }
}

const ISSUES: AdminDataQualityIssue[] = [
  issue('delta', 4, '2026-05-02T10:00:00Z'),
  issue('alpha', 9),
  issue('Charlie', 4, '2026-05-03T10:00:00Z'),
  issue('bravo', 1, '2026-05-01T10:00:00Z'),
  issue('echo', 7),
]

describe('IssueTable — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<IssueTable issues={ISSUES} columns={COLUMNS} sortable />)
    const recording = recordSortOrders(container, [
      { name: 'ID' },
      { name: 'Poids' },
      { name: 'Matchs' },
      { name: 'Vu pour la dernière fois' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "ID": "1: alpha > bravo > Charlie > delta > echo | 2: echo > delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > delta > echo | aria-sort: ascending,none,-,none,none",
        "Matchs": "1: alpha > echo > delta > Charlie > bravo | 2: bravo > delta > Charlie > echo > alpha | 3: alpha > echo > delta > Charlie > bravo | aria-sort: none,none,-,descending,none",
        "Poids": "1: delta > Charlie > bravo > alpha > echo | 2: alpha > delta > Charlie > bravo > echo | 3: delta > Charlie > bravo > alpha > echo | aria-sort: none,descending,-,none,none",
        "Vu pour la dernière fois": "1: Charlie > delta > bravo > alpha > echo | 2: bravo > delta > Charlie > alpha > echo | 3: Charlie > delta > bravo > alpha > echo | aria-sort: none,none,-,none,descending",
        "initial": "delta > alpha > Charlie > bravo > echo",
      }
    `)
  })
})
