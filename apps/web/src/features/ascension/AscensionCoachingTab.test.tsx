/**
 * Tests de structure pour AscensionCoachingTab (onglet « Entraînement »).
 *
 * Restructuration 4 onglets (2026-07, DEC-3). Le tab coaching compose :
 *   1. CoachFocusCard + CoachProposalsCard (suggestions)
 *   2. CampaignTracker (si campagne active)
 *   3. ProgressionSection (pistes de progression) + LeverList (leviers calibrés)
 *
 * L'identité (PlayerProfileV3) et les patterns contextuels ont migré vers
 * l'onglet « Profil » — on vérifie ici qu'ils ne sont PLUS rendus.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import type { PlayerProfile } from '@/lib/playerProfile'

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
vi.mock('./CoachFocusCard', () => ({
  CoachFocusCard: () => <div data-testid="coach-focus" />,
}))

// ── Profile / progression ─────────────────────────────────────────────────

const mockProfile: { current: PlayerProfile | null } = { current: null }

vi.mock('./profile/queries', () => ({
  useActiveCampaign: () => ({ data: null }),
  usePlayerProfile: () => ({ data: mockProfile.current, isLoading: false, isError: false }),
}))
vi.mock('./profile/ProgressionSection', () => ({
  ProgressionSection: () => <div data-testid="progression-section" />,
}))

// ── Patterns ──────────────────────────────────────────────────────────────

const mockPatterns: { current: unknown } = { current: null }

vi.mock('./queries', () => ({
  usePatterns: () => ({ data: mockPatterns.current, isLoading: false }),
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
    mockProfile.current = null
    mockPatterns.current = null
  })

  it('renders the coaching LayerSection header', () => {
    render(<AscensionCoachingTab />)
    expect(screen.getByText(/Ascension — Coaching d'amélioration/i)).toBeInTheDocument()
  })

  it('renders the Coach focus + proposals cards', () => {
    render(<AscensionCoachingTab />)
    expect(screen.getByTestId('coach-focus')).toBeInTheDocument()
    expect(screen.getByTestId('coach-proposals')).toBeInTheDocument()
  })

  it('does NOT render PlayerProfileV3 (moved to Profil tab)', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByTestId('player-profile-v3')).not.toBeInTheDocument()
    expect(screen.queryByTestId('pattern-context-grid')).not.toBeInTheDocument()
  })

  it('does not render CampaignTracker when no active campaign', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByTestId('campaign-tracker')).not.toBeInTheDocument()
  })

  it('does not render ProgressionSection when profile has no data', () => {
    render(<AscensionCoachingTab />)
    expect(screen.queryByTestId('progression-section')).not.toBeInTheDocument()
  })

  it('renders ProgressionSection when the profile has enough data', () => {
    mockProfile.current = {
      has_enough_data: true,
      leverages: [{ component: 'accuracy', leverage_value: 0.3, narrative_axes: [], coaching_message: 'k' }],
      suggested_challenges: [],
    } as unknown as PlayerProfile
    render(<AscensionCoachingTab />)
    expect(screen.getByTestId('progression-section')).toBeInTheDocument()
  })

  it('renders LeverList when patterns expose calibrated levers', () => {
    mockPatterns.current = { levers: [{ rank: 1, axis: 'accuracy', current_val: 0, target_val: 1, horizon: 10, impact: 0.2 }] }
    render(<AscensionCoachingTab />)
    expect(screen.getByTestId('lever-list')).toBeInTheDocument()
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
      screen.getByText(/Sélectionne un joueur pour voir l'entraînement/i),
    ).toBeInTheDocument()
  })
})
