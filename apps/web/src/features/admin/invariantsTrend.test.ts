/**
 * invariantsTrend.test.ts — logique pure de la tendance du dashboard
 * Intégrité des données (snapshot + delta).
 */
import { beforeEach, describe, expect, it } from 'vitest'

import type { AdminInvariantsResponse } from '@/lib/api/types'
import {
  INVARIANTS_SNAPSHOT_KEY,
  SHARED_SCOPE_KEY,
  buildInvariantsSnapshot,
  invariantDelta,
} from './invariantsTrend'
// A8.2 : read/write locaux supprimés — la persistance passe par countersTrend
// (utilisé par le hook canonique useCounterSnapshot).
import { readCountersSnapshot, writeCountersSnapshot } from './countersTrend'

function makeResponse(): AdminInvariantsResponse {
  return {
    title_slug: 'halo_infinite',
    generated_at: '2026-06-10T18:00:00Z',
    shared_violations: [
      { key: 'pair_name_uuid', severity: 'warn', count: 12, sample: ['m1'], description: 'd' },
    ],
    shared_fail_count: 0,
    shared_warn_count: 1,
    reports: [
      {
        player_slug: 'jgtm',
        gamertag: 'JGtm',
        xuid: '1',
        violations: [
          { key: 'psa_missing', severity: 'warn', count: 79, sample: [], description: 'd' },
        ],
        fail_count: 0,
        warn_count: 1,
      },
    ],
  }
}

describe('invariantsTrend', () => {
  beforeEach(() => {
    localStorage.removeItem(INVARIANTS_SNAPSHOT_KEY)
  })

  it('buildInvariantsSnapshot indexe par scope|clé, partagé inclus', () => {
    const snap = buildInvariantsSnapshot(makeResponse())
    expect(snap[`${SHARED_SCOPE_KEY}|pair_name_uuid`]).toBe(12)
    expect(snap['jgtm|psa_missing']).toBe(79)
  })

  it('round-trip localStorage (via countersTrend, A8.2)', () => {
    const snap = buildInvariantsSnapshot(makeResponse())
    writeCountersSnapshot(INVARIANTS_SNAPSHOT_KEY, snap)
    expect(readCountersSnapshot(INVARIANTS_SNAPSHOT_KEY)).toEqual(snap)
  })

  it('readCountersSnapshot tolère un JSON corrompu', () => {
    localStorage.setItem(INVARIANTS_SNAPSHOT_KEY, '{not-json')
    expect(readCountersSnapshot(INVARIANTS_SNAPSHOT_KEY)).toEqual({})
  })

  it('invariantDelta : signe correct, undefined si inchangé ou inconnu', () => {
    const prev = { 'jgtm|psa_missing': 79, [`${SHARED_SCOPE_KEY}|pair_name_uuid`]: 12 }
    // Baisse (backfill) : delta négatif.
    expect(invariantDelta(prev, SHARED_SCOPE_KEY, 'pair_name_uuid', 1)).toBe(-11)
    // Hausse (régression) : delta positif — le signal critique du dashboard.
    expect(invariantDelta(prev, 'jgtm', 'psa_missing', 90)).toBe(11)
    // Inchangé → pas de badge.
    expect(invariantDelta(prev, 'jgtm', 'psa_missing', 79)).toBeUndefined()
    // Clé jamais vue → pas de référence.
    expect(invariantDelta(prev, 'jgtm', 'enrichment_missing', 3)).toBeUndefined()
    // Pas de collision entre scopes pour une même clé d'invariant.
    expect(invariantDelta(prev, 'choco', 'psa_missing', 90)).toBeUndefined()
  })
})
