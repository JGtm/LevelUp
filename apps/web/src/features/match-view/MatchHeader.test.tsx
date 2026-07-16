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
  useToggleMatchFavorite: () => ({ mutate: vi.fn(), isPending: false }),
}))

const navigateToMatchMock = vi.fn()
vi.mock('@/lib/match-nav/useNavigateToMatch', () => ({
  useNavigateToMatch: () => navigateToMatchMock,
}))

type FakeResolved = {
  data: { previous_match_id: string | null; next_match_id: string | null; current_index: number; total_matches: number } | undefined
  isPending: boolean
  source: 'router-state' | 'session-storage' | 'api'
  contextLabel?: string
  contextDescriptor?: import('@/lib/match-nav/navContext').ContextDescriptor
  navContext?: { source: string; matchIds: string[]; filtersLabel?: string }
}
const resolvedRef: { current: FakeResolved } = {
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

const setExclusionMutateMock = vi.fn()
vi.mock('@/features/match-history/queries', () => ({
  useSetMatchExclusion: () => ({ mutate: setExclusionMutateMock, isPending: false }),
}))

vi.mock('sonner', () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

const baseHeader: MatchViewHeader = {
  match_id: 'm1',
  start_time: undefined,
  start_time_label: 'Dim. 4 mai 2026 · 19h35',
  outcome_code: 2,
  outcome_label: 'Victoire',
  outcome_color: '#22c55e',
  outcome_color_token: 'outcome-win',
  score_label: '87 - 62',
  dominance_flag: false,
  had_bot_teammate: false,
  map_ui: 'Aquarius',
  map_id: undefined,
  mode_ui: 'Slayer',
  playlist_label: 'Classée',
  performance_display: '76',
  performance_color: undefined,
  performance_color_token: 'perf-tier-2',
  is_excluded: false,
  is_ranked: false,
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
    // tier_label baké "Diamond 1" (EN, cas H5) localisé en FR → "Diamant 1",
    // affiché 2× : libellé du rang + label bas-gauche de la barre.
    expect(screen.getAllByText('Diamant 1').length).toBeGreaterThan(0)
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
    const noImage: MatchViewHeader = { ...baseHeader, map_image_url: undefined }
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
      tier_label: undefined,
      numeric_value: undefined,
      delta_value: undefined,
      icon_url: undefined,
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

  it('clic sur "Exclure" ouvre le dialogue de confirmation et ne déclenche pas la mutation immédiatement', () => {
    setExclusionMutateMock.mockClear()
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
    // Avant clic : un seul bouton "Exclure" (celui du header).
    fireEvent.click(screen.getByRole('button', { name: 'Exclure' }))
    expect(screen.getByRole('alertdialog')).toBeInTheDocument()
    expect(screen.getByText('Exclure ce match ?')).toBeInTheDocument()
    expect(setExclusionMutateMock).not.toHaveBeenCalled()
  })

  it('confirmation dans le dialogue déclenche la mutation avec excluded=true', () => {
    setExclusionMutateMock.mockClear()
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
    fireEvent.click(screen.getByRole('button', { name: 'Exclure' }))
    // Après ouverture du dialog : deux boutons "Exclure" — header + footer.
    // Le footer (dans le dialog) a `variant=destructive` mais aria-name reste "Exclure".
    const dialog = screen.getByRole('alertdialog')
    const confirmBtn = Array.from(dialog.querySelectorAll('button')).find(
      (b) => b.textContent?.trim() === 'Exclure',
    )
    expect(confirmBtn).toBeDefined()
    fireEvent.click(confirmBtn!)
    expect(setExclusionMutateMock).toHaveBeenCalledWith(
      { matchId: 'm1', excluded: true },
      expect.any(Object),
    )
  })

  it('match classé (is_ranked=true) : bouton "Exclure" désactivé avec tooltip explicite', () => {
    setExclusionMutateMock.mockClear()
    const ranked: MatchViewHeader = { ...baseHeader, is_ranked: true, is_excluded: false }
    renderWithQueryClient(
      <MatchHeaderCard
        header={ranked}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    const btn = screen.getByRole('button', { name: 'Exclure' })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('title', expect.stringMatching(/classés/i))
    fireEvent.click(btn)
    expect(setExclusionMutateMock).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('match classé déjà exclu : bouton "Réactiver" autorisé', () => {
    const rankedExcluded: MatchViewHeader = { ...baseHeader, is_ranked: true, is_excluded: true }
    renderWithQueryClient(
      <MatchHeaderCard
        header={rankedExcluded}
        rank={baseRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    const btn = screen.getByRole('button', { name: 'Réactiver' })
    expect(btn).not.toBeDisabled()
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

describe('MatchHeaderCard — barre de progression du rang', () => {
  // Régression du bug "tout vert" : sur LUSR (échelle legacy, sous-paliers de
  // largeur ≠ 50), progresser DANS un sous-palier doit laisser la base bleue
  // visible. L'ancienne reconstruction `progress_pct - delta/50` la mettait à 0.
  it('LUSR — progression dans le sous-palier : la base (avant match) reste visible', () => {
    // Platine LUSR [1600,1800], 2 sous-paliers de 100 pts. 1770 = Platine II
    // (sous-palier [1700,1800]). +30 → avant = 1740, toujours dans le même
    // sous-palier → base = 40 %, delta = 30 %. (Ancien code : base = 0.)
    const lusrRank: MatchViewRank = {
      rating_type: 'LUSR',
      tier_label: 'Platinum 2',
      numeric_value: 1770,
      delta_value: 30,
      icon_url: undefined,
    }
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={lusrRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    const base = screen.getByTestId('rank-progress-base')
    expect(base.style.width).toBe('40%')
    const delta = screen.getByTestId('rank-progress-delta')
    expect(delta.style.width).toBe('30%')
    expect(delta.style.left).toBe('40%')
  })

  it('LUSR — franchissement de sous-palier : base à 0 (barre qui se remplit du bas)', () => {
    // 1710 = Platine II, +30 → avant = 1680 (Platine I, sous-palier inférieur).
    // "avant" sous la borne du sous-palier courant → base = 0, tout en delta.
    const lusrRank: MatchViewRank = {
      rating_type: 'LUSR',
      tier_label: 'Platinum 2',
      numeric_value: 1710,
      delta_value: 30,
      icon_url: undefined,
    }
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={lusrRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.getByTestId('rank-progress-base').style.width).toBe('0%')
    expect(screen.getByTestId('rank-progress-delta').style.width).toBe('10%')
  })

  it('Onyx (palier ouvert) : pas de barre de progression', () => {
    const onyxRank: MatchViewRank = {
      rating_type: 'CSR',
      tier_label: 'Onyx 1600',
      numeric_value: 1600,
      delta_value: 12,
      icon_url: undefined,
    }
    renderWithQueryClient(
      <MatchHeaderCard
        header={baseHeader}
        rank={onyxRank}
        matchId="m1"
        playerSlug="MonGT"
        matchTitle="Slayer sur Aquarius"
        locale="fr"
      />,
    )
    expect(screen.queryByTestId('rank-progress-base')).toBeNull()
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
    expect(screen.getByText(/Sortir du contexte ↩/)).toBeInTheDocument()
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
    fireEvent.click(screen.getByText(/Sortir du contexte ↩/))
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

  it('descriptor `recent` : compteur intégré "Matchs récents X/Y"', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 11, total_matches: 47 },
      isPending: false,
      source: 'router-state',
      contextDescriptor: { kind: 'recent' },
      navContext: { source: 'home_recent', matchIds: ['m1'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    expect(screen.getByText('Matchs récents 12/47')).toBeInTheDocument()
    // Pas de fragment "·" suivi du label brut puisque le descriptor est intégré
    expect(screen.queryByText('Match 12/47')).toBeNull()
    expect(screen.getByText(/Sortir du contexte ↩/)).toBeInTheDocument()
  })

  it('descriptor `with_player` : compteur intégré "Matchs avec X"', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 0, total_matches: 18 },
      isPending: false,
      source: 'router-state',
      contextDescriptor: { kind: 'with_player', gamertag: 'CoolMate' },
      navContext: { source: 'session', matchIds: ['m1'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    expect(screen.getByText('Matchs avec CoolMate 1/18')).toBeInTheDocument()
  })

  it('descriptor `session` : intègre la date+heure courte', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 2, total_matches: 9 },
      isPending: false,
      source: 'router-state',
      // 21:30 UTC le 7 mai 2026 — Intl FR rend "07/05/26 à 21:30" (ou heure locale ; le runner Vitest est en TZ stable)
      contextDescriptor: { kind: 'session', startTimeUtc: '2026-05-07T21:30:00Z' },
      navContext: { source: 'session', matchIds: ['m1'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="fr" />,
    )
    // On vérifie le préfixe + la date + le total — l'heure dépend de la TZ du runner
    expect(screen.getByText(/Matchs de la session du \d{2}\/\d{2}\/\d{2} à \d{2}:\d{2} 3\/9/)).toBeInTheDocument()
  })

  it('descriptor `recent` (EN) : "Recent matches X/Y" capitalisé', () => {
    resolvedRef.current = {
      data: { previous_match_id: 'p', next_match_id: 'n', current_index: 0, total_matches: 5 },
      isPending: false,
      source: 'router-state',
      contextDescriptor: { kind: 'recent' },
      navContext: { source: 'home_recent', matchIds: ['m1'] },
    }
    renderWithQueryClient(
      <MatchNavigationBar playerSlug="MonGT" matchId="m1" locale="en" />,
    )
    expect(screen.getByText('Recent matches 1/5')).toBeInTheDocument()
  })
})
