/**
 * Tests PeriodSessionRail — barre sticky de navigation période/session.
 *
 * Couvre :
 *  - Mode hidden (pas de scope) → composant ne rend rien
 *  - Mode session → label session + boutons prev/next/latest
 *  - Mode multi-session → label "N sessions" + boutons disabled
 *  - Mode période → label "Période du X au Y" + boutons prev/next
 *  - Click prev/next déclenche les actions du store
 */
import { beforeEach, describe, expect, it } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useSoloFilterStore as useGlobalFilterStore } from '@/stores/soloFilterStore'
import { DEFAULT_GAP_MINUTES, DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import type { FilterContextResolved } from '@/lib/api/types'

import { PeriodSessionRail } from './PeriodSessionRail'

function buildResolved(allSessions: Array<{ id: string; label: string }>): FilterContextResolved {
  return {
    effective: DEFAULT_FILTER_CONTEXT,
    available_options: { experience_types: [], playlists: [], modes: [], maps: [] },
    session_options: {
      all_sessions: allSessions.map((s) => ({
        session_id: s.id,
        label: s.label,
        match_count: 5,
        match_count_filtered: 5,
        is_squad: false,
      })),
      solo_labels: [],
      squad_labels: [],
    },
    counts: { total_matches_before_filters: 100, total_matches_after_filters: 50 },
    period_presets: [],
  }
}

describe('PeriodSessionRail', () => {
  beforeEach(() => {
    useGlobalFilterStore.getState().resetFilters()
  })

  it('mode hidden : ne rend rien quand pas de session disponible (cold start)', () => {
    const { container } = renderWithProviders(<PeriodSessionRail />)
    expect(container.firstChild).toBeNull()
  })

  it('mode all-time : rail informatif "Toutes les sessions" + boutons disabled', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(
      buildResolved([
        { id: 's-1', label: '06/04' },
        { id: 's-2', label: '05/04' },
      ]),
    )
    // filterContext reste en defaults (pas de période, pas de session) → all-time

    renderWithProviders(<PeriodSessionRail />)
    expect(screen.getByText(/Toutes les sessions|All sessions/)).toBeTruthy()
    const prevBtn = screen.getByLabelText(/Session précédente|Previous session/) as HTMLButtonElement
    expect(prevBtn.disabled).toBe(true)
  })

  it('mode session : affiche le label de session + boutons', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(
      buildResolved([
        { id: 's-latest', label: '06/04 21h24' },
        { id: 's-old', label: '01/04 12h00' },
      ]),
    )
    store.setSessions({ picked_sessions: ['s-latest'], gap_minutes: DEFAULT_GAP_MINUTES })

    renderWithProviders(<PeriodSessionRail />)
    expect(screen.getByText('06/04 21h24')).toBeTruthy()
    expect(screen.getByLabelText(/Session précédente|Previous session/)).toBeTruthy()
    expect(screen.getByLabelText(/Session suivante|Next session/)).toBeTruthy()
  })

  it('mode multi-session : affiche le compteur + boutons disabled', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(
      buildResolved([
        { id: 's-1', label: '06/04' },
        { id: 's-2', label: '05/04' },
      ]),
    )
    store.setSessions({ picked_sessions: ['s-1', 's-2'], gap_minutes: DEFAULT_GAP_MINUTES })

    renderWithProviders(<PeriodSessionRail />)
    expect(screen.getByText(/2 sessions/i)).toBeTruthy()
    const prevBtn = screen.getByLabelText(/Session précédente|Previous session/) as HTMLButtonElement
    expect(prevBtn.disabled).toBe(true)
  })

  it('mode période : affiche le label de période + boutons prev/next', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(buildResolved([]))
    store.setPeriod({ start_date: '2026-04-01', end_date: '2026-04-08' })

    renderWithProviders(<PeriodSessionRail />)
    expect(screen.getByLabelText(/Période précédente|Previous period/)).toBeTruthy()
    expect(screen.getByText(/du.+au/i)).toBeTruthy()
  })

  it('matchCount : affiche le compteur de matchs en multi-session (où le rail n\'en a pas) + le trailing', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(
      buildResolved([
        { id: 's-1', label: '06/04' },
        { id: 's-2', label: '05/04' },
      ]),
    )
    store.setSessions({ picked_sessions: ['s-1', 's-2'], gap_minutes: DEFAULT_GAP_MINUTES })

    renderWithProviders(
      <PeriodSessionRail matchCount={42} trailing={<button type="button">Voir les matchs</button>} />,
    )
    expect(screen.getByText(/42 match/i)).toBeTruthy()
    expect(screen.getByText('Voir les matchs')).toBeTruthy()
  })

  it('clic ◀ Précédente bascule vers la session plus ancienne (label en sortie)', () => {
    const store = useGlobalFilterStore.getState()
    store.setResolvedContext(
      buildResolved([
        { id: 's-latest', label: '06/04' },
        { id: 's-mid', label: '05/04' },
      ]),
    )
    store.setSessions({ picked_sessions: ['s-latest'], gap_minutes: DEFAULT_GAP_MINUTES })

    renderWithProviders(<PeriodSessionRail />)
    const prevBtn = screen.getByLabelText(/Session précédente|Previous session/)
    fireEvent.click(prevBtn)
    expect(useGlobalFilterStore.getState().filterContext.sessions?.picked_sessions).toEqual(['05/04'])
  })
})
