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
import { render, screen, cleanup, fireEvent } from '@testing-library/react'

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

// Fixtures mutables (déclarées avant vi.mock ; la factory est une closure
// évaluée à l'import, donc l'init est OK).
type ArcFixture = { id: string; title: string }
const mockArcs: { current: ArcFixture[] } = { current: [] }
const deleteArcMutate = vi.fn()

vi.mock('@/features/prestige/hooks', () => ({
  useChallenges: () => ({
    data: { challenges: [] },
    isLoading: false,
    isError: false,
  }),
  useArcs: () => ({ data: { arcs: mockArcs.current }, isLoading: false }),
  useAbandonChallenge: () => ({ mutate: vi.fn(), isPending: false }),
  useDeleteArc: () => ({ mutate: deleteArcMutate, isPending: false }),
  // Lot B : picker de presets (rendu seulement à l'ouverture).
  useArcPresets: () => ({ data: { presets: [], count: 0 }, isLoading: false, isError: false }),
  useAdoptArcPreset: () => ({ mutate: vi.fn(), isPending: false }),
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
    mockArcs.current = []
    deleteArcMutate.mockClear()
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
      screen.getByText(/Aucun arc en cours. Adopte un arc preset/i),
    ).toBeInTheDocument()
  })

  it('shows the "+ Nouvel arc" create button in the empty arcs state', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/\+ Nouvel arc/i)).toBeInTheDocument()
  })

  it('shows the "Parcourir les presets" button in the empty arcs state', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByRole('button', { name: /Parcourir les presets/i })).toBeInTheDocument()
  })

  it('opens the preset picker when clicking "Parcourir les presets"', () => {
    render(<AscensionProfileTab />)
    fireEvent.click(screen.getByRole('button', { name: /Parcourir les presets/i }))
    // Le picker affiche son en-tête + l'état vide (presets mock = []).
    expect(screen.getByText(/Aucun preset disponible/i)).toBeInTheDocument()
  })

  it('shows the empty-state message for empty challenges (Prestige layer)', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByText(/Aucun objectif libre actif/i)).toBeInTheDocument()
  })

  it('renders the "+ Nouvel objectif" button in the objectives section', () => {
    render(<AscensionProfileTab />)
    expect(screen.getByRole('button', { name: /\+ Nouvel objectif/i })).toBeInTheDocument()
  })

  it('ouvre la confirmation de suppression d\'arc et appelle deleteArc', () => {
    mockArcs.current = [{ id: 'arc1', title: 'Mon Arc' }]
    render(<AscensionProfileTab />)

    // Ouvre la confirmation (bouton « Supprimer » de l'élément d'arc).
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    expect(screen.getByText(/Supprimer l'arc « Mon Arc » \?/i)).toBeInTheDocument()

    // 0 objectif → un seul bouton « Supprimer » de confirmation + Annuler.
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    expect(deleteArcMutate).toHaveBeenCalledWith({ id: 'arc1', cascade: true })
  })

  it('annule la suppression sans appeler deleteArc', () => {
    mockArcs.current = [{ id: 'arc1', title: 'Mon Arc' }]
    render(<AscensionProfileTab />)

    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(deleteArcMutate).not.toHaveBeenCalled()
  })
})
