import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { MatchHeaderCard } from './MatchHeader'
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
}))

vi.mock('./queries', () => ({
  useMatchNeighbors: () => ({ data: null }),
  useToggleMatchFavorite: () => ({ mutate: vi.fn(), isPending: false }),
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
    // Action labels FR
    expect(screen.getByText("Copier l'ID du match")).toBeInTheDocument()
    expect(screen.getByText('Marquer comme non pertinent ⊘')).toBeInTheDocument()
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
    expect(screen.getByText('Copy match ID')).toBeInTheDocument()
    expect(screen.getByText('Mark as irrelevant ⊘')).toBeInTheDocument()
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
    expect(screen.getByText('↩ Réactiver')).toBeInTheDocument()
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
