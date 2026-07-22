/**
 * Tests de structure pour AscensionObjectivesTab (onglet « Objectifs »).
 *
 * Restructuration 4 onglets (2026-07, DEC-3) : ex-AscensionProfileTab. Ce tab
 * ne porte QUE la couche Prestige (objectifs + arcs). Le profil/identité est
 * testé dans AscensionProfilTab.test.tsx, le coaching dans
 * AscensionCoachingTab.test.tsx.
 *
 * Tous les hooks externes sont mockés — le test se concentre sur la
 * composition, pas sur le comportement des sous-composants.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup, fireEvent, within } from '@testing-library/react'

// ── Stores ────────────────────────────────────────────────────────────────

const mockShellState = {
  currentPlayer: { player_slug: 'demo-player', gamertag: 'DemoPlayer' } as
    | { player_slug: string; gamertag: string }
    | null,
  locale: 'fr' as 'fr' | 'en',
  currentTitleSlug: 'halo_infinite',
}

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) =>
    selector(mockShellState),
}))

// ── Prestige (challenges + arcs) ──────────────────────────────────────────

// Fixtures mutables (déclarées avant vi.mock ; la factory est une closure
// évaluée à l'import, donc l'init est OK).
type ArcFixture = { id: string; title: string }
type ChallengeFixture = { id: string; mode: 'libre' | 'pilote'; status: string }
const mockArcs: { current: ArcFixture[] } = { current: [] }
const mockChallenges: { current: ChallengeFixture[] } = { current: [] }
const deleteArcMutate = vi.fn()
const pilotEnableMutate = vi.fn()
const pilotDisableMutate = vi.fn()
const abandonMutate = vi.fn()

const mockHistory: { current: ChallengeFixture[] } = { current: [] }

vi.mock('@/features/prestige/hooks', () => ({
  useChallenges: () => ({
    data: { challenges: mockChallenges.current },
    isLoading: false,
    isError: false,
  }),
  useChallengeHistory: () => ({
    data: { challenges: mockHistory.current },
    isLoading: false,
    isError: false,
  }),
  useArcs: () => ({ data: { arcs: mockArcs.current }, isLoading: false }),
  useAbandonChallenge: () => ({ mutate: abandonMutate, isPending: false }),
  useDeleteArc: () => ({ mutate: deleteArcMutate, isPending: false }),
  usePilotMode: () => ({
    enable: { mutate: pilotEnableMutate, isPending: false },
    disable: { mutate: pilotDisableMutate, isPending: false },
  }),
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
import { AscensionObjectivesTab, computeArcStepCounts } from './AscensionObjectivesTab'
import type { Challenge } from '@/lib/prestige'

// Fabrique un défi minimal (seuls arc_id/id/status comptent pour le décompte).
function ch(id: string, arcId: string | undefined, status: string): Challenge {
  return { id, arc_id: arcId, status } as unknown as Challenge
}

describe('computeArcStepCounts — décompte étapes complétées/total par arc (F1)', () => {
  it('compte les étapes complétées depuis les défis terminaux (actifs + historique)', () => {
    const counts = computeArcStepCounts([
      // Actifs (arc a1) : 2 en cours, aucun completed.
      ch('c1', 'a1', 'active'),
      ch('c2', 'a1', 'active'),
      // Terminaux (arc a1) : 1 complété, 1 abandonné.
      ch('c3', 'a1', 'completed'),
      ch('c4', 'a1', 'abandoned'),
    ])
    expect(counts.get('a1')).toEqual({ completed: 1, total: 4 })
  })

  it('ignore les défis détachés (arc_id absent)', () => {
    const counts = computeArcStepCounts([ch('c1', undefined, 'completed'), ch('c2', 'a1', 'completed')])
    expect(counts.has('__undefined__')).toBe(false)
    expect(counts.get('a1')).toEqual({ completed: 1, total: 1 })
  })

  it('dédoublonne un défi présent dans les deux listes (actif + historique)', () => {
    // Même id dupliqué : ne doit pas gonfler total.
    const counts = computeArcStepCounts([ch('c1', 'a1', 'completed'), ch('c1', 'a1', 'completed')])
    expect(counts.get('a1')).toEqual({ completed: 1, total: 1 })
  })
})

describe('AscensionObjectivesTab — composition (couche Prestige seule)', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.currentPlayer = {
      player_slug: 'demo-player',
      gamertag: 'DemoPlayer',
    }
    mockShellState.locale = 'fr'
    mockArcs.current = []
    mockChallenges.current = []
    mockHistory.current = []
    deleteArcMutate.mockClear()
    pilotEnableMutate.mockClear()
    pilotDisableMutate.mockClear()
    abandonMutate.mockClear()
  })

  it('renders the Prestige LayerSection header', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/Prestige — Objectifs et arcs/i)).toBeInTheDocument()
  })

  it('does NOT render the coaching layer (moved to AscensionCoachingTab)', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.queryByText(/Ascension — Coaching d'amélioration/i)).not.toBeInTheDocument()
    expect(screen.queryByTestId('coach-proposals')).not.toBeInTheDocument()
    expect(screen.queryByTestId('player-profile-v3')).not.toBeInTheDocument()
  })

  it('shows "Mes objectifs actifs" section title (Prestige layer)', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/Mes objectifs actifs/i)).toBeInTheDocument()
  })

  it('shows "Mes arcs en cours" section title (Prestige layer)', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/Mes arcs en cours/i)).toBeInTheDocument()
  })

  it('switches to English copy when locale is en', () => {
    mockShellState.locale = 'en'
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/Prestige — Objectives and arcs/i)).toBeInTheDocument()
    expect(screen.getByText(/My active objectives/i)).toBeInTheDocument()
  })

  it('shows "select a player" message when currentPlayer is null', () => {
    mockShellState.currentPlayer = null
    render(<AscensionObjectivesTab />)
    expect(
      screen.getByText(/Sélectionne un joueur pour voir tes objectifs/i),
    ).toBeInTheDocument()
    expect(screen.queryByText(/Prestige — Objectifs/i)).not.toBeInTheDocument()
  })

  it('shows the empty-state message for empty arcs (Prestige layer)', () => {
    render(<AscensionObjectivesTab />)
    expect(
      screen.getByText(/Aucun arc en cours. Adopte un arc preset/i),
    ).toBeInTheDocument()
  })

  it('shows the "+ Nouvel arc" create button in the empty arcs state', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/\+ Nouvel arc/i)).toBeInTheDocument()
  })

  it('shows the "Parcourir les presets" button in the empty arcs state', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByRole('button', { name: /Parcourir les presets/i })).toBeInTheDocument()
  })

  it('opens the preset picker when clicking "Parcourir les presets"', () => {
    render(<AscensionObjectivesTab />)
    fireEvent.click(screen.getByRole('button', { name: /Parcourir les presets/i }))
    // Le picker affiche son en-tête + l'état vide (presets mock = []).
    expect(screen.getByText(/Aucun preset disponible/i)).toBeInTheDocument()
  })

  it('shows the empty-state message for empty challenges (Prestige layer)', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByText(/Aucun objectif libre actif/i)).toBeInTheDocument()
  })

  it('renders the "+ Nouvel objectif" button in the objectives section', () => {
    render(<AscensionObjectivesTab />)
    expect(screen.getByRole('button', { name: /\+ Nouvel objectif/i })).toBeInTheDocument()
  })

  it('mode pilote OFF : le toggle propose « Activer » et l\'appuie active le pilote (B3)', () => {
    render(<AscensionObjectivesTab />)
    const toggle = screen.getByRole('button', { name: 'Activer', pressed: false })
    fireEvent.click(toggle)
    expect(pilotEnableMutate).toHaveBeenCalledTimes(1)
    expect(pilotDisableMutate).not.toHaveBeenCalled()
  })

  it('mode pilote ON (défi pilote actif) : le toggle propose « Désactiver » et désactive (B3)', () => {
    mockChallenges.current = [{ id: 'p1', mode: 'pilote', status: 'active' }]
    render(<AscensionObjectivesTab />)
    const toggle = screen.getByRole('button', { name: 'Désactiver', pressed: true })
    fireEvent.click(toggle)
    expect(pilotDisableMutate).toHaveBeenCalledTimes(1)
    expect(pilotEnableMutate).not.toHaveBeenCalled()
  })

  it('empty state objectifs libres : CTA d\'activation du mode pilote présent quand OFF (B3)', () => {
    render(<AscensionObjectivesTab />)
    expect(
      screen.getByRole('button', { name: /Activer le mode pilote/i }),
    ).toBeInTheDocument()
  })

  it('abandon d\'un objectif : confirme via AlertDialog puis appelle la mutation (B5)', () => {
    mockChallenges.current = [{ id: 'l1', mode: 'libre', status: 'active' }]
    render(<AscensionObjectivesTab />)

    // Le bouton « Abandonner » de la carte n'appelle plus confirm() natif : il
    // ouvre un AlertDialog (pas de mutation tant qu'on n'a pas confirmé).
    fireEvent.click(screen.getByRole('button', { name: 'Abandonner' }))
    const dialog = screen.getByRole('alertdialog')
    expect(within(dialog).getByText(/Abandonner cet objectif/i)).toBeInTheDocument()
    expect(abandonMutate).not.toHaveBeenCalled()

    // Confirmation → mutation déclenchée avec l'id de l'objectif.
    fireEvent.click(within(dialog).getByRole('button', { name: 'Abandonner' }))
    expect(abandonMutate).toHaveBeenCalledWith('l1')
  })

  it('ouvre la confirmation de suppression d\'arc et appelle deleteArc', () => {
    mockArcs.current = [{ id: 'arc1', title: 'Mon Arc' }]
    render(<AscensionObjectivesTab />)

    // Ouvre la confirmation (bouton « Supprimer » de l'élément d'arc).
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    expect(screen.getByText(/Supprimer l'arc « Mon Arc » \?/i)).toBeInTheDocument()

    // 0 objectif → un seul bouton « Supprimer » de confirmation + Annuler.
    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    expect(deleteArcMutate).toHaveBeenCalledWith({ id: 'arc1', cascade: true })
  })

  it('annule la suppression sans appeler deleteArc', () => {
    mockArcs.current = [{ id: 'arc1', title: 'Mon Arc' }]
    render(<AscensionObjectivesTab />)

    fireEvent.click(screen.getByRole('button', { name: 'Supprimer' }))
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(deleteArcMutate).not.toHaveBeenCalled()
  })
})
