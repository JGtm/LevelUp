import { describe, it, expect, beforeEach, vi } from 'vitest'

import {
  persistNavContext,
  readNavContext,
  clearNavContext,
  resolveNeighborsFromContext,
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

  it('purge automatique au-delà du TTL (1h)', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T10:00:00Z'))
    persistNavContext('m2', baseCtx)
    expect(readNavContext('m2')).not.toBeNull()

    vi.setSystemTime(new Date('2026-05-05T11:00:01Z'))
    expect(readNavContext('m2')).toBeNull()
  })

  it('reste lisible juste avant le TTL', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-05T10:00:00Z'))
    persistNavContext('m2', baseCtx)

    vi.setSystemTime(new Date('2026-05-05T10:59:59Z'))
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

  it('liste à 1 élément : ni prev ni next', () => {
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
