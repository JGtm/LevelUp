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
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useSoloFilterStore as useGlobalFilterStore } from '@/stores/soloFilterStore'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useAppShellStore } from '@/stores/appShellStore'
import { DEFAULT_GAP_MINUTES } from '@/stores/createFilterStore'
import type { FilterContextResolved } from '@/lib/api/types'

import { FilterOmnibar } from './FilterOmnibar'

// Contexte effectif conforme au contrat (FilterContextInput côté API : champs requis,
// nullable explicites). DEFAULT_FILTER_CONTEXT côté store est l'input front (optionnels)
// et ne matche pas la forme résolue renvoyée par l'API.
const DEFAULT_EFFECTIVE: FilterContextResolved['effective'] = {
  filter_mode: 'period',
  period: { start_date: null, end_date: null },
  sessions: {
    gap_minutes: DEFAULT_GAP_MINUTES,
    picked_sessions: [],
    picked_session_label: null,
    picked_solo_session_label: null,
    picked_squad_session_label: null,
  },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

function buildResolved(): FilterContextResolved {
  return {
    effective: DEFAULT_EFFECTIVE,
    available_options: {
      experience_types: [{ label: 'PVP non classé', value: 'PVP non classé', count: 8 }],
      playlists: [{ label: 'Arène classée', value: 'Arène classée', count: 8 }],
      modes: [{ label: 'Slayer', value: 'Slayer', count: 8 }],
      maps: [{ label: 'Recharge', value: 'Recharge', count: 8 }],
    },
    session_options: {
      all_sessions: [
        // FilterOmnibar filtre is_squad=true (la pill ne montre que solo).
        { session_id: 's1', label: '01/04/2026 21:00', match_count: 5, match_count_filtered: 5, is_squad: false, started_at_utc: '2026-04-01T21:00:00Z', ended_at_utc: '2026-04-01T22:30:00Z' },
        { session_id: 's2', label: '15/03/2026 18:00', match_count: 3, match_count_filtered: 3, is_squad: false, started_at_utc: '2026-03-15T18:00:00Z', ended_at_utc: '2026-03-15T19:30:00Z' },
      ],
      solo_labels: [],
      squad_labels: [],
    },
    counts: { total_matches_before_filters: 8, total_matches_after_filters: 8 },
    period_presets: [
      { preset_id: '7d', days: 7, count: 2 },
      { preset_id: '30d', days: 30, count: 5 },
      { preset_id: '90d', days: 90, count: 8 },
      { preset_id: 'all', days: 0, count: 8 },
    ],
  }
}

describe('FilterOmnibar', () => {
  beforeEach(() => {
    // Locale pinnée : les libellés (Filtres, périodes, « N matchs », Analyser)
    // sont désormais résolus via le manifest i18n (GH-4) — on fige 'fr' pour que
    // les assertions FR ci-dessous soient explicites, pas dépendantes du défaut.
    useAppShellStore.setState({ locale: 'fr' })
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
    // SessionMultiSelect (refacto 2026-05) : placeholder "Rechercher…" + checkbox
    // par session + bouton "Valider" différé.
    expect(screen.getByPlaceholderText(/rechercher/i)).toBeInTheDocument()
    expect(screen.getByText('01/04/2026 21:00')).toBeInTheDocument()
    expect(screen.getByText('15/03/2026 18:00')).toBeInTheDocument()
  })

  it('sélection d\'une session + Valider + Analyser déclenche setFilterContext + auto-derive', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    // Click sur le label de la session → toggle la checkbox associée.
    fireEvent.click(screen.getByText('15/03/2026 18:00'))
    // Pending uniquement — store pas encore mis à jour
    expect(useGlobalFilterStore.getState().filterContext.sessions!.picked_sessions).toEqual([])
    // Confirm sélection via "Valider" (validation différée du multi-select)
    fireEvent.click(screen.getByRole('button', { name: /^valider$/i }))
    // Commit via Analyser (commit pending → store global)
    fireEvent.click(screen.getByRole('button', { name: /analyser/i }))
    const ctx = useGlobalFilterStore.getState().filterContext
    // Le nouveau composant utilise des labels de session, pas des IDs.
    expect(ctx.sessions!.picked_sessions).toEqual(['15/03/2026 18:00'])
    expect(ctx.filter_mode).toBe('sessions')
  })

  it('recherche filtre les sessions affichées', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    const search = screen.getByPlaceholderText(/rechercher/i)
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
    expect(screen.getByText('Sélections')).toBeInTheDocument()
    expect(screen.getByText('Modes')).toBeInTheDocument()
    expect(screen.getByText('Cartes')).toBeInTheDocument()
    expect(screen.getByText(/Type d'expérience/i)).toBeInTheDocument()
  })

  it('toggle cascade + Analyser déclenche setFilterContext', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /^filtres/i }))
    // Le label inclut maintenant le count (ex: "Slayer 8") — on cible
    // l'option par son texte et on remonte au label parent.
    const slayerLabel = screen.getByText('Slayer').closest('label')!
    fireEvent.click(slayerLabel.querySelector('input[type="checkbox"]')!)
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

  it('options à count=0 sont repliées sous "+ N indisponibles"', () => {
    // Dataset PVE only : 2 matchs Firefight sur Sanctuary. Cocher PVE doit
    // masquer Slayer/CTF/Strongholds dans la cascade Modes.
    const resolved = buildResolved()
    resolved.available_options.modes = [
      { label: 'Firefight', value: 'Firefight', count: 2 },
      { label: 'Slayer', value: 'Slayer', count: 0 },
      { label: 'CTF', value: 'CTF', count: 0 },
      { label: 'Strongholds', value: 'Strongholds', count: 0 },
      { label: 'Oddball', value: 'Oddball', count: 0 },
    ]
    useGlobalFilterStore.getState().setResolvedContext(resolved)
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /^filtres/i }))

    // Firefight (count > 0) visible, les 4 autres repliées
    expect(screen.getByText('Firefight')).toBeInTheDocument()
    expect(screen.queryByText('Slayer')).not.toBeInTheDocument()
    expect(screen.queryByText('CTF')).not.toBeInTheDocument()
    expect(screen.getByText(/4 options indisponibles/)).toBeInTheDocument()
  })

  it('Escape ferme le popover ouvert', () => {
    renderWithProviders(<FilterOmnibar />)
    fireEvent.click(screen.getByRole('button', { name: /toutes les sessions/i }))
    expect(screen.getByPlaceholderText(/rechercher/i)).toBeInTheDocument()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(screen.queryByPlaceholderText(/rechercher/i)).not.toBeInTheDocument()
  })

  // ── Bouton « Copier le lien avec les filtres » (share-link à la demande) ──

  it('bouton « Copier le lien » visible avec un store urlEnabled (solo)', () => {
    renderWithProviders(<FilterOmnibar />)
    expect(
      screen.getByRole('button', { name: /copier le lien avec les filtres/i }),
    ).toBeInTheDocument()
  })

  it('bouton « Copier le lien » absent avec un store non-urlEnabled (escouade)', () => {
    renderWithProviders(<FilterOmnibar filterStore={useSquadFilterStore} />)
    expect(
      screen.queryByRole('button', { name: /copier le lien/i }),
    ).not.toBeInTheDocument()
  })

  it('clic sur « Copier le lien » copie l’URL de partage dans le presse-papier', () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    renderWithProviders(<FilterOmnibar />)
    // L'URL attendue = celle que le store construit à la demande (identique à
    // l'appel du handler : même filterContext, même window.location).
    const expected = useGlobalFilterStore.getState().buildShareUrl()
    fireEvent.click(screen.getByRole('button', { name: /copier le lien avec les filtres/i }))
    expect(writeText).toHaveBeenCalledWith(expected)
  })
})
