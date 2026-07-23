/**
 * Tests computeCompareScale — bornes d'axe partagées A/B (échelles comparables en mode
 * comparaison). Fonction pure : on vérifie la combinaison des bornes des deux sessions.
 */
import { describe, expect, it } from 'vitest'

import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'

import { computeCompareScale } from './_compareScale'

function match(o: Partial<SessionDetailMatchRow>): SessionDetailMatchRow {
  return {
    match_id: 'm',
    start_time: '2026-01-01T00:00:00Z',
    outcome: 2,
    playlist_name: 'P',
    pair_name: 'Slayer',
    is_ranked: false,
    kills: 0,
    deaths: 0,
    assists: 0,
    kda: undefined,
    accuracy: undefined,
    personal_score: undefined,
    performance_score: undefined,
    session_label: undefined,
    dominant_category: undefined,
    offensive_conversion: undefined,
    defensive_resistance: undefined,
    ...o,
  }
}

const entry = (scores: number[]): SessionCompareEntry =>
  ({ match_series: scores.map((s, i) => ({ index: i, engagement_score: s })) }) as unknown as SessionCompareEntry

// Session A : 2 matchs Slayer. Session B : 1 match CTF.
const A = [
  match({ start_time: 'a1', kills: 5, deaths: 2, assists: 3, duration_seconds: 60, placement: 1, lobby_size: 8 }),
  match({ start_time: 'a2', kills: 1, deaths: 4, assists: 1, duration_seconds: 60, placement: 2, lobby_size: 8 }),
]
const B = [
  match({
    start_time: 'b1',
    kills: 10,
    deaths: 2,
    assists: 5,
    duration_seconds: 120,
    placement: 5,
    lobby_size: 12,
    pair_name: 'CTF',
  }),
]

describe('computeCompareScale', () => {
  it('combine les bornes des deux sessions', () => {
    const s = computeCompareScale(A, entry([-1.2, 2.0]), B, entry([0.5]), 225)

    // Net cumulé : A=[+3, 0], B=[+8] → [min∪0, max] = [0, 8].
    expect(s.netScore).toEqual([0, 8])
    // FDA/min : A frags 3/assists 2/morts 3 ; B frags 5/assists 2.5/morts 1 → [−3, 5].
    expect(s.fdaMinute).toEqual([-3, 5])
    // Engagement : valeurs A+B = [-1.2, 2, 0.5] → [min∪0, max∪0] = [-1.2, 2].
    expect(s.engagement).toEqual([-1.2, 2])
    // Placements : max compte = 1 ; axe X commun = max(8, 12) = 12.
    expect(s.placementMaxCount).toBe(1)
    expect(s.placementAxisMax).toBe(12)
    // Modes : A=Slayer×2, B=CTF×1 → max compte = 2.
    expect(s.modeMaxCount).toBe(2)
  })

  it('sessions vides → aucune borne (auto-scale partout)', () => {
    const s = computeCompareScale([], null, [], null, 225)
    expect(s.netScore).toBeUndefined()
    expect(s.fdaMinute).toBeUndefined()
    expect(s.engagement).toBeUndefined()
    expect(s.placementMaxCount).toBeUndefined()
    expect(s.modeMaxCount).toBeUndefined()
  })
})
