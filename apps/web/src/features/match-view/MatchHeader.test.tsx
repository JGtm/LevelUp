import { describe, it, expect, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { MatchHeaderCard, MatchNavigationBar } from './MatchHeader'
import type { MatchViewHeader, MatchViewRank } from '@/lib/api/types'

// Mocks shared
vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(${token})`,
}))

vi.mock('@/lib/accessibility/scales', () => ({
  skillDeltaScale: () => 'outcome-win',
}))

vi.mock('@tanstack/react-router', () => ({
  Link: ({ children }: { children: ReactNode }) => <a>{children}</a>,
  useNavigate: () => vi.fn(),
  useRouter: () => ({ history: { length: 1, back: vi.fn() }, navigate: vi.fn() }),
  useRouterState: () => undefined,
}))

vi.mock('./queries', () => ({
  useMatchNeighbors: () => ({ data: null }),
  useToggleMatchFavorite: () => ({ mutate: vi.fn(), isPending: false }),
}))

const navigateToMatchMock = vi.fn()
vi.mock('@/lib/match-nav/useNavigateToMatch', () => ({
  useNavigateToMatch: () => navigateToMatchMock,
}))

const resolvedRef: {
  current: {
    data: { previous_match_id: string | null; next_match_id: string | null; current_index: number; total_matches: number } | undefined
    isPending: boolean
    source: 'router-state' | 'session-storage' | 'api'
    contextLabel?: string
    navContext?: { source: string; matchIds: string[]; filtersLabel?: string }
  }
} = {
  current: {
    data: { previous_match_id: 'prev-id', next_match_id: 'next-id', current_index: 1, total_matches: 4 },
    isPending: false,
    source: 'api',
  },
}
vi.mock('@/lib/match-nav/useMatchNeighborsResolved', () => ({
  useMatchNeighborsResolved: () => resolvedRef.current,
}))

const clearNavContextMock = vi.fn()
vi.mock('@/lib/match-nav/navContext', () => ({
  clearNavContext: (...args: unknown[]) => clearNavContextMock(...args),
}))

vi.mock('@/features/match-history/queries', () => ({
  useSetMatchExclusion: () => ({ mutate: vi.fn(), isPending: false }),
}))

const baseHeader: MatchViewHeader = {
  match_id: 'm1',
  start_time: null,
  start_time_label: 'Dim. 4 mai 2026 · 19h35',
  outcome_code: 2,
  outcome_label: 'Victoire',
  outcome_color: '#22c55e',
  outcome_color_token: 'outcome-win',
  score_label: '87 - 62',
  dominance_flag: false,
  had_bot_teammate: false,
  map_ui: 'Aquarius',
  map_id: null,
  mode_ui: 'Slayer',
  playlist_label: 'Classée',
  performance_display: '76',
  performance_color: null,
  performance_color_token: 'perf-tier-2',
  is_excluded: false,
  is_favorite: false,
  map_image_url: '/static/maps/halo_infinite/Aquarius.png',
}

const baseRank: MatchViewRank = {
  rating_type: 'CSR',
  tier_label: 'Diamond 1',
  numeric_value: 1452,
  delta_value: 34,
  icon_url: '/static/ranks/halo_infinite/120px-HINF-CSR_Diamond1.png',
}

function renderWithQueryClient(node: ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>)
}

describe('MatchHeaderCard', () => {
  it('affiche outcome, score, playlist, performance, rang en FR', () => {
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.getByText('Slayer sur Aquarius')).toBeInTheDocument()
    expect(screen.getByText('Victoire')).toBeInTheDocument()
    expect(screen.getByText('87 - 62')).toBeInTheDocument()
    expect(screen.getByText('Classée')).toBeInTheDocument()
    expect(screen.getByText('76')).toBeInTheDocument()
    expect(screen.getByText('Diamond 1')).toBeInTheDocument()
    expect(screen.getByText('CSR 1452')).toBeInTheDocument()
    expect(screen.getByText('▲ +34')).toBeInTheDocument()
    expect(screen.getByText('Performance')).toBeInTheDocument()
    expect(screen.getByText('Rang')).toBeInTheDocument()
    // Action labels FR (boutons courts)
    expect(screen.getByText('Copier ID')).toBeInTheDocument()
    expect(screen.getByText('Exclure')).toBeInTheDocument()
  })

  it('affiche les libellés EN quand locale=en', () => {
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer on Aquarius"
        locale="en"
      />,
    )
    expect(screen.getByText('Performance')).toBeInTheDocument()
    expect(screen.getByText('Rank')).toBeInTheDocument()
    expect(screen.getByText('Copy ID')).toBeInTheDocument()
    expect(screen.getByText('Exclude')).toBeInTheDocument()
  })

  it('affiche le fallback texte si map_image_url est null', () => {
    const noImage: MatchViewHeader = { ...baseHeader, map_image_url: null }
    renderWithQueryClient(
      <MatchHeaderCard
        header={noImage}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    // Le fallback : nom de map (deux occurrences possibles via alt + texte)
    const aquarius = screen.queryAllByText('Aquarius')
    expect(aquarius.length).toBeGreaterThan(0)
    // Pas d'image
    expect(screen.queryByRole('img', { name: /Aquarius/ })).toBeNull()
  })

  it('match exclu : affiche le bouton "Réactiver"', () => {
    const excluded: MatchViewHeader = { ...baseHeader, is_excluded: true }
    renderWithQueryClient(
      <MatchHeaderCard
        header={excluded}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.getByText('Réactiver')).toBeInTheDocument()
  })

  it('rating_type=none : ne rend pas la section rang', () => {
    const noRank: MatchViewRank = {
      rating_type: 'none',
      tier_label: null,
      numeric_value: null,
      delta_value: null,
      icon_url: null,
    }
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={noRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.queryByText('Rang')).toBeNull()
  })

  it('is_favorite=true : aria-label "Retirer des favoris"', () => {
    const fav: MatchViewHeader = { ...baseHeader, is_favorite: true }
    renderWithQueryClient(
      <MatchHeaderCard
        header={fav}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.getByRole('button', { name: 'Retirer des favoris' })).toBeInTheDocument()
  })
})

describe('MatchNavigationBar', () => {
  beforeEach(() => {
    navigateToMatchMock.mockClear()
    clearNavContextMock.mockClear()
    resolvedRef.current = {
      data: { previous_match_id: 'prev-id', next_match_id: 'next-id', current_index: 1, total_matches: 4 },
      isPending: false,
      source: 'api',
    }
  })

  it('rendu fallback API : affiche compteur sans contextLabel ni bouton sortir', () => {
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    expect(screen.getByText('Match 2/4')).toBeInTheDocument()
    expect(screen.queryByText(/Sortir du contexte/)).toBeNull()
  })

  it('rendu router-state : affiche contextLabel + lien sortir', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 0, total_matches: 12 },
      isPending: false,
      source: 'router-state',
      contextLabel: 'Classée · 7 derniers jours',
      navContext: { source: 'history', matchIds: ['m1', 'p', 'n'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    expect(screen.getByText('Classée · 7 derniers jours')).toBeInTheDocument()
    expect(screen.getByText(/↩ Sortir du contexte/)).toBeInTheDocument()
  })

  it('clic prev/next : propage le navContext courant au helper', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'prev-id', next_match_id: 'next-id', current_index: 1, total_matches: 4 },
      isPending: false,
      source: 'session-storage',
      contextLabel: 'Session 04-30',
      navContext: { source: 'session', matchIds: ['next-id', 'm1', 'prev-id'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'Match suivant' }))
    expect(navigateToMatchMock).toHaveBeenCalledWith(
      'next-id',
      expect.objectContaining({ source: 'session', matchIds: ['next-id', 'm1', 'prev-id'] }),
    )
  })

  it('clic Sortir du contexte : appelle clearNavContext(matchId)', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 0, total_matches: 3 },
      isPending: false,
      source: 'router-state',
      contextLabel: 'Top matchs',
      navContext: { source: 'history', matchIds: ['m1', 'p', 'n'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    fireEvent.click(screen.getByText(/↩ Sortir du contexte/))
    expect(clearNavContextMock).toHaveBeenCalledWith('m1')
  })

  it('locale=en : counter et boutons en EN', () => {
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="en" />,
    )
    expect(screen.getByText('Match 2/4')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous match' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Next match' })).toBeInTheDocument()
  })
})
