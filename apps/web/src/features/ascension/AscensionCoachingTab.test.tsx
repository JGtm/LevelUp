/**
 * Tests de structure pour AscensionCoachingTab (onglet « Entraînement »).
 *
 * Vérifient que le tab coaching :
 *   1. Rend la LayerSection « Ascension — Coaching d'amélioration »
 *   2. Compose coach proposals + PlayerProfileV3 + patterns
 *   3. Affiche le message d'absence de joueur quand currentPlayer est null
 *
 * Hooks et sous-composants mockés — on teste la composition, pas le détail.
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

// ── Profile ───────────────────────────────────────────────────────────────

vi.mock('./profile/queries', () => ({
  useActiveCampaign: () => ({ data: null }),
  usePlayerProfile: () => ({ data: null, isLoading: false, isError: false }),
}))
vi.mock('./profile/PlayerProfileV3', () => ({
  PlayerProfileV3: () => <div data-testid="player-profile-v3" />,
}))

// ── Patterns ──────────────────────────────────────────────────────────────

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

// ── Campaign ──────────────────────────────────────────────────────────────

vi.mock('./campaign/CampaignTracker', () => ({
  CampaignTracker: () => <div data-testid="campaign-tracker" />,
}))
vi.mock('./campaign/StartCampaignModal', () => ({
  StartCampaignModal: () => null,
}))

// Import after mocks
import { AscensionCoachingTab } from './AscensionCoachingTab'

describe('AscensionCoachingTab — composition (couche Coaching)', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.currentPlayer = {
      player_slug: 'demo-player',
      gamertag: 'DemoPlayer',
    }
    mockShellState.locale = 'fr'
  })

  it('renders the coaching LayerSection header', () => {
    render(<AscensionCoachingTab />)
    expect(screen.getByText(/Ascension — Coaching d'amélioration/i)).toBeInTheDocument()
  })

  it('renders the Coach proposals card', () => {
    render(<AscensionCoachingTab />)
    expect(screen.getByTestId('coach-proposals')).toBeInTheDocument()
  })

  it('renders PlayerProfileV3', () => {
    render(<AscensionCoachingTab />)
    expect(screen.getByTestId('player-profile-v3')).toBeInTheDocument()
  })

  it('does NOT render the Prestige objectives/arcs sections (moved to ProfileTab)', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByText(/Mes objectifs actifs/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Mes arcs en cours/i)).not.toBeInTheDocument()
  })

  it('does not render CampaignTracker when no active campaign', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByTestId('campaign-tracker')).not.toBeInTheDocument()
  })

  it('does not render patterns sections when usePatterns returns empty', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByTestId('pattern-context-grid')).not.toBeInTheDocument()
    expect(screen.queryByTestId('behavior-alerts')).not.toBeInTheDocument()
    expect(screen.queryByTestId('lever-list')).not.toBeInTheDocument()
  })

  it('switches to English copy when locale is en', () => {
    mockShellState.locale = 'en'
    render(<AscensionCoachingTab />)
    expect(screen.getByText(/Ascension — Improvement coaching/i)).toBeInTheDocument()
  })

  it('shows "select a player" message when currentPlayer is null', () => {
    mockShellState.currentPlayer = null
    render(<AscensionCoachingTab />)
    expect(
      screen.getByText(/Sélectionne un joueur pour voir ton entraînement/i),
    ).toBeInTheDocument()
  })
})
