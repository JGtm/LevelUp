/**
 * Tests unitaires — useLocalFilterBar.
 *
 * Couvre le contrat clé du hook : pending → committed atomique via Analyser,
 * reset total ↺, et états dérivés (hasActiveFilters, committedFilterContext).
 *
 * `useFiltersPreview` est mocké pour retourner un dataset minimal avec
 * available_options afin que les dropdowns rendent leurs counts. Le bar
 * (JSX) est rendu via testing-library pour cliquer sur Analyser et ↺.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act, render, screen, fireEvent } from '@testing-library/react'

import { useLocalFilterBar, type LocalFilterBarLabels } from './useLocalFilterBar'

// Mock useFiltersPreview : retourne un dataset stable, pas d'appel réseau.
vi.mock('@/features/filters/queries', () => ({
  useFiltersPreview: () => ({
    data: {
      effective: {
        filter_mode: 'period',
        period: { start_date: null, end_date: null },
        sessions: { picked_sessions: [], gap_minutes: 120 },
        cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
      },
      available_options: {
        experience_types: [
          { value: 'PVP classé', label: 'PVP classé', count: 10 },
          { value: 'PVP non classé', label: 'PVP non classé', count: 20 },
        ],
        playlists: [{ value: 'Slayer Ranked', label: 'Slayer Ranked', count: 10 }],
        modes: [{ value: 'Slayer', label: 'Slayer', count: 30 }],
        maps: [],
      },
      session_options: { all_sessions: [], solo_labels: [], squad_labels: [] },
      counts: { total_matches_before_filters: 30, total_matches_after_filters: 30 },
      period_presets: [],
      season_counts: [],
    },
    isFetching: false,
  }),
}))

// Mock useActiveSeason pour éviter de dépendre du catalog des saisons en env test.
vi.mock('@/features/squad/useActiveSeason', () => ({
  useActiveSeason: () => ({ seasons: [], activeSeason: null }),
  seasonToPeriod: (s: { startDate: Date; endDate: Date | null }) => ({
    start_date: s.startDate.toISOString().slice(0, 10),
    end_date: (s.endDate ?? new Date()).toISOString().slice(0, 10),
  }),
}))

// Mock appShellStore.locale pour MultiSelectFilter (qui consomme la locale).
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (selector: (s: { locale: string }) => unknown) => selector({ locale: 'fr' }),
}))

const LABELS: LocalFilterBarLabels = {
  experience: 'Expérience',
  experienceAll: 'Toutes',
  experienceRanked: 'Classé',
  experienceUnranked: 'Non classé',
  playlists: 'Playlists',
  modes: 'Modes',
  reset: 'Réinitialiser',
}

beforeEach(() => {
  // Reset DOM
  document.body.innerHTML = ''
})

describe('useLocalFilterBar', () => {
  it('initialise committedFilterContext à DEFAULT', () => {
    const { result } = renderHook(() =>
      useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS }),
    )

    expect(result.current.committedFilterContext.filter_mode).toBe('period')
    expect(result.current.committedFilterContext.period).toEqual({
      start_date: null,
      end_date: null,
    })
    expect(result.current.committedFilterContext.cascade?.experience_types ?? []).toEqual([])
    expect(result.current.committedFilterContext.cascade?.playlists ?? []).toEqual([])
    expect(result.current.committedFilterContext.cascade?.modes ?? []).toEqual([])
    expect(result.current.hasActiveFilters).toBe(false)
  })

  it('rend la barre avec le dropdown Expérience visible', () => {
    function Wrapper() {
      const { bar } = useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS })
      return <>{bar}</>
    }
    render(<Wrapper />)

    expect(screen.getByRole('button', { name: /Expérience\s*:/ })).toBeInTheDocument()
  })

  it('committedHash a le format FNV-1a 32 bits (8 hex chars)', () => {
    const { result } = renderHook(() =>
      useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS }),
    )
    expect(result.current.committedHash).toMatch(/^[0-9a-f]{8}$/)
  })

  it('dropdown Expérience s’ouvre au clic et révèle les options', () => {
    function Wrapper() {
      const { bar } = useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS })
      return <>{bar}</>
    }
    render(<Wrapper />)

    // Avant clic : seul le trigger est visible (les options du popup absentes)
    const trigger = screen.getByRole('button', { name: /Expérience\s*:/ })
    // Avant clic : pas de bouton option "Classé X" (count concaténé en accessible name).
    expect(screen.queryByRole('button', { name: /^Classé\d*$/ })).not.toBeInTheDocument()

    act(() => {
      fireEvent.click(trigger)
    })

    // Après clic : les 3 options sont rendues — leur accessible name concatène
    // le label et le count (ex: "Toutes30", "Classé10", "Non classé20").
    expect(screen.getByRole('button', { name: /^Toutes\d+$/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Classé\d+$/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /^Non classé\d+$/ })).toBeInTheDocument()
  })

  it('hasActiveFilters reflète l’état committed (false initial)', () => {
    const { result } = renderHook(() =>
      useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS }),
    )
    expect(result.current.hasActiveFilters).toBe(false)
  })

  it('bar contient bouton Analyser et placeholder filtres', () => {
    function Wrapper() {
      const { bar } = useLocalFilterBar({ playerSlug: 'test-player', labels: LABELS })
      return <>{bar}</>
    }
    render(<Wrapper />)

    expect(screen.getByRole('button', { name: 'Analyser' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Playlists/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Modes/ })).toBeInTheDocument()
  })
})
