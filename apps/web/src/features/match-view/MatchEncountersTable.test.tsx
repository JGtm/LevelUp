/**
 * MatchEncountersTable.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 *
 * Pattern DetectionsPanel minimal (clic direct sur le <th>), ordre par défaut =
 * ordre serveur (aucun tri actif tant qu'aucun en-tête n'a été cliqué).
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { createTestQueryClient } from '@/test/render-utils'
import type { MatchEncounterRow } from '@/lib/api/types'
import { MatchEncountersTable } from './MatchEncountersTable'

vi.mock('@tanstack/react-router', () => ({
  useParams: () => ({}),
  useNavigate: () => vi.fn(),
}))

function renderTable(ui: ReactNode) {
  const qc = createTestQueryClient()
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

function makeRow(overrides: Partial<MatchEncounterRow>): MatchEncounterRow {
  return {
    xuid: 'x',
    gamertag: 'Player',
    is_ally: true,
    count_together: 1,
    ...overrides,
  }
}

/** Ordre des lignes du tbody, identifiées par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('MatchEncountersTable — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function rows(): MatchEncounterRow[] {
    return [
      makeRow({ xuid: 'x1', gamertag: 'Alpha', count_together: 5 }),
      makeRow({ xuid: 'x2', gamertag: 'Bravo', count_together: 20 }),
      makeRow({ xuid: 'x3', gamertag: 'Charlie', count_together: 10 }),
    ]
  }

  it('sans clic : ordre serveur conservé (aucun tri actif par défaut)', () => {
    renderTable(<MatchEncountersTable rows={rows()} hideCardWrapper />)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
    const header = screen.getByText('Rencontres').closest('th')
    expect(header).toHaveAttribute('aria-sort', 'none')
  })

  it('clic sur « Rencontres » trie par count_together, un 2e clic inverse l’ordre', () => {
    renderTable(<MatchEncountersTable rows={rows()} hideCardWrapper />)
    const header = screen.getByText('Rencontres').closest('th') as HTMLElement
    fireEvent.click(header)
    const afterFirstClick = rowOrder(names)
    expect(afterFirstClick).not.toEqual(['Alpha', 'Bravo', 'Charlie'])
    expect(header).not.toHaveAttribute('aria-sort', 'none')
    fireEvent.click(header)
    const afterSecondClick = rowOrder(names)
    expect(afterSecondClick).toEqual([...afterFirstClick].reverse())
  })

  it('colonne « Joueur » triable alphabétiquement', () => {
    renderTable(<MatchEncountersTable rows={rows()} hideCardWrapper />)
    const header = screen.getByText('Joueur').closest('th') as HTMLElement
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })
})
