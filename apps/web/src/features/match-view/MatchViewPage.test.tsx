/**
 * MatchViewPage — état dédié 404 match_not_found (V72-15.4).
 *
 * Le backend ne fait plus AUCUN fetch live vers l'API du titre quand un match est
 * absent du substrat local (fallback LIVE retiré le 2026-07-25, cf. BACKLOG
 * "Retirer le fallback LIVE du Match view") : il renvoie un 404 typé
 * {code: "match_not_found"}. Ce test vérifie que la page affiche l'état "pas
 * encore synchronisé" (PageUnavailable dédié) plutôt que l'écran d'erreur
 * générique, et que les autres branches d'erreur (ADR 0029) restent inchangées.
 *
 * Couvre aussi la structure de l'onglet Général : le bloc Médias est SORTI de la
 * grille Médailles/Citations — dernier bloc, seul sur sa rangée, pleine largeur
 * (règle produit : aucun bloc jamais seul à largeur partielle). Les enfants lourds
 * (charts ECharts, header) sont mockés — seule la structure des rangées est testée.
 */
import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchViewPage } from './MatchViewPage'

const hoisted = vi.hoisted(() => ({
  matchView: {
    data: undefined as unknown,
    isPending: false,
    isError: true,
    error: { code: 'match_not_found', message: 'match introuvable : m1' } as unknown,
    refetch: vi.fn(),
  },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player', matchId: 'm1' }),
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
    useRouter: () => ({ history: { length: 1, back: vi.fn() } }),
  }
})

vi.mock('./queries', () => ({
  useMatchView: () => hoisted.matchView,
  // Les deux calques du film sont best-effort : ces tests portent sur la structure
  // de la page, pas sur eux — `data` undefined reproduit le cas d'un titre sans film.
  useMatchObjectiveEvents: () => ({ data: undefined }),
  useMatchPositions: () => ({ data: undefined }),
}))

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { friend_gamertags: [] } }),
}))

// availableTitles vide → useCapability (FeatureGate) fail-open : le bloc Médias est rendu.
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (
    selector: (s: { locale: 'fr' | 'en'; availableTitles: unknown[]; currentTitleSlug: string }) => unknown,
  ) => selector({ locale: 'fr', availableTitles: [], currentTitleSlug: 'halo_infinite' }),
}))

// Enfants de l'onglet Général mockés : on ne teste ici que la structure des rangées.
vi.mock('./MatchHeader', () => ({
  MatchBreadcrumb: () => <div data-testid="breadcrumb" />,
  MatchNavigationBar: () => <div data-testid="navbar" />,
  MatchHeaderCard: () => <div data-testid="header-card" />,
}))
vi.mock('./MatchStatCards', () => ({
  MatchSummaryCardsSection: () => <div data-testid="summary-cards" />,
}))
vi.mock('./MatchSummaryCharts', () => ({
  MatchKdaExpectedChart: () => <div data-testid="chart-kda" />,
  MatchSpreeChart: () => <div data-testid="chart-spree" />,
  MatchSummaryRadarChart: () => <div data-testid="chart-radar" />,
}))
vi.mock('./MatchFragCard', () => ({
  MatchFragCard: () => <div data-testid="frag-card" />,
}))
vi.mock('./MatchSummaryMedalsAndCitations', () => ({
  MatchMedalsSection: () => <div data-testid="medals" />,
  MatchCitationsSection: () => <div data-testid="citations" />,
  MatchNativeCommendationsSection: () => <div data-testid="native-commendations" />,
}))
vi.mock('./MatchMediaTab', () => ({
  MatchMediaTab: () => <div data-testid="media-tab" />,
}))

describe('MatchViewPage — match_not_found (pas encore synchronisé)', () => {
  it('affiche l\'état dédié « pas encore synchronisé », pas l\'erreur générique', () => {
    hoisted.matchView.error = { code: 'match_not_found', message: 'match introuvable : m1' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText('Match pas encore synchronisé')).toBeInTheDocument()
    expect(
      screen.getByText(/n'est pas encore présent dans la base locale/),
    ).toBeInTheDocument()
    // Pas l'écran d'erreur générique (pageErrorTitle) ni le bouton Réessayer.
    expect(screen.queryByText('Match introuvable ou erreur de chargement.')).not.toBeInTheDocument()
    expect(screen.queryByText('Réessayer')).not.toBeInTheDocument()
  })

  it('propose des actions de navigation (Accueil / Précédent / Mes matchs)', () => {
    hoisted.matchView.error = { code: 'match_not_found', message: 'match introuvable : m1' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByRole('button', { name: 'Accueil' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Précédent' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Mes matchs' })).toBeInTheDocument()
  })

  it('conserve la branche existante match_not_participant (ADR 0029, non régressée)', () => {
    hoisted.matchView.error = { code: 'match_not_participant', message: 'non participant' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText('Match indisponible')).toBeInTheDocument()
    expect(screen.queryByText('Match pas encore synchronisé')).not.toBeInTheDocument()
  })
})

describe('MatchViewPage — onglet Général : structure des rangées', () => {
  it('le bloc Médias est HORS de la grille Médailles/Citations, dernier et pleine largeur', () => {
    hoisted.matchView.isError = false
    hoisted.matchView.error = null
    hoisted.matchView.data = {
      header: { map_ui: 'Forest', mode_ui: 'Slayer', start_time_label: null },
      rank: null,
      summary_tab: { kpis: {}, expected_stats: null, medals: [], citations: [] },
      combat_tab: {},
      team_tab: {},
      media_tab: { media_items: [] },
      citations_tab: { native_commendations: [] },
      radar: null,
    }
    renderWithProviders(<MatchViewPage />)

    const medals = screen.getByTestId('medals')
    const grid = medals.parentElement as HTMLElement
    expect(grid.className).toContain('grid')
    expect(grid.className).not.toContain('lg:grid-cols-3')
    // Citations partagent la grille des Médailles.
    expect(grid.contains(screen.getByTestId('citations'))).toBe(true)
    // Le bloc Médias n'est PAS dans cette grille…
    const mediaTab = screen.getByTestId('media-tab')
    expect(grid.contains(mediaTab)).toBe(false)
    // …c'est le DERNIER bloc de la pile summary, frère direct de la grille (pleine largeur).
    const summaryStack = grid.parentElement as HTMLElement
    const mediaCard = mediaTab.closest('.rounded-lg') as HTMLElement
    expect(mediaCard.parentElement).toBe(summaryStack)
    expect(summaryStack.lastElementChild).toBe(mediaCard)
    expect(screen.getByText('Médias')).toBeInTheDocument()
  })
})
