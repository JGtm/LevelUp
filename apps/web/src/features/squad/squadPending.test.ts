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
import { deriveSquadPending, reconcileSquadSessionLabels, stripSessionCountSuffix } from './squadPending'
import type { FilterContextInput, SessionLabelEntry } from '@/lib/api/types'

function session(label: string): SessionLabelEntry {
  return { label, started_at: '2026-04-02T19:00:00Z', ended_at: '2026-04-02T23:45:00Z' }
}

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

describe('stripSessionCountSuffix', () => {
  it('retire le suffixe de comptage', () => {
    expect(stripSessionCountSuffix('02/04/2026 19:00–23:45 (13)')).toBe('02/04/2026 19:00–23:45')
    expect(stripSessionCountSuffix('02/04/2026 19:00 (5)')).toBe('02/04/2026 19:00')
  })

  it('idempotent quand aucun suffixe', () => {
    expect(stripSessionCountSuffix('02/04/2026 19:00–23:45')).toBe('02/04/2026 19:00–23:45')
  })
})

describe('reconcileSquadSessionLabels', () => {
  it('remappe un label dont le suffixe a dérivé (zombie de sync) vers la forme courante', () => {
    // L'utilisateur avait pické "(13)" ; un sync a porté le compte à "(15)".
    const picked = ['02/04/2026 19:00–23:45 (13)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('droppe un label introuvable (session disparue) — fix du compteur "minimum 2"', () => {
    const picked = ['session-zombie (3)', '02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('droppe le dernier zombie → liste vide (fix du "1 session" après tout désélectionner)', () => {
    const picked = ['session-zombie (3)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual([])
  })

  it('déduplique deux labels pointant sur la même session', () => {
    const picked = ['02/04/2026 19:00–23:45 (13)', '02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['02/04/2026 19:00–23:45 (15)'])
  })

  it('préserve une sélection déjà à jour (no-op)', () => {
    const picked = ['02/04/2026 19:00–23:45 (15)']
    const current = [session('02/04/2026 19:00–23:45 (15)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(picked)
  })

  it('ne touche à rien si la liste de sessions est vide (chargement en cours)', () => {
    const picked = ['02/04/2026 19:00–23:45 (13)']
    expect(reconcileSquadSessionLabels(picked, [])).toEqual(picked)
  })

  it('préserve l\'ordre des labels valides', () => {
    const picked = ['B (2)', 'A (1)']
    const current = [session('A (9)'), session('B (9)')]
    expect(reconcileSquadSessionLabels(picked, current)).toEqual(['B (9)', 'A (9)'])
  })
})
