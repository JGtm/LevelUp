/**
 * MatchScoreboard.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 *
 * Pattern DetectionsPanel minimal : clic sur le <th> (pas de bouton dédié),
 * flèche suffixe, ordre par défaut = ordre serveur (aucun tri actif tant
 * qu'aucun en-tête n'a été cliqué). L'ordre EXACT du 1er clic (asc/desc)
 * n'est pas figé côté TanStack (pas de `sortDescFirst` de table) : le test
 * vérifie que le clic change bien l'ordre et qu'un second clic l'inverse.
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { createTestQueryClient } from '@/test/render-utils'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { MATCH_VIEW_TEXT } from './i18n'
import { MatchScoreboard } from './MatchScoreboard'

vi.mock('@tanstack/react-router', () => ({
  useParams: () => ({}),
  useNavigate: () => vi.fn(),
}))

function renderScoreboard(ui: ReactNode) {
  const qc = createTestQueryClient()
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>)
}

function makeRow(overrides: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return {
    xuid: 'x',
    gamertag: 'Player',
    team_side: 't0',
    is_me: false,
    rank: 1,
    score: 100,
    kills: 0,
    deaths: 0,
    assists: 0,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'Victoire',
    ...overrides,
  }
}

/** Ordre des lignes du tbody (une seule équipe/table dans ce test), identifiées
 *  par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('MatchScoreboard — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function rows(): MatchScoreboardRow[] {
    return [
      makeRow({ xuid: 'x1', gamertag: 'Alpha', kills: 5 }),
      makeRow({ xuid: 'x2', gamertag: 'Bravo', kills: 20 }),
      makeRow({ xuid: 'x3', gamertag: 'Charlie', kills: 10 }),
    ]
  }

  it('sans clic : ordre serveur conservé (aucun tri actif par défaut)', () => {
    renderScoreboard(<MatchScoreboard rows={rows()} t={MATCH_VIEW_TEXT.fr} />)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
    const fragsHeader = screen.getByText('Frags').closest('th')
    expect(fragsHeader).toHaveAttribute('aria-sort', 'none')
  })

  it('clic sur « Frags » trie par frags, un 2e clic inverse l’ordre', () => {
    renderScoreboard(<MatchScoreboard rows={rows()} t={MATCH_VIEW_TEXT.fr} />)
    const fragsHeader = screen.getByText('Frags').closest('th') as HTMLElement
    fireEvent.click(fragsHeader)
    const afterFirstClick = rowOrder(names)
    expect(afterFirstClick).not.toEqual(['Alpha', 'Bravo', 'Charlie'])
    expect(fragsHeader).not.toHaveAttribute('aria-sort', 'none')
    fireEvent.click(fragsHeader)
    const afterSecondClick = rowOrder(names)
    expect(afterSecondClick).toEqual([...afterFirstClick].reverse())
  })

  it('colonne « Rang & MMR » (badge image) n’est jamais triable', () => {
    renderScoreboard(<MatchScoreboard rows={rows()} t={MATCH_VIEW_TEXT.fr} />)
    // sbDetailLusr est le libellé par défaut (match non classé dans ce test).
    const badgeHeader = screen.getByText(MATCH_VIEW_TEXT.fr.sbDetailLusr).closest('th')
    expect(badgeHeader).not.toHaveAttribute('aria-sort')
  })
})
