import { describe, expect, it } from 'vitest'

import type { RelationInsight } from '@/lib/api/types'

import {
  computeWhatsNew,
  hasWhatsNew,
  isNewFace,
  isReunion,
  NEW_FACE_WINDOW_DAYS,
  WHATS_NEW_MAX_PER_GROUP,
} from './whatsNew'

const NOW = Date.UTC(2026, 6, 17, 12, 0, 0) // 2026-07-17T12:00:00Z
const DAY = 86_400_000

function rel(overrides: Partial<RelationInsight>): RelationInsight {
  return {
    xuid: overrides.xuid ?? 'x',
    gamertag: overrides.gamertag ?? 'GT',
    total_matches: 5,
    teammate_matches: 3,
    teammate_wins: 2,
    teammate_win_rate: 0.5,
    enemy_matches: 2,
    enemy_wins: 1,
    enemy_win_rate: 0.5,
    avg_kda_with: 1,
    avg_kda_against: 1,
    kills_dealt: 4,
    deaths_suffered: 4,
    duel_ratio: 1,
    first_seen_at: null,
    last_seen_at: null,
    category: 'mixed',
    is_core: false,
    is_revived: false,
    badges: [],
    ...overrides,
  }
}

function isoDaysAgo(days: number): string {
  return new Date(NOW - days * DAY).toISOString()
}

describe('isNewFace', () => {
  it('vrai dans la fenêtre (bornes incluses)', () => {
    expect(isNewFace(rel({ first_seen_at: isoDaysAgo(5) }), NOW)).toBe(true)
    expect(isNewFace(rel({ first_seen_at: isoDaysAgo(NEW_FACE_WINDOW_DAYS) }), NOW)).toBe(true)
  })
  it('faux hors fenêtre / donnée absente ou invalide', () => {
    expect(isNewFace(rel({ first_seen_at: isoDaysAgo(NEW_FACE_WINDOW_DAYS + 1) }), NOW)).toBe(false)
    expect(isNewFace(rel({ first_seen_at: null }), NOW)).toBe(false)
    expect(isNewFace(rel({ first_seen_at: 'not-a-date' }), NOW)).toBe(false)
  })
})

describe('isReunion', () => {
  it('lit le flag serveur is_revived', () => {
    expect(isReunion(rel({ is_revived: true }))).toBe(true)
    expect(isReunion(rel({ is_revived: false }))).toBe(false)
  })
})

describe('computeWhatsNew', () => {
  it('sépare nouvelles têtes et retrouvailles', () => {
    const rows = [
      rel({ xuid: 'a', gamertag: 'NewA', first_seen_at: isoDaysAgo(3) }),
      rel({ xuid: 'b', gamertag: 'OldB', first_seen_at: isoDaysAgo(120) }),
      rel({ xuid: 'c', gamertag: 'RevC', is_revived: true, last_seen_at: isoDaysAgo(2) }),
    ]
    const w = computeWhatsNew(rows, NOW)
    expect(w.newFaces.players.map((r) => r.gamertag)).toEqual(['NewA'])
    expect(w.reunions.players.map((r) => r.gamertag)).toEqual(['RevC'])
    expect(hasWhatsNew(w)).toBe(true)
  })

  it('trie les nouvelles têtes du plus récent au plus ancien', () => {
    const rows = [
      rel({ xuid: 'a', gamertag: 'Older', first_seen_at: isoDaysAgo(20) }),
      rel({ xuid: 'b', gamertag: 'Newer', first_seen_at: isoDaysAgo(1) }),
    ]
    const w = computeWhatsNew(rows, NOW)
    expect(w.newFaces.players.map((r) => r.gamertag)).toEqual(['Newer', 'Older'])
  })

  it('plafonne à WHATS_NEW_MAX_PER_GROUP et expose overflow', () => {
    const rows = Array.from({ length: WHATS_NEW_MAX_PER_GROUP + 3 }, (_, i) =>
      rel({ xuid: `x${i}`, gamertag: `NF${i}`, first_seen_at: isoDaysAgo(i + 1) }),
    )
    const w = computeWhatsNew(rows, NOW)
    expect(w.newFaces.players).toHaveLength(WHATS_NEW_MAX_PER_GROUP)
    expect(w.newFaces.overflow).toBe(3)
    expect(w.newFaces.total).toBe(WHATS_NEW_MAX_PER_GROUP + 3)
  })

  it('hasWhatsNew faux quand aucun groupe ne matche', () => {
    const rows = [rel({ xuid: 'a', gamertag: 'OldOnly', first_seen_at: isoDaysAgo(300) })]
    expect(hasWhatsNew(computeWhatsNew(rows, NOW))).toBe(false)
  })
})
