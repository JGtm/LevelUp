/**
 * Tests SessionParamPills — pills paramètres de session (nb matchs / catégorie FR / durée).
 * Locale par défaut = fr.
 */
import { describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { SessionCompareEntry } from '@/lib/api/types'

import { SessionParamPills } from './SessionParamPills'

function entry(overrides: Partial<SessionCompareEntry> = {}): SessionCompareEntry {
  return {
    session_label: 'S1',
    start_time: '2026-04-21T19:30:00Z',
    end_time: '2026-04-21T20:05:00Z',
    total_matches: 8,
    wins: 6,
    losses: 2,
    kda: 2.4,
    performance_score: 68,
    win_rate: 75,
    kdr: 1.8,
    kills_per_match: 14,
    with_friends: false,
    dominant_category: 'Ranked',
    match_series: [],
    participation: [],
    matches: [],
    ...overrides,
  }
}

describe('SessionParamPills', () => {
  it('affiche nb de matchs, catégorie FR et durée', () => {
    renderWithProviders(<SessionParamPills entry={entry()} />)
    expect(screen.getByText('8 matchs')).toBeInTheDocument()
    expect(screen.getByText('Classé')).toBeInTheDocument() // Ranked → FR
    expect(screen.getByText('35 min')).toBeInTheDocument() // 19h30 → 20h05
  })

  it('localise les catégories (Arena → Arène)', () => {
    renderWithProviders(<SessionParamPills entry={entry({ dominant_category: 'Arena' })} />)
    expect(screen.getByText('Arène')).toBeInTheDocument()
  })

  it('ne rend rien si entry est null', () => {
    const { container } = renderWithProviders(<SessionParamPills entry={null} />)
    expect(container).toBeEmptyDOMElement()
  })
})
