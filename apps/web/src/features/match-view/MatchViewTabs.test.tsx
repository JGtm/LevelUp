/**
 * Page match — 4 onglets « Général / Chronologie / Armes et terrain / Joueurs ».
 *
 * Le troisième onglet a été ajouté le 2026-09-19 (plan
 * `.ai/V7.5/PLAN_AJUSTEMENTS_PRE_V75_2026-09-19.md`, lot 2) sous le nom « Contrôle » : il
 * prend à Chronologie les trois blocs qui disent QUI A TENU QUOI — bilan d'équipement,
 * contrôle des armes, occupation du terrain. Il s'appelle « Armes et terrain » (`arsenal`)
 * depuis le 2026-09-22 et a repris de Général la répartition des frags et la distance des
 * frags : un onglet, un axe de lecture.
 *
 * Couvre : la rétro-compat des deep-links (`?tab=details` → Chronologie et
 * `?tab=control` → Armes et terrain, résolus au décodage par le schéma de recherche de la
 * route, sans redirection), la répartition des sections entre les onglets, les titres de
 * section de Général et d'Armes et terrain, et le fait que les deux calques de film ne
 * sont tirés que lorsque l'onglet qui les affiche est actif — les événements d'objectif sur
 * Chronologie, les positions sur Armes et terrain.
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

vi.mock('@/features/friends/queries', () => ({
  useFriendGamertags: () => [],
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
vi.mock('@/features/engagement/EngagementMatchSection', () => ({
  EngagementMatchSection: () => <div data-testid="engagement" />,
}))

// Feuilles mockées — onglet « Armes et terrain » : les deux blocs venus de Général le
// 2026-09-22, puis les trois blocs déplacés depuis Chronologie le 2026-09-19.
vi.mock('./MatchFragCard', () => ({ MatchFragCard: () => <div data-testid="frag-card" /> }))
vi.mock('./MatchKillDistanceSection', () => ({
  MatchKillDistanceSection: () => <div data-testid="kill-distance" />,
}))
vi.mock('./MatchPositionsHeatmap', () => ({
  MatchPositionsHeatmap: () => <div data-testid="positions-heatmap" />,
}))
vi.mock('@/features/match-replay/MatchEquipmentUsageSection', () => ({
  MatchEquipmentUsageSection: () => <div data-testid="equipment-usage" />,
}))
vi.mock('@/features/match-replay/MatchPadControlSection', () => ({
  MatchPadControlSection: () => <div data-testid="pad-control" />,
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
// Titres de section posés le 2026-09-22 (gabarit `DetailSection`).
const SECTION_COMBAT = 'Combat'
const SECTION_REWARDS = 'Récompenses'
const SECTION_MEDIA = 'Médias'
const SECTION_KILLS_WEAPONS = 'Frags et armes'
const SECTION_EQUIPMENT_TERRAIN = 'Équipement et terrain'

beforeEach(() => {
  hoisted.search = {}
  hoisted.objectiveEventsCalls = []
  hoisted.positionsCalls = []
})

describe('resolveMatchViewTab — ids canoniques et alias', () => {
  it('accepte les quatre ids canoniques', () => {
    expect(resolveMatchViewTab('summary')).toBe('summary')
    expect(resolveMatchViewTab('chronology')).toBe('chronology')
    expect(resolveMatchViewTab('arsenal')).toBe('arsenal')
    expect(resolveMatchViewTab('players')).toBe('players')
  })

  it('résout l\'ancien deep-link `details` vers Chronologie', () => {
    expect(resolveMatchViewTab('details')).toBe('chronology')
  })

  it('résout l\'ancien deep-link `control` vers Armes et terrain', () => {
    expect(resolveMatchViewTab('control')).toBe('arsenal')
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

  it('`tab=control` est accepté et résolu vers `arsenal` (pas de redirection)', () => {
    expect(parse({ tab: 'control' }).tab).toBe('arsenal')
  })

  it('laisse passer les quatre ids canoniques', () => {
    expect(parse({ tab: 'summary' }).tab).toBe('summary')
    expect(parse({ tab: 'chronology' }).tab).toBe('chronology')
    expect(parse({ tab: 'arsenal' }).tab).toBe('arsenal')
    expect(parse({ tab: 'players' }).tab).toBe('players')
  })

  it('`tab` absent reste absent (aucun `?tab=` ajouté aux liens)', () => {
    expect(parse({}).tab).toBeUndefined()
  })

  it('une valeur inconnue retombe sur `summary`', () => {
    expect(parse({ tab: 'zzz' }).tab).toBe('summary')
  })
})

describe('MatchViewPage — barre des 4 onglets', () => {
  it('affiche Général, Chronologie, Armes et terrain et Joueurs (FR)', () => {
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByRole('button', { name: 'Général' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Chronologie' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Armes et terrain' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Joueurs' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Détails' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Contrôle' })).not.toBeInTheDocument()
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

  it('onglet Général : ses trois titres de section, et la bande de KPI sans titre', () => {
    renderWithProviders(<MatchViewPage />)

    for (const title of [SECTION_COMBAT, SECTION_REWARDS, SECTION_MEDIA]) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
    // La bande de KPI ouvre l'onglet SANS titre au-dessus : elle est le premier enfant de
    // la pile, avant la section « Combat » (comme l'accueil).
    const cards = screen.getByTestId('summary-cards')
    const stack = cards.parentElement as HTMLElement
    expect(stack.firstElementChild).toBe(cards)
    expect(screen.getByText(SECTION_COMBAT).closest('section')?.parentElement).toBe(stack)
    // Les deux blocs partis vers « Armes et terrain » le 2026-09-22 ne sont plus ici.
    for (const id of ['frag-card', 'kill-distance']) {
      expect(screen.queryByTestId(id)).not.toBeInTheDocument()
    }
  })

  it('onglet Chronologie : déroulé du match seul, avec ses blocs', () => {
    hoisted.search = { tab: 'chronology' }
    renderWithProviders(<MatchViewPage />)

    expect(screen.getByText(SECTION_FLOW)).toBeInTheDocument()
    for (const id of ['impact-badges', 'kd-cumul', 'score-curve', 'tug-of-war', 'cadence', 'engagement']) {
      expect(screen.getByTestId(id)).toBeInTheDocument()
    }
    // Les trois blocs d'« Armes et terrain » ont quitté cet onglet le 2026-09-19.
    for (const id of ['positions-heatmap', 'equipment-usage', 'pad-control']) {
      expect(screen.queryByTestId(id)).not.toBeInTheDocument()
    }
    expect(screen.queryByTestId('summary-cards')).not.toBeInTheDocument()
    for (const title of [SECTION_DUELS, SECTION_SCOREBOARD, SECTION_ENCOUNTERS]) {
      expect(screen.queryByText(title)).not.toBeInTheDocument()
    }
  })

  it('onglet Armes et terrain : frags, distance, équipement, armes et terrain, et rien d’autre', () => {
    hoisted.search = { tab: 'arsenal' }
    renderWithProviders(<MatchViewPage />)

    for (const id of ['frag-card', 'kill-distance', 'equipment-usage', 'pad-control', 'positions-heatmap']) {
      expect(screen.getByTestId(id)).toBeInTheDocument()
    }
    expect(screen.queryByText(SECTION_FLOW)).not.toBeInTheDocument()
    expect(screen.queryByTestId('summary-cards')).not.toBeInTheDocument()
    expect(screen.queryByTestId('kd-cumul')).not.toBeInTheDocument()
  })

  it('onglet Armes et terrain : deux titres de section, chacun sur ses blocs', () => {
    hoisted.search = { tab: 'arsenal' }
    renderWithProviders(<MatchViewPage />)

    const kills = screen.getByText(SECTION_KILLS_WEAPONS).closest('section') as HTMLElement
    for (const id of ['frag-card', 'kill-distance']) {
      expect(kills.contains(screen.getByTestId(id))).toBe(true)
    }
    const terrain = screen.getByText(SECTION_EQUIPMENT_TERRAIN).closest('section') as HTMLElement
    for (const id of ['equipment-usage', 'pad-control', 'positions-heatmap']) {
      expect(terrain.contains(screen.getByTestId(id))).toBe(true)
    }
    // Aucun titre de l'onglet Général n'a suivi les deux blocs déplacés.
    for (const title of [SECTION_COMBAT, SECTION_REWARDS, SECTION_MEDIA]) {
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

describe('MatchViewPage — chaque calque de film est tiré par le SEUL onglet qui le consomme', () => {
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

  it('onglet Chronologie : les événements d’objectif seuls', () => {
    hoisted.search = { tab: 'chronology' }
    renderWithProviders(<MatchViewPage />)

    expect(hoisted.objectiveEventsCalls[0]).toEqual(['test-player', 'm1', true])
    expect(hoisted.positionsCalls[0]).toEqual(['test-player', 'm1', false])
  })

  it('onglet Armes et terrain : les positions seules', () => {
    hoisted.search = { tab: 'arsenal' }
    renderWithProviders(<MatchViewPage />)

    expect(hoisted.objectiveEventsCalls[0]).toEqual(['test-player', 'm1', false])
    expect(hoisted.positionsCalls[0]).toEqual(['test-player', 'm1', true])
  })
})
