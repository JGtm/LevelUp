/**
 * Tests SessionNavBar — barre sticky h-12 avec gros boutons de nav session.
 *
 * Couvre :
 *  - Visibilité conditionnelle (Stats uniquement, mode Session uniquement)
 *  - Mode session : affiche label + Précédente/Suivante/Dernière
 *  - Mode analyse (sans session pickée) : barre masquée — info redondante avec
 *    le briefing en haut de page (KPI grid + FilterOmnibar)
 *  - États disabled des boutons selon position dans la liste
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import {
  useGlobalFilterStore,
  DEFAULT_GAP_MINUTES,
  DEFAULT_FILTER_CONTEXT,
} from '@/stores/globalFilterStore'
import type { FilterContextResolved, SessionOption } from '@/lib/api/types'

import { SessionNavBar } from './SessionNavBar'

let mockPathname = '/players/test-player/squad/synergies'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useRouterState: () => ({ location: { pathname: mockPathname } }),
  }
})

const fakeSessions: SessionOption[] = [
  { session_id: 'sess-newest', label: '01/04/2026 21:00–22:00', match_count: 5, is_squad: true },
  { session_id: 'sess-mid', label: '15/03/2026 18:00–19:00', match_count: 3, is_squad: false },
  { session_id: 'sess-oldest', label: '01/02/2026 12:00–13:00', match_count: 1, is_squad: false },
]

function buildResolved(sessions: SessionOption[] = fakeSessions): FilterContextResolved {
  return {
    effective: DEFAULT_FILTER_CONTEXT,
    available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
    session_options: { all_sessions: sessions, solo_labels: [], squad_labels: [] },
    counts: { total_matches_before_filters: 9, total_matches_after_filters: 9 },
  }
}

describe('SessionNavBar', () => {
  beforeEach(() => {
    // Commit 26111a3a (cascade filters refactor) : la section Escouade gère
    // sa propre barre dans SquadLayout, donc SessionNavBar ne s'affiche
    // plus que sur les routes Stats. Tous les tests partagent ce path par
    // défaut, sauf ceux qui testent explicitement le rendu hors Stats.
    mockPathname = '/players/test-player/stats/history'
    useGlobalFilterStore.getState().resetFilters()
  })

  it('ne rend rien hors Stats', () => {
    mockPathname = '/players/test-player/home'
    const { container } = renderWithProviders(<SessionNavBar />)
    expect(container.firstChild).toBeNull()
  })

  it('ne rend rien sur la section Escouade (SquadLayout porte sa propre barre)', () => {
    mockPathname = '/players/test-player/squad/synergies'
    const { container } = renderWithProviders(<SessionNavBar />)
    expect(container.firstChild).toBeNull()
  })

  it('ne rend pas la barre sans session pickée (mode Analyse masqué)', () => {
    // La barre n'apparaît qu'en mode Session — l'info "Toutes les sessions ·
    // N filtres · N matchs" était redondante avec le briefing (KPI grid +
    // FilterOmnibar) et a été retirée.
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    const { container } = renderWithProviders(<SessionNavBar />)
    expect(container.firstChild).toBeNull()
  })

  it('rend la barre sur la section Stats en mode Session', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-mid'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<SessionNavBar />)
    expect(screen.getByRole('navigation', { name: /session/i })).toBeInTheDocument()
  })

  it('mode session : affiche le label et les 3 boutons', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-mid'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<SessionNavBar />)
    expect(screen.getByText('15/03/2026 18:00–19:00')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /précédente/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /suivante/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /dernière/i })).toBeInTheDocument()
  })

  it('mode session : Précédente disabled sur la session la plus ancienne', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-oldest'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<SessionNavBar />)
    const prevBtn = screen.getByRole('button', { name: /session précédente/i })
    expect(prevBtn).toBeDisabled()
  })

  it('mode session : Suivante et Dernière disabled sur la session la plus récente', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-newest'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<SessionNavBar />)
    expect(screen.getByRole('button', { name: /session suivante/i })).toBeDisabled()
    expect(screen.getByRole('button', { name: /dernière session/i })).toBeDisabled()
  })

  it('mode session : Suivante navigue vers une session plus récente', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-mid'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<SessionNavBar />)
    fireEvent.click(screen.getByRole('button', { name: /session suivante/i }))
    expect(useGlobalFilterStore.getState().filterContext.sessions!.picked_sessions).toEqual([
      'sess-newest',
    ])
  })

  it('mode session : badge "auto" affiché quand auto-snap actif', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().autoSnapToLatestSession('sess-newest', true)
    renderWithProviders(<SessionNavBar />)
    expect(screen.getByText('auto')).toBeInTheDocument()
  })

  it('mode analyse : barre masquée (rendu null)', () => {
    // L'info "Toutes les sessions · filtres · N matchs" est désormais portée
    // par le briefing + FilterOmnibar — on ne rend rien.
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    const { container } = renderWithProviders(<SessionNavBar />)
    expect(container.firstChild).toBeNull()
  })

  it('hauteur fixe h-12 en mode Session', () => {
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['sess-mid'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    const { container } = renderWithProviders(<SessionNavBar />)
    const bar = container.querySelector('[role="navigation"]')!
    expect(bar.className).toContain('h-12')
  })
})
