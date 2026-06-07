/**
 * Tests de structure pour AscensionProfileTab.
 *
 * Vérifient que le tab orchestrateur :
 *   1. Compose les deux LayerSection (Prestige + Ascension) dans le bon ordre
 *   2. Place les bons composants enfants dans la bonne couche
 *   3. Affiche le message d'absence de joueur quand currentPlayer est null
 *
 * Tous les hooks externes sont mockés — le test se concentre sur la
 * composition, pas sur le comportement des sous-composants (eux-mêmes
 * testés séparément : tips-ticker, PlayerProfileV3, etc.).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'

// ── Stores ────────────────────────────────────────────────────────────────

const mockShellState = {
  currentPlayer: { player_slug: 'demo-player', gamertag: 'DemoPlayer' } as
    | { player_slug: string; gamertag: string }
    | null,
  locale: 'fr' as 'fr' | 'en',
}

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) =>
    selector(mockShellState),
}))

// ── Settings ──────────────────────────────────────────────────────────────

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { coach_proactive_mode: false } }),
}))

// ── Coach ─────────────────────────────────────────────────────────────────

vi.mock('@/features/coach/CoachProposalsCard', () => ({
  CoachProposalsCard: () => <div data-testid="coach-proposals" />,
}))
vi.mock('@/features/coach/i18n', () => ({
  getCoachStrings: () => ({}),
}))

// ── Profile (Ascension) ───────────────────────────────────────────────────

vi.mock('./profile/queries', () => ({
  useActiveCampaign: () => ({ data: null }),
  usePlayerProfile: () => ({ data: null, isLoading: false, isError: false }),
}))
vi.mock('./profile/PlayerProfileV3', () => ({
  PlayerProfileV3: () => <div data-testid="player-profile-v3" />,
}))

// ── Patterns (Ascension) ──────────────────────────────────────────────────

vi.mock('./queries', () => ({
  usePatterns: () => ({ data: null, isLoading: false }),
}))
vi.mock('./PatternContextGrid', () => ({
  PatternContextGrid: () => <div data-testid="pattern-context-grid" />,
}))
vi.mock('./SquadVsSoloCard', () => ({
  SquadVsSoloCard: () => <div data-testid="squad-vs-solo" />,
}))
vi.mock('./BehaviorAlertList', () => ({
  BehaviorAlertList: () => <div data-testid="behavior-alerts" />,
}))
vi.mock('./LeverList', () => ({
  LeverList: () => <div data-testid="lever-list" />,
}))

// ── Campaign (Ascension) ──────────────────────────────────────────────────

vi.mock('./campaign/CampaignTracker', () => ({
  CampaignTracker: () => <div data-testid="campaign-tracker" />,
}))
vi.mock('./campaign/StartCampaignModal', () => ({
  StartCampaignModal: () => null,
}))

// ── Prestige (challenges + arcs) ──────────────────────────────────────────

vi.mock('@/features/prestige/hooks', () => ({
  useChallenges: () => ({
    data: { challenges: [] },
    isLoading: false,
    isError: false,
  }),
  useArcs: () => ({ data: { arcs: [] }, isLoading: false }),
  useAbandonChallenge: () => ({ mutate: vi.fn(), isPending: false }),
}))
vi.mock('@/features/prestige/components/ChallengeCard', () => ({
  ChallengeCard: () => <div data-testid="challenge-card" />,
}))
vi.mock('@/features/prestige/components/ArcSummary', () => ({
  ArcSummary: () => <div data-testid="arc-summary" />,
}))
vi.mock('@/features/prestige/components/CreateChallengeForm', () => ({
  CreateChallengeForm: () => null,
}))
vi.mock('@/features/prestige/components/CreateArcForm', () => ({
  CreateArcForm: () => null,
}))

// ── UI ────────────────────────────────────────────────────────────────────

vi.mock('@/components/ui/tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

// Import after mocks
import { AscensionProfileTab } from './AscensionProfileTab'

describe('AscensionProfileTab — composition', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.currentPlayer = {
      player_slug: 'demo-player',
      gamertag: 'DemoPlayer',
    }
    mockShellState.locale = 'fr'
  })

  it('renders both LayerSection headers (Prestige + Ascension)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Prestige — Objectifs et arcs/i)).toBeInTheDocument()
    expect(screen.getByText(/Ascension — Coaching d'amélioration/i)).toBeInTheDocument()
  })

  it('places Prestige layer before Ascension layer in DOM order', () => {
    render(<AscensionProfileTab />)
    const prestige = screen.getByText(/Prestige — Objectifs et arcs/i)
    const ascension = screen.getByText(/Ascension — Coaching d'amélioration/i)
    // Position bitmask : Node.DOCUMENT_POSITION_FOLLOWING (4) = ascension is after prestige
    const position = prestige.compareDocumentPosition(ascension)
    expect(position & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('renders the Coach proposals card inside the Ascension layer', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByTestId('coach-proposals')).toBeInTheDocument()
  })

  it('renders PlayerProfileV3 (Ascension layer)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByTestId('player-profile-v3')).toBeInTheDocument()
  })

  it('shows "Mes objectifs actifs" section title (Prestige layer)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Mes objectifs actifs/i)).toBeInTheDocument()
  })

  it('shows "Mes arcs en cours" section title (Prestige layer)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Mes arcs en cours/i)).toBeInTheDocument()
  })

  it('switches to English copy when locale is en', () => {
    mockShellState.locale = 'en'
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Prestige — Objectives and arcs/i)).toBeInTheDocument()
    expect(screen.getByText(/Ascension — Improvement coaching/i)).toBeInTheDocument()
    expect(screen.getByText(/My active objectives/i)).toBeInTheDocument()
  })

  it('shows "select a player" message when currentPlayer is null', () => {
    mockShellState.currentPlayer = null
    render(<AscensionProfileTab />)
    expect(
      screen.getByText(/Sélectionne un joueur pour voir tes objectifs/i),
    ).toBeInTheDocument()
    // No layer headers when no player
    expect(screen.queryByText(/Prestige — Objectifs/i)).not.toBeInTheDocument()
  })

  it('shows the empty-state message for empty arcs (Prestige layer)', () => {
    render(<AscensionProfileTab />)
    expect(
      screen.getByText(/Aucun arc en cours. Crée ton premier arc/i),
    ).toBeInTheDocument()
  })

  it('shows the "+ Nouvel arc" create button in the empty arcs state', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/\+ Nouvel arc/i)).toBeInTheDocument()
  })

  it('shows the empty-state message for empty challenges (Prestige layer)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Aucun objectif libre actif/i)).toBeInTheDocument()
  })

  it('does not render CampaignTracker when no active campaign', () => {
    render(<AscensionProfileTab />)
    expect(screen.queryByTestId('campaign-tracker')).not.toBeInTheDocument()
  })

  it('renders the "+ Nouvel objectif" button in the objectives section', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByRole('button', { name: /\+ Nouvel objectif/i })).toBeInTheDocument()
  })

  it('does not render patterns sections when usePatterns returns empty', () => {
    render(<AscensionProfileTab />)
    expect(screen.queryByTestId('pattern-context-grid')).not.toBeInTheDocument()
    expect(screen.queryByTestId('behavior-alerts')).not.toBeInTheDocument()
    expect(screen.queryByTestId('lever-list')).not.toBeInTheDocument()
  })
})
