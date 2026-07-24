/**
 * CareerEncountersSection.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 *
 * Pattern SortableTh partagé (components/ui/sortable-th.tsx). Ordre par défaut =
 * ordre serveur (teammates puis enemies) tant qu'aucun en-tête n'a été cliqué.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { EncounterDTO } from '@/lib/api/types'
import { CareerEncountersSection } from './CareerEncountersSection'

function makeEncounter(overrides: Partial<EncounterDTO>): EncounterDTO {
  return {
    xuid: 'x',
    gamertag: 'Player',
    match_count: 1,
    as_teammate: 1,
    as_enemy: 0,
    avg_kda: 1,
    ...overrides,
  }
}

const dataRef: { current: { teammates: EncounterDTO[]; enemies: EncounterDTO[]; total: number } | undefined } = {
  current: undefined,
}

vi.mock('./queries', () => ({
  useCareerEncounters: () => ({ data: dataRef.current, isLoading: false }),
}))

/** Ordre des lignes du tbody, identifiées par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('CareerEncountersSection — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function setup() {
    dataRef.current = {
      teammates: [makeEncounter({ xuid: 'x1', gamertag: 'Alpha', match_count: 5 })],
      enemies: [
        makeEncounter({ xuid: 'x2', gamertag: 'Bravo', match_count: 20 }),
        makeEncounter({ xuid: 'x3', gamertag: 'Charlie', match_count: 10 }),
      ],
      total: 3,
    }
  }

  it('sans clic : ordre serveur conservé (teammates puis enemies)', () => {
    setup()
    renderWithProviders(<CareerEncountersSection playerSlug="me" />)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('clic sur la colonne Matchs trie par match_count, un 2e clic inverse l’ordre', () => {
    setup()
    renderWithProviders(<CareerEncountersSection playerSlug="me" />)
    // useFieldMappings n'est pas mocké dans ce test → labelOf retombe sur la clé
    // brute (fallback documenté dans fieldMappings.ts), pas le libellé traduit.
    const header = screen.getByText('total_matches_played')
    fireEvent.click(header)
    // 1er clic sur une colonne numérique → descendant : 20, 10, 5.
    expect(rowOrder(names)).toEqual(['Bravo', 'Charlie', 'Alpha'])
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['Alpha', 'Charlie', 'Bravo'])
  })
})
