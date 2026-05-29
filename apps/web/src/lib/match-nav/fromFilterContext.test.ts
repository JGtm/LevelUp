import { describe, it, expect } from 'vitest'

import { filterContextToMatchFilterSpec } from './fromFilterContext'
import type { FilterContextInput } from '@/lib/api/types'

const empty: FilterContextInput = {
  filter_mode: 'period',
  period: { start_date: null, end_date: null },
  sessions: { picked_sessions: [], gap_minutes: 120 },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

describe('filterContextToMatchFilterSpec', () => {
  it('null/undefined : retourne null', () => {
    expect(filterContextToMatchFilterSpec(null)).toBeNull()
    expect(filterContextToMatchFilterSpec(undefined)).toBeNull()
  })

  it('contexte vide : retourne null', () => {
    expect(filterContextToMatchFilterSpec(empty)).toBeNull()
  })

  it('1 seule playlist : mappée en playlist_names (slice de 1)', () => {
    const ctx: FilterContextInput = {
      ...empty,
      cascade: { ...empty.cascade, playlists: ['Ranked Arena'] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      playlist_names: ['Ranked Arena'],
    })
  })

  it('multi-playlists : toutes mappées en playlist_names (Phase 3)', () => {
    const ctx: FilterContextInput = {
      ...empty,
      cascade: { ...empty.cascade, playlists: ['Ranked Arena', 'Quick Play'] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      playlist_names: ['Ranked Arena', 'Quick Play'],
    })
  })

  it('multi-modes : tous mappés en mode_categories (Phase 3)', () => {
    const ctx: FilterContextInput = {
      ...empty,
      cascade: { ...empty.cascade, modes: ['BTB', 'Fiesta'] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      mode_categories: ['BTB', 'Fiesta'],
    })
  })

  it('1 seul mode : mappé en mode_categories (slice de 1)', () => {
    const ctx: FilterContextInput = {
      ...empty,
      cascade: { ...empty.cascade, modes: ['BTB'] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      mode_categories: ['BTB'],
    })
  })

  it('period dates : mappées avec T00:00:00Z et T23:59:59Z', () => {
    const ctx: FilterContextInput = {
      ...empty,
      period: { start_date: '2026-04-01', end_date: '2026-05-01' },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T23:59:59Z',
    })
  })

  it('period déjà ISO complet : passe-thru sans modifier', () => {
    const ctx: FilterContextInput = {
      ...empty,
      period: { start_date: '2026-04-01T08:30:00Z', end_date: null },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      date_from: '2026-04-01T08:30:00Z',
    })
  })

  it('mode=sessions : picked_session_label → session_id', () => {
    const ctx: FilterContextInput = {
      ...empty,
      filter_mode: 'sessions',
      sessions: { picked_session_label: 'session-2026-04-30', picked_sessions: [] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toEqual({
      session_id: 'session-2026-04-30',
    })
  })

  it('mode=sessions : fallback solo puis squad si picked principal vide', () => {
    const solo: FilterContextInput = {
      ...empty,
      filter_mode: 'sessions',
      sessions: { picked_solo_session_label: 'solo-X', picked_sessions: [] },
    }
    expect(filterContextToMatchFilterSpec(solo)?.session_id).toBe('solo-X')

    const squad: FilterContextInput = {
      ...empty,
      filter_mode: 'sessions',
      sessions: { picked_squad_session_label: 'squad-Y', picked_sessions: [] },
    }
    expect(filterContextToMatchFilterSpec(squad)?.session_id).toBe('squad-Y')
  })

  it('mode=period : sessions ignorées même si remplies', () => {
    const ctx: FilterContextInput = {
      ...empty,
      filter_mode: 'period',
      sessions: { picked_session_label: 'should-be-ignored', picked_sessions: [] },
    }
    expect(filterContextToMatchFilterSpec(ctx)).toBeNull()
  })

  it('outcome optionnel via options : ajouté à la spec', () => {
    expect(
      filterContextToMatchFilterSpec(empty, { outcome: 'win' }),
    ).toEqual({ outcome: 'win' })
  })

  it('combinaison complète', () => {
    const ctx: FilterContextInput = {
      ...empty,
      cascade: { ...empty.cascade, playlists: ['Ranked Arena'], modes: ['Ranked'] },
      period: { start_date: '2026-04-01', end_date: '2026-05-01' },
    }
    expect(
      filterContextToMatchFilterSpec(ctx, { outcome: 'loss' }),
    ).toEqual({
      playlist_names: ['Ranked Arena'],
      mode_categories: ['Ranked'],
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T23:59:59Z',
      outcome: 'loss',
    })
  })
})
