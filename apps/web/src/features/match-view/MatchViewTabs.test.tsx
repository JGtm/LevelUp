/**
 * Page match — 3 onglets « Général / Chronologie / Joueurs » (2026-08-24).
 *
 * Couvre : la rétro-compat des deep-links (`?tab=details` → Chronologie, résolu au
 * décodage par le schéma de recherche de la route, sans redirection), la
 * répartition des sections entre les deux nouveaux onglets, et le fait que les
 * deux calques de film (événements d'objectif, positions) ne sont tirés que
 * lorsque l'onglet qui les affiche est actif.
 *
 * Les feuilles lourdes (charts ECharts, tables) sont mockées : seule la structure
 * des onglets est testée ici.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { MatchViewPage } from './MatchViewPage'
import { resolveMatchViewTab } from './tabs'
import { Route as MatchLayoutRoute } from '@/routes/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId'

const hoisted = vi.hoisted(() => ({
  search: {} as { tab?: string },
  objectiveEventsCalls: [] as unknown[][],
  positionsCalls: [] as unknown[][],
  matchView: {
    data: {
      header: { map_ui: 'Forest', mode_ui: 'Slayer', start_time_label: null, replay_available: false },
      rank: null,
      summary_tab: { kpis: {}, expected_stats: null, medals: [], citations: [] },
      combat_tab: {},
      team_tab: {},
      media_tab: { media_items: [] },
      citations_tab: { native_commendations: [] },
      radar: null,
    } as unknown,
    isPending: false,
    isError: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player', matchId: 'm1' }),
    useSearch: () => hoisted.search,
    useNavigate: () => vi.fn(),
    useRouter: () => ({ history: { length: 1, back: vi.fn() } }),
  }
})

vi.mock('./queries', () => ({
  useMatchView: () => hoisted.matchView,
  useMatchObjectiveEvents: (...args: unknown[]) => {
    hoisted.objectiveEventsCalls.push(args)
    return { data: undefined }
  },
  useMatchPositions: (...args: unknown[]) => {
    hoisted.positionsCalls.push(args)
    return { data: undefined }
  },
}))

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { friend_gamertags: [] } }),
}))

// availableTitles vide → useCapability (FeatureGate) fail-open.
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (
    selector: (s: { locale: 'fr' | 'en'; availableTitles: unknown[]; currentTitleSlug: string }) => unknown,
  ) => selector({ locale: 'fr', availableTitles: [], currentTitleSlug: 'halo_infinite' }),
}))

// Feuilles mockées — en-tête et onglet Général.
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
vi.mock('./MatchFragCard', () => ({ MatchFragCard: () => <div data-testid="frag-card" /> }))
vi.mock('./MatchSummaryMedalsAndCitations', () => ({
  MatchMedalsSection: () => <div data-testid="medals" />,
  MatchCitationsSection: () => <div data-testid="citations" />,
  MatchNativeCommendationsSection: () => <div data-testid="native-commendations" />,
}))
vi.mock('./MatchMediaTab', () => ({ MatchMediaTab: () => <div data-testid="media-tab" /> }))

// Feuilles mockées — onglet Chronologie.
vi.mock('./MatchImpactBadgesBar', () => ({
  MatchImpactBadgesBar: () => <div data-testid="impact-badges" />,
}))
vi.mock('./MatchKDCumulChart', () => ({ MatchKDCumulChart: () => <div data-testid="kd-cumul" /> }))
vi.mock('./MatchScoreCurveChart', () => ({
  MatchScoreCurveChart: () => <div data-testid="score-curve" />,
}))
vi.mock('./MatchTugOfWarChart', () => ({ MatchTugOfWarChart: () => <div data-testid="tug-of-war" /> }))
vi.mock('./MatchCadenceChart', () => ({ MatchCadenceChart: () => <div data-testid="cadence" /> }))
vi.mock('./MatchPositionsHeatmap', () => ({
  MatchPositionsHeatmap: () => <div data-testid="positions-heatmap" />,
}))
vi.mock('@/features/engagement/EngagementMatchSection', () => ({
  EngagementMatchSection: () => <div data-testid="engagement" />,
}))

// Feuilles mockées — onglet Joueurs.
vi.mock('./MatchNemesisCards', () => ({ MatchNemesisCards: () => <div data-testid="nemesis" /> }))
vi.mock('./MatchAntagonistChart', () => ({
  MatchAntagonistChart: () => <div data-testid="antagonist" />,
}))
vi.mock('./MatchFragDiffChart', () => ({ MatchFragDiffChart: () => <div data-testid="frag-diff" /> }))
vi.mock('./MatchScoreboard', () => ({ MatchScoreboard: () => <div data-testid="scoreboard" /> }))
vi.mock('./MatchEncountersTable', () => ({
  MatchEncountersTable: () => <div data-testid="encounters" />,
}))

const SECTION_FLOW = 'Déroulé du match'
const SECTION_DUELS = 'Duels & confrontations'
const SECTION_SCOREBOARD = 'Tableau des scores'
const SECTION_ENCOUNTERS = 'Historique des rencontres'

beforeEach(() => {
  hoisted.search = {}
  hoisted.objectiveEventsCalls = []
  hoisted.positionsCalls = []
})

describe('resolveMatchViewTab — ids canoniques et alias', () => {
  it('accepte les trois ids canoniques', () => {
    expect(resolveMatchViewTab('summary')).toBe('summary')
    expect(resolveMatchViewTab('chronology')).toBe('chronology')
    expect(resolveMatchViewTab('players')).toBe('players')
  })

  it('résout l\'ancien deep-link `details` vers Chronologie', () => {
    expect(resolveMatchViewTab('details')).toBe('chronology')
  })

  it('retombe sur `summary` pour une valeur inconnue ou absente', () => {
    expect(resolveMatchViewTab('nope')).toBe('summary')
    expect(resolveMatchViewTab(undefined)).toBe('summary')
    expect(resolveMatchViewTab(null)).toBe('summary')
    expect(resolveMatchViewTab(42)).toBe('summary')
  })
})

describe('schéma de recherche de la route match', () => {
  const parse = (search: Record<string, unknown>) =>
    (MatchLayoutRoute.options.validateSearch as { parse: (v: unknown) => { tab?: string } }).parse(
      search,
    )

  it('`tab=details` est accepté et résolu vers `chronology` (pas de redirection)', () => {
    expect(parse({ tab: 'details' }).tab).toBe('chronology')
  })

  it('laisse passer les trois ids canoniques', () => {
    expect(parse({ tab: 'summary' }).tab).toBe('summary')
    expect(parse({ tab: 'chronology' }).tab).toBe('chronology')
    expect(parse({ tab: 'players' }).tab).toBe('players')
  })

  it('`tab` absent reste absent (aucun `?tab=` ajouté aux liens)', () => {
    expect(parse({}).tab).toBeUndefined()
  })

  it('une valeur inconnue retombe sur `summary`', () => {
    expect(parse({ tab: 'zzz' }).tab).toBe('summary')
  })
})

describe('MatchViewPage — barre des 3 onglets', () => {
  it('affiche Général, Chronologie et Joueurs (FR)', () => {
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByRole('button', { name: 'Général' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Chronologie' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Joueurs' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Détails' })).not.toBeInTheDocument()
  })
})

describe('MatchViewPage — contenu par onglet', () => {
  it('onglet Général (défaut) : aucune section de Chronologie ni de Joueurs', () => {
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByTestId('summary-cards')).toBeInTheDocument()
    for (const title of [SECTION_FLOW, SECTION_DUELS, SECTION_SCOREBOARD, SECTION_ENCOUNTERS]) {
      expect(screen.queryByText(title)).not.toBeInTheDocument()
    }
  })

  it('onglet Chronologie : déroulé du match seul, avec ses blocs', () => {
    hoisted.search = { tab: 'chronology' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText(SECTION_FLOW)).toBeInTheDocument()
    for (const id of ['impact-badges', 'kd-cumul', 'score-curve', 'tug-of-war', 'cadence', 'positions-heatmap', 'engagement']) {
      expect(screen.getByTestId(id)).toBeInTheDocument()
    }
    expect(screen.queryByTestId('summary-cards')).not.toBeInTheDocument()
    for (const title of [SECTION_DUELS, SECTION_SCOREBOARD, SECTION_ENCOUNTERS]) {
      expect(screen.queryByText(title)).not.toBeInTheDocument()
    }
  })

  it('onglet Joueurs : duels, tableau des scores et rencontres', () => {
    hoisted.search = { tab: 'players' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText(SECTION_DUELS)).toBeInTheDocument()
    expect(screen.getByText(SECTION_SCOREBOARD)).toBeInTheDocument()
    expect(screen.getByText(SECTION_ENCOUNTERS)).toBeInTheDocument()
    for (const id of ['nemesis', 'antagonist', 'frag-diff', 'scoreboard', 'encounters']) {
      expect(screen.getByTestId(id)).toBeInTheDocument()
    }
    expect(screen.queryByText(SECTION_FLOW)).not.toBeInTheDocument()
    expect(screen.queryByTestId('summary-cards')).not.toBeInTheDocument()
  })
})

describe('MatchViewPage — calques de film tirés seulement sur Chronologie', () => {
  it('onglet Général : les deux queries sont désactivées', () => {
    renderWithProviders(<MatchViewPage />)

    expect(hoisted.objectiveEventsCalls[0]).toEqual(['test-player', 'm1', false])
    expect(hoisted.positionsCalls[0]).toEqual(['test-player', 'm1', false])
  })

  it('onglet Joueurs : les deux queries restent désactivées', () => {
    hoisted.search = { tab: 'players' }
    renderWithProviders(<MatchViewPage />)

    expect(hoisted.objectiveEventsCalls[0]).toEqual(['test-player', 'm1', false])
    expect(hoisted.positionsCalls[0]).toEqual(['test-player', 'm1', false])
  })

  it('onglet Chronologie : les deux queries sont activées', () => {
    hoisted.search = { tab: 'chronology' }
    renderWithProviders(<MatchViewPage />)

    expect(hoisted.objectiveEventsCalls[0]).toEqual(['test-player', 'm1', true])
    expect(hoisted.positionsCalls[0]).toEqual(['test-player', 'm1', true])
  })
})
