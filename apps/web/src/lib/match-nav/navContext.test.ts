import { describe, it, expect, beforeEach, vi } from 'vitest'

import {
  persistNavContext,
  readNavContext,
  clearNavContext,
  resolveNeighborsFromContext,
  filterSpecToQueryString,
  parseFilterSpecFromSearch,
  type MatchFilterSpec,
  type MatchNavContext,
} from './navContext'

const baseCtx: MatchNavContext = {
  source: 'history',
  matchIds: ['m1', 'm2', 'm3', 'm4'],
  filtersLabel: 'Classée · 7 derniers jours',
}

beforeEach(() => {
  sessionStorage.clear()
  vi.useRealTimers()
})

describe('persistNavContext / readNavContext', () => {
  it('round-trip : ce qui est persist est lu sans perte', () => {
    persistNavContext('m2', baseCtx)
    expect(readNavContext('m2')).toEqual(baseCtx)
  })

  it('lit null si aucune entrée pour le matchId', () => {
    expect(readNavContext('inconnu')).toBeNull()
  })

  it('ignore une entrée vide ou matchIds=[]', () => {
    persistNavContext('m2', { ...baseCtx, matchIds: [] })
    expect(readNavContext('m2')).toBeNull()
  })

  it('ignore quand matchId est vide', () => {
    persistNavContext('', baseCtx)
    expect(readNavContext('')).toBeNull()
  })

  it('purge automatique au-delà du TTL (24h — Phase 3)', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T10:00:00Z'))
    persistNavContext('m2', baseCtx)
    expect(readNavContext('m2')).not.toBeNull()

    // +24h +1s → expiré
    vi.setSystemTime(new Date('2026-05-06T10:00:01Z'))
    expect(readNavContext('m2')).toBeNull()
  })

  it('reste lisible juste avant le TTL (24h)', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T10:00:00Z'))
    persistNavContext('m2', baseCtx)

    // +23h59m59s → encore valide
    vi.setSystemTime(new Date('2026-05-06T09:59:59Z'))
    expect(readNavContext('m2')).toEqual(baseCtx)
  })

  it('retourne null si JSON corrompu en storage', () => {
    sessionStorage.setItem('levelup:matchNav:m2', '{ this is not json')
    expect(readNavContext('m2')).toBeNull()
  })

  it('retourne null si payload mal formé (ts manquant)', () => {
    sessionStorage.setItem('levelup:matchNav:m2', JSON.stringify({ ctx: baseCtx }))
    expect(readNavContext('m2')).toBeNull()
  })
})

describe('clearNavContext', () => {
  it('supprime l\'entrée ciblée', () => {
    persistNavContext('m2', baseCtx)
    clearNavContext('m2')
    expect(readNavContext('m2')).toBeNull()
  })

  it('idempotent sur un matchId inexistant', () => {
    expect(() => clearNavContext('inconnu')).not.toThrow()
  })
})

describe('resolveNeighborsFromContext', () => {
  it('match au milieu : prev = idx+1, next = idx-1 (chronologie DESC)', () => {
    const got = resolveNeighborsFromContext(baseCtx, 'm2')
    expect(got).toEqual({
      next_match_id: 'm1',
      prev_match_id: 'm3',
      current_index: 1,
      total: 4,
    })
  })

  it('match en tête : pas de next', () => {
    const got = resolveNeighborsFromContext(baseCtx, 'm1')
    expect(got).toEqual({
      next_match_id: null,
      prev_match_id: 'm2',
      current_index: 0,
      total: 4,
    })
  })

  it('match en queue : pas de prev', () => {
    const got = resolveNeighborsFromContext(baseCtx, 'm4')
    expect(got).toEqual({
      next_match_id: 'm3',
      prev_match_id: null,
      current_index: 3,
      total: 4,
    })
  })

  it('match absent de la liste : null (fallback API)', () => {
    expect(resolveNeighborsFromContext(baseCtx, 'm99')).toBeNull()
  })

  it('liste à 1 élément : ni prev ni next (DEDUP_MARKER)', () => {
    const got = resolveNeighborsFromContext(
      { ...baseCtx, matchIds: ['m1'] },
      'm1',
    )
    expect(got).toEqual({
      next_match_id: null,
      prev_match_id: null,
      current_index: 0,
      total: 1,
    })
  })
})

