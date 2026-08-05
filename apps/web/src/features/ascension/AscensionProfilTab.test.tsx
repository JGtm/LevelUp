/**
 * Tests de structure pour AscensionProfilTab (onglet « Profil », index).
 *
 * Restructuration 4 onglets (2026-07, DEC-3). Le tab Profil compose l'identité
 * (PlayerProfileV3) + les patterns contextuels (grille + solo/escouade +
 * comportements). Les leviers/pistes de progression vivent dans l'onglet
 * « Entraînement », la couche Prestige dans « Objectifs » — on vérifie ici
 * qu'ils ne sont PAS rendus.
 *
 * PlayerProfileV3 est mocké (évite le piège jsdom du radar echarts-for-react).
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

// ── Profil ────────────────────────────────────────────────────────────────

vi.mock('./profile/PlayerProfileV3', () => ({
  PlayerProfileV3: () => <div data-testid="player-profile-v3" />,
}))

// ── Patterns ──────────────────────────────────────────────────────────────

const mockPatterns: { current: unknown } = { current: null }

vi.mock('./queries', () => ({
  usePatterns: () => ({ data: mockPatterns.current, isLoading: false }),
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

// Import after mocks
import { AscensionProfilTab } from './AscensionProfilTab'

describe('AscensionProfilTab — composition (couche Profil)', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.currentPlayer = {
      player_slug: 'demo-player',
      gamertag: 'DemoPlayer',
    }
    mockShellState.locale = 'fr'
    mockPatterns.current = null
  })

  it('renders the Profil LayerSection header', () => {
    render(<AscensionProfilTab />)
    expect(screen.getByText(/Profil de jeu/i)).toBeInTheDocument()
  })

  it('renders PlayerProfileV3', () => {
    render(<AscensionProfilTab />)
    expect(screen.getByTestId('player-profile-v3')).toBeInTheDocument()
  })

  it('does NOT render the Prestige objectives layer (moved to Objectives tab)', () => {
    render(<AscensionProfilTab />)
    expect(screen.queryByText(/Mes objectifs actifs/i)).not.toBeInTheDocument()
    expect(screen.queryByText(/Prestige — Objectifs et arcs/i)).not.toBeInTheDocument()
  })

  it('does NOT render calibrated levers (moved to Training tab)', () => {
    mockPatterns.current = {
      context_patterns: [{ type: 'by_mode', key: 'slayer' }],
      behavior_patterns: [],
    }
    render(<AscensionProfilTab />)
    expect(screen.queryByTestId('lever-list')).not.toBeInTheDocument()
  })

  it('renders the pattern context grid when context patterns are present', () => {
    mockPatterns.current = {
      context_patterns: [{ type: 'by_mode', key: 'slayer' }],
      behavior_patterns: [],
    }
    render(<AscensionProfilTab />)
    expect(screen.getByTestId('pattern-context-grid')).toBeInTheDocument()
  })

  it('renders behavior alerts when behavior patterns are present', () => {
    mockPatterns.current = {
      context_patterns: [],
      behavior_patterns: [{ type: 'tilt' }],
    }
    render(<AscensionProfilTab />)
    expect(screen.getByTestId('behavior-alerts')).toBeInTheDocument()
  })

  it('switches to English copy when locale is en', () => {
    mockShellState.locale = 'en'
    render(<AscensionProfilTab />)
    expect(screen.getByText(/Play profile/i)).toBeInTheDocument()
  })

  it('shows "select a player" message when currentPlayer is null', () => {
    mockShellState.currentPlayer = null
    render(<AscensionProfilTab />)
    expect(
      screen.getByText(/Sélectionne un joueur pour voir les objectifs/i),
    ).toBeInTheDocument()
  })
})
