/**
 * ApiHaloSection.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { AdminPerfStats, PerfCallStats } from '@/lib/api/types'
import { ApiHaloSection } from './ApiHaloSection'

function callStats(name: string, count: number): PerfCallStats {
  return { name, count, avg_ms: 10, max_ms: 20, sum_ms: 50, errors: 0 }
}

function makePerf(): AdminPerfStats {
  return {
    api_calls: [callStats('alpha', 5), callStats('bravo', 20), callStats('charlie', 10)],
    api_by_player: [],
    api_buckets: { rate_limited_429: 0, auth: 0, server_5xx: 0, network: 0, other: 0 },
    blocked_window: callStats('blocked_window', 0),
    generated_at: '2026-07-24T00:00:00Z',
    persist_phases: [],
    postsync_steps: [],
    postsync_total: callStats('postsync_total', 0),
  }
}

/** Ordre des lignes du tbody, identifiées par le nom d'appel qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('ApiHaloSection — tri CLIENT par en-têtes (I16)', () => {
  const names = ['alpha', 'bravo', 'charlie']

  it('sans clic : ordre serveur conservé', () => {
    render(<ApiHaloSection perf={makePerf()} />)
    expect(rowOrder(names)).toEqual(['alpha', 'bravo', 'charlie'])
  })

  it('clic sur « Appels » trie par count, un 2e clic inverse l’ordre', () => {
    render(<ApiHaloSection perf={makePerf()} />)
    const header = screen.getByText('Appels')
    fireEvent.click(header)
    // 1er clic numérique → descendant : 20, 10, 5.
    expect(rowOrder(names)).toEqual(['bravo', 'charlie', 'alpha'])
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['alpha', 'charlie', 'bravo'])
  })
})
