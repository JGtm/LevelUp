/**
 * Tests — HomeSessionCarousel.
 *
 * Vérifie le câblage de navigation (onNavigate reçoit le label ET les coéquipiers
 * de la session) et l'affordance de cliquabilité (bordure au survol).
 */
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import type { SessionSummaryItem } from '@/lib/api/types'
import { HomeSessionCarousel } from './HomeSessionCarousel'

function makeSession(overrides: Partial<SessionSummaryItem> = {}): SessionSummaryItem {
  return {
    session_label: '10/06/2026 19:52–20:14 (3)',
    match_count: 3,
    win_rate: 0.66,
    global_ratio: 1.2,
    started_at: '2026-06-10T19:52:00Z',
    ended_at: '2026-06-10T20:14:00Z',
    wins: 2,
    losses: 1,
    draws: 0,
    dnfs: 0,
    avg_player_performance: 80,
    avg_team_performance: 75,
    avg_kda: 1.5,
    dominant_playlist: 'Ranked',
    dominant_mode: 'Slayer',
    ...overrides,
  }
}

describe('HomeSessionCarousel', () => {
  it('escouade : onNavigate reçoit le label ET les coéquipiers de la session', () => {
    const onNavigate = vi.fn()
    const session = makeSession({ teammates: ['Alice', 'Bob'] })
    renderWithProviders(
      <HomeSessionCarousel
        sessions={[session]}
        idx={0}
        onIdxChange={() => {}}
        variant="squad"
        playerSlug="p"
        onNavigate={onNavigate}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Voir le détail de la session/i }))
    expect(onNavigate).toHaveBeenCalledWith(session.session_label, ['Alice', 'Bob'])
  })

  it('solo : onNavigate reçoit une liste de coéquipiers vide (teammates absent)', () => {
    const onNavigate = vi.fn()
    const session = makeSession({ teammates: undefined })
    renderWithProviders(
      <HomeSessionCarousel
        sessions={[session]}
        idx={0}
        onIdxChange={() => {}}
        variant="solo"
        playerSlug="p"
        onNavigate={onNavigate}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Voir le détail de la session/i }))
    expect(onNavigate).toHaveBeenCalledWith(session.session_label, [])
  })

  it('affiche une bordure au survol (affordance de cliquabilité, token sémantique)', () => {
    renderWithProviders(
      <HomeSessionCarousel
        sessions={[makeSession()]}
        idx={0}
        onIdxChange={() => {}}
        variant="solo"
        playerSlug="p"
        onNavigate={() => {}}
      />,
    )
    const card = screen.getByRole('button', { name: /Voir le détail de la session/i })
    expect(card.className).toContain('border')
    expect(card.className).toContain('hover:border-primary')
  })
})
