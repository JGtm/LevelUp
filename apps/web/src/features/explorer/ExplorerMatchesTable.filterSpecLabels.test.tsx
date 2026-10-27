/**
 * ExplorerMatchesTable — le descripteur de retour du match ouvert depuis le filtre global porte les
 * libellés affichés (langue de la page) des playlists et modes choisis, pas leurs clés (nom de la
 * langue par défaut). Les clés restent ce qui filtre : `playlist_names` / `mode_categories`.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { cleanup, fireEvent, screen } from '@testing-library/react'

const navigateToMatch = vi.fn()
vi.mock('@/lib/match-nav/useNavigateToMatch', () => ({ useNavigateToMatch: () => navigateToMatch }))

import { renderWithProviders } from '@/test/render-utils'
import type { ExplorerMatchRow, FilterContextResolved } from '@/lib/api/types'
import { useSoloFilterStore } from '@/stores/soloFilterStore'

import { ExplorerMatchesTable } from './ExplorerMatchesTable'

const row: ExplorerMatchRow = {
  match_id: 'match-1',
  start_time: '2026-05-26T12:00:00Z',
  start_time_label: '2026-05-26 12:00',
  map_ui: 'Map',
  mode_ui: 'Slayer',
  playlist_label: 'Quick Play',
  outcome: 'win',
  outcome_code: 2,
  score_label: '50-30',
  is_with_friends: false,
  experience_type_label: 'PvP',
  kills: 10,
  deaths: 5,
  assists: 3,
  kda: 2.1,
}

afterEach(() => {
  cleanup()
  navigateToMatch.mockReset()
  useSoloFilterStore.setState({ resolvedContext: null })
})

describe('ExplorerMatchesTable : libellés du filtre global dans le descripteur', () => {
  it('pose les libellés des options du filtre résolu, les clés restent les valeurs du filtre', () => {
    const base = useSoloFilterStore.getState().filterContext
    useSoloFilterStore.setState({
      filterContext: { ...base, cascade: { ...base.cascade, playlists: ['Partie rapide'], modes: ['Assassin'] } },
      resolvedContext: {
        available_options: {
          playlists: [{ value: 'Partie rapide', label: 'Quick Play' }],
          modes: [{ value: 'Assassin', label: 'Slayer' }],
          maps: [],
          experience_types: [],
        },
      } as unknown as FilterContextResolved,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)

    fireEvent.click(screen.getByRole('button', { name: /Ouvrir|Open/ }))

    expect(navigateToMatch).toHaveBeenCalledTimes(1)
    const spec = navigateToMatch.mock.calls[0][1].filterSpec
    expect(spec.playlist_names).toEqual(['Partie rapide'])
    expect(spec.playlist_labels).toEqual({ 'Partie rapide': 'Quick Play' })
    expect(spec.mode_categories).toEqual(['Assassin'])
    expect(spec.mode_labels).toEqual({ Assassin: 'Slayer' })
  })

  it('sans options résolues, le descripteur ne porte que les clés', () => {
    const base = useSoloFilterStore.getState().filterContext
    useSoloFilterStore.setState({
      filterContext: { ...base, cascade: { ...base.cascade, playlists: ['Partie rapide'] } },
      resolvedContext: null,
    })
    renderWithProviders(<ExplorerMatchesTable rows={[row]} playerSlug="me" />)
    fireEvent.click(screen.getByRole('button', { name: /Ouvrir|Open/ }))
    const spec = navigateToMatch.mock.calls[0][1].filterSpec
    expect(spec.playlist_names).toEqual(['Partie rapide'])
    expect(spec.playlist_labels).toBeUndefined()
  })
})
