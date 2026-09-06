/**
 * TacticalFilterBar — la barre L2 de l'onglet, montée pour de vrai.
 *
 * Ce que ces tests cadenassent, et ce qui casse à l'écran sans eux :
 *   - les pièces ASSEMBLÉES sont bien là (expérience, playlists, modes, vue,
 *     sessions, composition) : un contrôle qui disparaîtrait d'un `extras` mal câblé
 *     ne se verrait dans aucun autre test ;
 *   - les SESSIONS PROPOSÉES SUIVENT LA COMPOSITION — sessions d'escouade dès qu'un
 *     coéquipier est choisi, sessions solo sinon. C'est la règle produit du
 *     2026-09-06, et elle est invisible tant qu'on ne l'exerce pas dans les DEUX
 *     états ;
 *   - la barre ÉCRIT dans le scope (donc dans l'URL) plutôt que dans un état local :
 *     sans cela, la sélection ne survivrait ni au retour navigateur ni au partage.
 */
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { getTacticalText } from './i18n'
import { TacticalFilterBar } from './TacticalFilterBar'
import { TACTICAL_SCOPE_DEFAUT, type TacticalScope } from './tacticalScope'

const sessions = [
  {
    label: 'Session solo',
    session_id: 's1',
    match_count: 4,
    match_count_filtered: 4,
    is_squad: false,
    started_at_utc: '2026-03-01T10:00:00Z',
    ended_at_utc: '2026-03-01T12:00:00Z',
  },
  {
    label: 'Session escouade',
    session_id: 's2',
    match_count: 7,
    match_count_filtered: 7,
    is_squad: true,
    started_at_utc: '2026-03-02T10:00:00Z',
    ended_at_utc: '2026-03-02T12:00:00Z',
  },
]

vi.mock('@/features/filters/queries', () => ({
  useFiltersPreview: () => ({
    data: {
      available_options: {
        experience_types: [{ value: 'PVP classé', label: 'PVP classé', count: 10 }],
        playlists: [{ value: 'Ranked Arena', label: 'Ranked Arena', count: 10 }],
        modes: [{ value: 'Slayer', label: 'Slayer', count: 8 }],
        maps: [],
      },
      session_options: { all_sessions: sessions, solo_labels: [], squad_labels: [] },
      counts: { total_matches_before_filters: 10, total_matches_after_filters: 10 },
      period_presets: [],
    },
    isFetching: false,
  }),
}))

vi.mock('@/features/squad/useActiveSeason', () => ({
  useActiveSeason: () => ({ seasons: [], activeSeason: null }),
  seasonToPeriod: () => ({ start_date: null, end_date: null }),
}))

const setScope = vi.fn()

function monter(scope: Partial<TacticalScope> = {}) {
  return renderWithProviders(
    <TacticalFilterBar
      playerSlug="JGtm"
      locale="fr"
      t={getTacticalText('fr')}
      scope={{ ...TACTICAL_SCOPE_DEFAUT, ...scope }}
      setScope={setScope}
      coequipierOptions={[{ gamertag: 'Ami', xuid: 'xuid(42)', encounter_count: 12 }]}
    />,
  )
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  setScope.mockReset()
})

describe('TacticalFilterBar', () => {
  it('assemble les contrôles existants, sans en inventer un seul', () => {
    monter()
    expect(screen.getByRole('button', { name: /Expérience\s*:/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Playlists/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Modes/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Vue\s*:/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Analyser' })).toBeInTheDocument()
  })

  it('sans composition : le sélecteur propose les sessions SOLO', () => {
    monter()
    fireEvent.click(screen.getByText('Sessions'))
    expect(screen.getByText('Session solo')).toBeInTheDocument()
    expect(screen.queryByText('Session escouade')).toBeNull()
  })

  it('avec une composition : il propose les sessions d’ESCOUADE', () => {
    monter({ coequipiers: ['Ami'] })
    fireEvent.click(screen.getByText('Sessions'))
    expect(screen.getByText('Session escouade')).toBeInTheDocument()
    expect(screen.queryByText('Session solo')).toBeNull()
  })

  it('le clic sur « Analyser » écrit la période et la cascade dans le scope', () => {
    monter()
    fireEvent.click(screen.getByRole('button', { name: 'Analyser' }))
    expect(setScope).toHaveBeenCalledTimes(1)
    expect(setScope.mock.calls[0][0]).toMatchObject({ vue: 'all', playlists: [], modes: [] })
  })

  it('la barre reflète le scope reçu — elle ne garde AUCUN état committed à elle', () => {
    monter({ playlists: ['Ranked Arena'], vue: 'squad' })
    // La vue committed est celle du scope : le contrôle l'affiche sans qu'on ait
    // cliqué sur quoi que ce soit (c'est ce qui rend le retour navigateur correct).
    expect(screen.getByRole('button', { name: /Vue\s*: En escouade/ })).toBeInTheDocument()
  })
})
