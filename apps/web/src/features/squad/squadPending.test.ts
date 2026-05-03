/**
 * Tests deriveSquadPending — propagation des sessions multi-select vers le
 * preview filter.
 *
 * Régression : avant ce fix, le SessionMultiSelect de l'écran Escouade vivait
 * dans un useState local et n'était envoyé qu'à la requête teammates. Le POST
 * filters/resolve (qui alimente le compteur sticky « N matchs » + les counts
 * cascade post-filtres) ignorait totalement la sélection — d'où l'impression
 * que les autres filtres n'avaient aucun effet sur les sessions et inversement.
 */
import { describe, expect, it } from 'vitest'
import { deriveSquadPending } from './squadPending'
import type { FilterContextInput } from '@/lib/api/types'

const basePending: FilterContextInput = {
  filter_mode: 'period',
  period: { start_date: '2026-04-01', end_date: '2026-04-30' },
  sessions: { picked_sessions: [], gap_minutes: 120 },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

describe('deriveSquadPending', () => {
  it('sans squad sessions : ajoute uniquement match_context=squad', () => {
    const out = deriveSquadPending(basePending, [])
    expect(out.match_context).toBe('squad')
    expect(out.filter_mode).toBe('period')
    expect(out.period).toEqual(basePending.period)
    expect(out.sessions?.picked_sessions).toEqual([])
  })

  it('avec sessions cochées : bascule en filter_mode=sessions et injecte les labels', () => {
    const labels = ['30/04/2026 18:30 (12)', '01/05/2026 14:00 (8)']
    const out = deriveSquadPending(basePending, labels)
    expect(out.filter_mode).toBe('sessions')
    expect(out.sessions?.picked_sessions).toEqual(labels)
  })

  it('avec sessions cochées : la période est neutralisée pour le preview', () => {
    const out = deriveSquadPending(basePending, ['session-A'])
    expect(out.period?.start_date).toBeNull()
    expect(out.period?.end_date).toBeNull()
  })

  it('match_context reste squad indépendamment du filter_mode dérivé', () => {
    const withSessions = deriveSquadPending(basePending, ['session-A'])
    const withoutSessions = deriveSquadPending(basePending, [])
    expect(withSessions.match_context).toBe('squad')
    expect(withoutSessions.match_context).toBe('squad')
  })

  it('préserve gap_minutes des sessions courantes', () => {
    const pending = {
      ...basePending,
      sessions: { picked_sessions: [], gap_minutes: 60 },
    }
    const out = deriveSquadPending(pending, ['session-A'])
    expect(out.sessions?.gap_minutes).toBe(60)
  })

  it('préserve la cascade quel que soit le mode', () => {
    const pending = {
      ...basePending,
      cascade: { experience_types: ['PVE'], playlists: [], modes: [], maps: [] },
    }
    const withSessions = deriveSquadPending(pending, ['session-A'])
    const withoutSessions = deriveSquadPending(pending, [])
    expect(withSessions.cascade?.experience_types).toEqual(['PVE'])
    expect(withoutSessions.cascade?.experience_types).toEqual(['PVE'])
  })
})
