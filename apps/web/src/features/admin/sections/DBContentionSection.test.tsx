/**
 * DBContentionSection.test.tsx — tri CLIENT par clic sur les en-têtes (I16),
 * table de ventilation des détenteurs (holders).
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { DBContentionResponse } from '@/lib/api/types'
import { DBContentionSection } from './DBContentionSection'

function makeHolder(label: string, total_ms: number) {
  return { label, count: 1, total_ms, avg_ms: total_ms, max_ms: total_ms, watchdog_fired: 0 }
}

function makeData(): DBContentionResponse {
  return {
    avg_acquire_ms: 0,
    avg_blocked_ms: 0,
    avg_release_ms: 0,
    avg_rw_window_ms: 0,
    drain_ms_total: 0,
    generated_at: '2026-07-24T00:00:00Z',
    holders: [makeHolder('alpha', 5), makeHolder('bravo', 20), makeHolder('charlie', 10)],
    max_blocked_ms: 0,
    max_rw_window_ms: 0,
    readers_in_use: 0,
    reads_rejected: 0,
    state: 'RO',
    swap_failures: 0,
    swaps: 0,
    watchdog_fired: 0,
  }
}

const dataRef: { current: DBContentionResponse | undefined } = { current: undefined }

vi.mock('../queries', () => ({
  useAdminDBContention: () => ({ data: dataRef.current, isLoading: false, isError: false, refetch: vi.fn(), isFetching: false }),
}))

/** Ordre des lignes du tbody, identifiées par le label qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('DBContentionSection — tri CLIENT par en-têtes (I16)', () => {
  const names = ['alpha', 'bravo', 'charlie']

  it('sans clic : ordre serveur conservé (total_ms DESC déjà appliqué API)', () => {
    dataRef.current = makeData()
    render(<DBContentionSection />)
    expect(rowOrder(names)).toEqual(['alpha', 'bravo', 'charlie'])
  })

  it('clic sur « Total » trie numériquement (desc au 1er clic)', () => {
    dataRef.current = makeData()
    render(<DBContentionSection />)
    const header = screen.getByText('Total')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['bravo', 'charlie', 'alpha'])
  })
})