describe('filterSpecToQueryString', () => {
  it('spec null/undefined : retourne chaîne vide', () => {
    expect(filterSpecToQueryString(null)).toBe('')
    expect(filterSpecToQueryString(undefined)).toBe('')
  })

  it('spec vide : retourne chaîne vide', () => {
    expect(filterSpecToQueryString({})).toBe('')
  })

  it('mappe playlist_names → playlist, mode_categories → mode, etc.', () => {
    const spec: MatchFilterSpec = {
      playlist_names: ['Ranked Arena'],
      mode_categories: ['BTB'],
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T23:59:59Z',
      session_id: 'session-123',
      outcome: 'win',
    }
    const got = filterSpecToQueryString(spec)
    const params = new URLSearchParams(got)
    expect(params.get('playlist')).toBe('Ranked Arena')
    expect(params.get('mode')).toBe('BTB')
    expect(params.get('from')).toBe('2026-04-01T00:00:00Z')
    expect(params.get('to')).toBe('2026-05-01T23:59:59Z')
    expect(params.get('session')).toBe('session-123')
    expect(params.get('outcome')).toBe('win')
  })

  it('joint les valeurs multi par virgule (Phase 3)', () => {
    const got = filterSpecToQueryString({
      playlist_names: ['Ranked Arena', 'Big Team Battle'],
      mode_categories: ['BTB', 'Fiesta'],
    })
    const params = new URLSearchParams(got)
    expect(params.get('playlist')).toBe('Ranked Arena,Big Team Battle')
    expect(params.get('mode')).toBe('BTB,Fiesta')
  })

  it('encode correctement les espaces et caractères spéciaux', () => {
    const got = filterSpecToQueryString({ playlist_names: ['Ranked Arena'] })
    expect(got).toContain('playlist=Ranked+Arena')
  })
})

describe('parseFilterSpecFromSearch', () => {
  it('null/undefined : retourne null', () => {
    expect(parseFilterSpecFromSearch(null)).toBeNull()
    expect(parseFilterSpecFromSearch(undefined)).toBeNull()
  })

  it('search vide : retourne null', () => {
    expect(parseFilterSpecFromSearch({})).toBeNull()
  })

  it('parse depuis URLSearchParams', () => {
    const sp = new URLSearchParams('playlist=Ranked&outcome=win')
    expect(parseFilterSpecFromSearch(sp)).toEqual({
      playlist_names: ['Ranked'],
      outcome: 'win',
    })
  })

  it('parse depuis Record (TanStack Router search)', () => {
    expect(
      parseFilterSpecFromSearch({ playlist: 'Classée', mode: 'Assassin', outcome: 'loss' }),
    ).toEqual({
      playlist_names: ['Classée'],
      mode_categories: ['Assassin'],
      outcome: 'loss',
    })
  })

  it('parse multi-valeurs (virgule) en arrays (Phase 3)', () => {
    expect(
      parseFilterSpecFromSearch({ playlist: 'Ranked Arena,Big Team Battle', mode: 'BTB,Fiesta' }),
    ).toEqual({
      playlist_names: ['Ranked Arena', 'Big Team Battle'],
      mode_categories: ['BTB', 'Fiesta'],
    })
  })

  it('outcome hors whitelist : ignoré silencieusement', () => {
    expect(parseFilterSpecFromSearch({ outcome: 'invalid' })).toBeNull()
    expect(parseFilterSpecFromSearch({ playlist: 'X', outcome: 'invalid' })).toEqual({
      playlist_names: ['X'],
    })
  })

  it('round-trip toQueryString → parseFromSearch', () => {
    const orig: MatchFilterSpec = {
      playlist_names: ['Ranked Arena', 'Big Team Battle'],
      mode_categories: ['BTB'],
      date_from: '2026-04-01T00:00:00Z',
      outcome: 'win',
    }
    const qs = filterSpecToQueryString(orig)
    const parsed = parseFilterSpecFromSearch(new URLSearchParams(qs))
    expect(parsed).toEqual(orig)
  })

  it('with_player : XUID numérique valide est conservé', () => {
    expect(
      parseFilterSpecFromSearch({ with_player: '2533274791785593' }),
    ).toEqual({
      with_player_xuid: '2533274791785593',
    })
  })

  it('with_player : non-numérique est rejeté silencieusement', () => {
    expect(parseFilterSpecFromSearch({ with_player: 'abc123' })).toBeNull()
    expect(
      parseFilterSpecFromSearch({ playlist: 'Ranked', with_player: '<script>' }),
    ).toEqual({ playlist_names: ['Ranked'] })
  })

  it('round-trip with_player_xuid via toQueryString', () => {
    const qs = filterSpecToQueryString({ with_player_xuid: '2533274791785593' })
    expect(qs).toContain('with_player=2533274791785593')
    expect(parseFilterSpecFromSearch(new URLSearchParams(qs))).toEqual({
      with_player_xuid: '2533274791785593',
    })
  })
})
