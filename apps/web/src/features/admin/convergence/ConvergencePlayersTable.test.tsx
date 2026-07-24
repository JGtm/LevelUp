/**
 * ConvergencePlayersTable.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'

import { createTestQueryClient } from '@/test/render-utils'
import type { AdminConvergenceReport, PlayerConvergenceReport } from '@/lib/api/types'
import { ConvergencePlayersTable } from './ConvergencePlayersTable'

vi.mock('../data-quality/mutations', () => ({
  useRunPlayerConvergence: () => ({ mutateAsync: vi.fn() }),
}))

function makePlayer(overrides: Partial<PlayerConvergenceReport>): PlayerConvergenceReport {
  return {
    gamertag: 'Player',
    xuid: 'x',
    player_slug: 'player',
    missing_enrichment: 0,
    missing_psa: 0,
    missing_events: 0,
    missing_weapons: 0,
    ...overrides,
  }
}

function makeReport(players: PlayerConvergenceReport[]): AdminConvergenceReport {
  return {
    generated_at: '2026-07-24T00:00:00Z',
    horizon: 100,
    players,
    title_slug: 'halo_infinite',
    totals_since_boot: { aliases_upserted: 0, events_processed: 0, psa_processed: 0, weapons_processed: 0 },
  }
}

function renderTable(report: AdminConvergenceReport) {
  const qc = createTestQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <ConvergencePlayersTable report={report} />
    </QueryClientProvider>,
  )
}

/** Ordre des lignes du tbody, identifiées par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('ConvergencePlayersTable — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function report() {
    return makeReport([
      makePlayer({ xuid: 'x1', gamertag: 'Alpha', missing_enrichment: 5 }),
      makePlayer({ xuid: 'x2', gamertag: 'Bravo', missing_enrichment: 20 }),
      makePlayer({ xuid: 'x3', gamertag: 'Charlie', missing_enrichment: 10 }),
    ])
  }

  it('sans clic : ordre serveur conservé', () => {
    renderTable(report())
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('clic sur « Enrichissement » trie numériquement (desc au 1er clic)', () => {
    renderTable(report())
    const header = screen.getByText('Enrichment manquant')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['Bravo', 'Charlie', 'Alpha'])
  })
})
