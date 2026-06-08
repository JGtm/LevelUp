/**
 * Tests de structure pour AscensionProfileTab.
 *
 * Depuis le split en 2 onglets (2026-06-08), ce tab ne porte QUE la couche
 * Prestige (objectifs + arcs). Le coaching (proposals, profil, patterns,
 * campagne) est testé dans AscensionCoachingTab.test.tsx.
 *
 * Tous les hooks externes sont mockés — le test se concentre sur la
 * composition, pas sur le comportement des sous-composants.
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

describe('AscensionProfileTab — composition (couche Prestige seule)', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.currentPlayer = {
      player_slug: 'demo-player',
      gamertag: 'DemoPlayer',
    }
    mockShellState.locale = 'fr'
  })

  it('renders the Prestige LayerSection header', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Prestige — Objectifs et arcs/i)).toBeInTheDocument()
  })

  it('does NOT render the coaching layer (moved to AscensionCoachingTab)', () => {
    render(<AscensionProfileTab />)
    expect(screen.queryByText(/Ascension — Coaching d'amélioration/i)).not.toBeInTheDocument()
    expect(screen.queryByTestId('coach-proposals')).not.toBeInTheDocument()
    expect(screen.queryByTestId('player-profile-v3')).not.toBeInTheDocument()
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
    expect(screen.getByText(/My active objectives/i)).toBeInTheDocument()
  })

  it('shows "select a player" message when currentPlayer is null', () => {
    mockShellState.currentPlayer = null
    render(<AscensionProfileTab />)
    expect(
      screen.getByText(/Sélectionne un joueur pour voir tes objectifs/i),
    ).toBeInTheDocument()
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

  it('renders the "+ Nouvel objectif" button in the objectives section', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByRole('button', { name: /\+ Nouvel objectif/i })).toBeInTheDocument()
  })
})
