/**
 * Tests FilterOmnibar — bandeau de pills (Filtres | sep | Période | Session | Analyser).
 *
 * Couvre :
 *  - Rendu des 3 pills (Session affichée seulement si sessions disponibles)
 *  - Click pill ouvre/ferme le popover
 *  - Sélection session + "Analyser" déclenche setFilterContext
 *  - Sélection période + "Analyser" déclenche setFilterContext + auto-derive
 *  - Toggle cascade + "Analyser" déclenche setFilterContext
 *  - Compteur cascade actif sur la pill Filtres (état pending)
 *  - Bouton Réinitialiser visible quand filtres actifs
 *  - Compteur global de matchs depuis resolvedContext.counts
 *  - Escape ferme tous les popovers
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import {
  useGlobalFilterStore,
  DEFAULT_GAP_MINUTES,
  DEFAULT_FILTER_CONTEXT,
} from '@/stores/globalFilterStore'
import type { FilterContextResolved } from '@/lib/api/types'

import { FilterOmnibar } from './FilterOmnibar'

function buildResolved(): FilterContextResolved {
  return {
    effective: DEFAULT_FILTER_CONTEXT,
    available_options: {
      experience_types: [{ label: 'PVP non classé', value: 'PVP non classé' }],
      playlists: [{ label: 'Arène classée', value: 'Arène classée' }],
      modes: [{ label: 'Slayer', value: 'Slayer' }],
      maps: [{ label: 'Recharge', value: 'Recharge' }],
    },
    session_options: {
      all_sessions: [
        { session_id: 's1', label: '01/04/2026 21:00', match_count: 5, is_squad: true },
        { session_id: 's2', label: '15/03/2026 18:00', match_count: 3, is_squad: false },
      ],
      solo_labels: [],
      squad_labels: [],
    },
    counts: { total_matches_before_filters: 8, total_matches_after_filters: 8 },
  }
}

describe('FilterOmnibar', () => {
  beforeEach(() => {
    useGlobalFilterStore.getState().resetFilters()
    useGlobalFilterStore.getState().setResolvedContext(buildResolved())
  })

  it('affiche les 3 pills (Filtres / Période / Session)', () => {
    renderWithProviders(<FilterOmnibar />)
    expect(screen.getByRole('button', { name: /^filtres/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /toutes les périodes/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /toutes les sessions/i })).toBeInTheDocument()
  })

  it('pill Session masquée s\'il n\'y a aucune session', () => {
    const empty = buildResolved()
    empty.session_options.all_sessions = []
    useGlobalFilterStore.getState().setResolvedContext(empty)
    renderWithProviders(<FilterOmnibar />)
    expect(screen.queryByRole('button', { name: /toutes les sessions/i })).not.toBeInTheDocument()
  })

  it('click sur pill Session ouvre le popover avec recherche', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    expect(screen.getByPlaceholderText(/rechercher une session/i)).toBeInTheDocument()
    expect(screen.getByText('01/04/2026 21:00')).toBeInTheDocument()
    expect(screen.getByText('15/03/2026 18:00')).toBeInTheDocument()
  })

  it('sélection d\'une session + Analyser déclenche setFilterContext + auto-derive', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    fireEvent.click(screen.getByText('15/03/2026 18:00'))
    // Pending uniquement — store pas encore mis à jour
    expect(useGlobalFilterStore.getState().filterContext.sessions!.picked_sessions).toEqual([])
    // Commit via Analyser
    fireEvent.click(screen.getByRole('button', { name: /analyser/i }))
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.sessions!.picked_sessions).toEqual(['s2'])
    expect(ctx.filter_mode).toBe('sessions')
  })

  it('recherche filtre les sessions affichées', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    const search = screen.getByPlaceholderText(/rechercher une session/i)
    fireEvent.change(search, { target: { value: '01/04' } })
    expect(screen.getByText('01/04/2026 21:00')).toBeInTheDocument()
    expect(screen.queryByText('15/03/2026 18:00')).not.toBeInTheDocument()
  })

  it('click sur pill Période ouvre le popover avec presets', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les périodes/i }))
    expect(screen.getByText('7 jours')).toBeInTheDocument()
    expect(screen.getByText('30 jours')).toBeInTheDocument()
    expect(screen.getByText('90 jours')).toBeInTheDocument()
    expect(screen.getByText('Toutes')).toBeInTheDocument()
  })

  it('preset Période 30 jours + Analyser pose une période et auto-derive mode=period', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['s1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /période/i }))
    fireEvent.click(screen.getByText('30 jours'))
    // Pending uniquement — store pas encore mis à jour
    expect(useGlobalFilterStore.getState().filterContext.sessions!.picked_sessions).toEqual(['s1'])
    // Commit via Analyser
    fireEvent.click(screen.getByRole('button', { name: /analyser/i }))
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.period!.start_date).not.toBeNull()
    expect(ctx.filter_mode).toBe('period')
    expect(ctx.sessions!.picked_sessions).toEqual([])
  })

  it('click sur pill Filtres ouvre le popover avec 4 groupes', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /^filtres/i }))
    expect(screen.getByText('Playlists')).toBeInTheDocument()
    expect(screen.getByText('Modes')).toBeInTheDocument()
    expect(screen.getByText('Cartes')).toBeInTheDocument()
    expect(screen.getByText(/Type d'expérience/i)).toBeInTheDocument()
  })

  it('toggle cascade + Analyser déclenche setFilterContext', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /^filtres/i }))
    fireEvent.click(screen.getByLabelText('Slayer'))
    // Pending uniquement — store pas encore mis à jour
    expect(useGlobalFilterStore.getState().filterContext.cascade!.modes).not.toContain('Slayer')
    // Commit via Analyser
    fireEvent.click(screen.getByRole('button', { name: /analyser/i }))
    expect(useGlobalFilterStore.getState().filterContext.cascade!.modes).toContain('Slayer')
  })

  it('badge de count affiché sur la pill Filtres quand cascade active', () => {
    useGlobalFilterStore.getState().setCascade({
      experience_types: [],
      playlists: ['Arène classée'],
      modes: ['Slayer'],
      maps: [],
    })
    renderWithProviders(<FilterOmnibar />)
    const filtresPill = screen.getByRole('button', { name: /^filtres/i })
    expect(filtresPill).toHaveTextContent('2')
  })

  it('bouton Réinitialiser visible seulement si filtres actifs', () => {
    renderWithProviders(<FilterOmnibar />)
    expect(screen.queryByRole('button', { name: /réinitialiser/i })).not.toBeInTheDocument()
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['s1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<FilterOmnibar />)
    expect(screen.getAllByRole('button', { name: /réinitialiser/i }).length).toBeGreaterThan(0)
  })

  it('Réinitialiser remet le store au défaut', () => {
    useGlobalFilterStore.getState().setSessions({
      picked_sessions: ['s1'],
      gap_minutes: DEFAULT_GAP_MINUTES,
    })
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /réinitialiser/i }))
    const ctx = useGlobalFilterStore.getState().filterContext
    expect(ctx.sessions!.picked_sessions).toEqual([])
  })

  it('compteur global de matchs depuis resolvedContext.counts', () => {
    const resolved = buildResolved()
    resolved.counts.total_matches_after_filters = 23
    useGlobalFilterStore.getState().setResolvedContext(resolved)
    renderWithProviders(<FilterOmnibar />)
    expect(screen.getByText(/23 matchs/)).toBeInTheDocument()
  })

  it('Escape ferme le popover ouvert', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    expect(screen.getByPlaceholderText(/rechercher une session/i)).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByPlaceholderText(/rechercher une session/i)).not.toBeInTheDocument()
  })
})
