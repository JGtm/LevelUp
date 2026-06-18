/**
 * Tests unitaires — periodSessionNav (helpers purs).
 *
 * Couvre :
 *  - getRailMode : 4 cas (session unique, multi, période valide, hidden)
 *  - computePrev/NextSession : navigation idx, bornes
 *  - computePrev/NextWindow : sliding window, cap aujourd'hui
 *  - daysBetween : calcul durée
 */
import { describe, it, expect } from 'vitest'
import {
  computeNextSession,
  computeNextWindow,
  computePrevSession,
  computePrevWindow,
  daysBetween,
  getRailMode,
} from './periodSessionNav'
import type { FilterContextInput, SessionOption } from '@/lib/api/types'

const DEFAULT_CTX: FilterContextInput = {
  filter_mode: 'period',
  period: { start_date: null, end_date: null },
  sessions: { picked_sessions: [], gap_minutes: 120 },
  cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
}

function mkSession(id: string, label = id): SessionOption {
  return {
    session_id: id,
    label,
    match_count: 5,
    match_count_filtered: 5,
    is_squad: false,
    started_at_utc: '2026-04-01T12:00:00Z',
    ended_at_utc: '2026-04-01T13:30:00Z',
  }
}

const SESSIONS: SessionOption[] = [
  mkSession('s-latest', '06/04 21:24'), // idx 0 = latest
  mkSession('s-mid', '05/04 18:00'),
  mkSession('s-old', '01/04 12:00'),
]

describe('getRailMode', () => {
  it('all-time quand pas de scope mais sessions disponibles', () => {
    const mode = getRailMode(DEFAULT_CTX, SESSIONS)
    expect(mode.kind).toBe('all-time')
    if (mode.kind === 'all-time') expect(mode.total).toBe(3)
  })

  it('hidden quand pas de scope ET aucune session disponible', () => {
    expect(getRailMode(DEFAULT_CTX, []).kind).toBe('hidden')
  })

  it('session quand 1 session pickée et trouvée', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      filter_mode: 'sessions',
      sessions: { picked_sessions: ['s-mid'], gap_minutes: 120 },
    }
    const mode = getRailMode(ctx, SESSIONS)
    expect(mode.kind).toBe('session')
    if (mode.kind === 'session') {
      expect(mode.session.session_id).toBe('s-mid')
      expect(mode.index).toBe(1)
      expect(mode.total).toBe(3)
    }
  })

  it('hidden quand session pickée introuvable (cache stale)', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      filter_mode: 'sessions',
      sessions: { picked_sessions: ['ghost-id'], gap_minutes: 120 },
    }
    expect(getRailMode(ctx, SESSIONS).kind).toBe('hidden')
  })

  it('multi-session quand ≥2 sessions pickées', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      filter_mode: 'sessions',
      sessions: { picked_sessions: ['s-latest', 's-mid'], gap_minutes: 120 },
    }
    const mode = getRailMode(ctx, SESSIONS)
    expect(mode.kind).toBe('multi-session')
    if (mode.kind === 'multi-session') expect(mode.count).toBe(2)
  })

  it('period quand range valide', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      period: { start_date: '2026-04-01', end_date: '2026-04-08' },
    }
    const mode = getRailMode(ctx, SESSIONS)
    expect(mode.kind).toBe('period')
    if (mode.kind === 'period') expect(mode.durationDays).toBe(7)
  })
})

describe('computePrevSession / computeNextSession', () => {
  it('prev depuis latest → idx 1', () => {
    expect(computePrevSession('s-latest', SESSIONS)?.session_id).toBe('s-mid')
  })

  it('prev depuis oldest → null (borne)', () => {
    expect(computePrevSession('s-old', SESSIONS)).toBeNull()
  })

  it('next depuis latest → null (borne)', () => {
    expect(computeNextSession('s-latest', SESSIONS)).toBeNull()
  })

  it('next depuis oldest → s-mid', () => {
    expect(computeNextSession('s-old', SESSIONS)?.session_id).toBe('s-mid')
  })

  it('id introuvable → null', () => {
    expect(computePrevSession('ghost', SESSIONS)).toBeNull()
    expect(computeNextSession('ghost', SESSIONS)).toBeNull()
  })
})

describe('daysBetween', () => {
  it('compte les jours UTC inclusifs', () => {
    expect(daysBetween('2026-04-01', '2026-04-08')).toBe(7)
  })

  it('renvoie 0 si end < start', () => {
    expect(daysBetween('2026-04-08', '2026-04-01')).toBe(0)
  })

  it('renvoie 0 sur ISO invalide', () => {
    expect(daysBetween('not-a-date', '2026-04-01')).toBe(0)
  })
})

describe('computePrevWindow', () => {
  // Sémantique : delta cohérent avec presetPeriod. [04-01, 04-08] = delta 7 →
  // précédent = newEnd (= start - 1 = 03-31), newStart = newEnd - delta = 03-24.
  // La nouvelle fenêtre a le même delta (7) que l'originale, sans chevauchement.
  it('shift de la durée vers le passé sans chevauchement', () => {
    const out = computePrevWindow({ start_date: '2026-04-01', end_date: '2026-04-08' })
    expect(out).toEqual({ start_date: '2026-03-24', end_date: '2026-03-31' })
  })

  it('null si période invalide', () => {
    expect(computePrevWindow({ start_date: null, end_date: '2026-04-08' })).toBeNull()
    expect(computePrevWindow({ start_date: 'foo', end_date: 'bar' })).toBeNull()
  })
})

describe('getRailMode — edge cases', () => {
  it('hidden si all_sessions vide même avec session pickée', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      filter_mode: 'sessions',
      sessions: { picked_sessions: ['s-1'], gap_minutes: 120 },
    }
    expect(getRailMode(ctx, []).kind).toBe('hidden')
  })

  it('all-time si période durée 0 mais sessions dispos (fallback)', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      period: { start_date: '2026-04-08', end_date: '2026-04-08' },
    }
    expect(getRailMode(ctx, SESSIONS).kind).toBe('all-time')
  })

  it('all-time si période invalide (end < start) mais sessions dispos', () => {
    const ctx: FilterContextInput = {
      ...DEFAULT_CTX,
      period: { start_date: '2026-04-10', end_date: '2026-04-01' },
    }
    expect(getRailMode(ctx, SESSIONS).kind).toBe('all-time')
  })
})

describe('computeNextWindow', () => {
  it('shift vers le futur si la fenêtre se termine avant aujourd\'hui', () => {
    const today = new Date('2026-05-03T00:00:00Z')
    const out = computeNextWindow(
      { start_date: '2026-04-01', end_date: '2026-04-08' },
      today,
    )
    expect(out).toEqual({ start_date: '2026-04-09', end_date: '2026-04-16' })
  })

  it('cap à aujourd\'hui si la nouvelle fenêtre dépasse', () => {
    const today = new Date('2026-04-12T00:00:00Z')
    const out = computeNextWindow(
      { start_date: '2026-04-01', end_date: '2026-04-08' },
      today,
    )
    expect(out).toEqual({ start_date: '2026-04-09', end_date: '2026-04-12' })
  })

  it('null si la fenêtre se termine déjà aujourd\'hui ou plus tard', () => {
    const today = new Date('2026-04-08T00:00:00Z')
    expect(
      computeNextWindow({ start_date: '2026-04-01', end_date: '2026-04-08' }, today),
    ).toBeNull()
  })
})
