import { describe, it, expect } from 'vitest'
import {
  buildSoloFilterLink,
  dayWindowUTC,
  encodeFilterContextParam,
} from './filterLink'
import type { FilterContextInput } from '@/lib/api/types'

// Décode le payload `?f=` (miroir de createFilterStore.decodeFromUrl) pour vérifier
// que le lien porte bien un contexte de filtre exploitable par le store solo.
function decodeF(url: string): { t: string; c: FilterContextInput } {
  const f = new URL(url, 'http://x').searchParams.get('f')!
  return JSON.parse(decodeURIComponent(atob(f)))
}

describe('encodeFilterContextParam', () => {
  it('round-trip : encode → decode conserve l’enveloppe { t, c }', () => {
    const ctx: FilterContextInput = {
      filter_mode: 'period',
      period: { start_date: null, end_date: null },
      sessions: { picked_sessions: [], gap_minutes: 120 },
      cascade: { experience_types: [], playlists: [], modes: ['Slayer'], maps: [] },
    }
    const encoded = encodeFilterContextParam('halo_infinite', ctx)
    const decoded = JSON.parse(decodeURIComponent(atob(encoded)))
    expect(decoded.t).toBe('halo_infinite')
    expect(decoded.c.cascade?.modes).toEqual(['Slayer'])
  })

  it('applique le titre par défaut si vide', () => {
    const ctx: FilterContextInput = {
      filter_mode: 'period',
      period: { start_date: null, end_date: null },
      sessions: { picked_sessions: [], gap_minutes: 120 },
      cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
    }
    const decoded = JSON.parse(decodeURIComponent(atob(encodeFilterContextParam('', ctx))))
    expect(decoded.t).toBe('halo_infinite')
  })
})

describe('buildSoloFilterLink', () => {
  it('cible la page timeseries du joueur (forme title-scoped, playerSlug encodé)', () => {
    const url = buildSoloFilterLink({ playerSlug: 'jg tm', titleSlug: 'halo_infinite' })
    expect(url.startsWith('/t/halo_infinite/players/jg%20tm/stats/timeseries?f=')).toBe(true)
  })

  it('by_mode → cascade.modes, mode période, période nulle', () => {
    const url = buildSoloFilterLink({
      playerSlug: 'p1',
      titleSlug: 'halo_infinite',
      cascade: { modes: ['Assassin'] },
    })
    const { c } = decodeF(url)
    expect(c.filter_mode).toBe('period')
    expect(c.cascade?.modes).toEqual(['Assassin'])
    expect(c.cascade?.maps).toEqual([])
    expect(c.period).toEqual({ start_date: null, end_date: null })
  })

  it('by_map → cascade.maps (nom résolu, jamais le GUID)', () => {
    const url = buildSoloFilterLink({
      playerSlug: 'p1',
      titleSlug: 'halo_infinite',
      cascade: { maps: ['Aquarius'] },
    })
    const { c } = decodeF(url)
    expect(c.cascade?.maps).toEqual(['Aquarius'])
    expect(c.cascade?.modes).toEqual([])
  })

  it('période bornée → start_date/end_date renseignés', () => {
    const url = buildSoloFilterLink({
      playerSlug: 'p1',
      titleSlug: 'halo_infinite',
      period: { start: '2026-05-30', end: '2026-05-30' },
    })
    const { c } = decodeF(url)
    expect(c.period).toEqual({ start_date: '2026-05-30', end_date: '2026-05-30' })
  })

  it('estampille le titre actif dans l’enveloppe ET dans le segment /t/', () => {
    const url = buildSoloFilterLink({ playerSlug: 'p1', titleSlug: 'halo_5' })
    expect(decodeF(url).t).toBe('halo_5')
    expect(url.startsWith('/t/halo_5/players/p1/stats/timeseries?f=')).toBe(true)
  })
})

describe('dayWindowUTC', () => {
  it('extrait la journée UTC d’un timestamp ISO', () => {
    expect(dayWindowUTC('2026-05-30T22:15:00Z')).toEqual({
      start: '2026-05-30',
      end: '2026-05-30',
    })
  })
})
