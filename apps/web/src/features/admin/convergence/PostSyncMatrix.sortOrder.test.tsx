/**
 * Ordre des lignes de PostSyncMatrix après chaque clic d'en-tête : figé avant et après le passage du
 * tri sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { PostSyncCounters, SchedulerPlayerOutcome } from '@/lib/api/types'
import { recordSortOrders } from '@/lib/table/sortOrder.testutil'
import { PostSyncMatrix } from './PostSyncMatrix'

function postSync(over: Partial<PostSyncCounters>): PostSyncCounters {
  return {
    perf_scores_computed: 0,
    lusr_updated: 0,
    career_synced: false,
    views_refreshed: 0,
    achievements_synced: false,
    matches_promoted_friends: 0,
    engagement_scores_computed: 0,
    engagement_coefs_updated: 0,
    sessions_assigned: 0,
    citations_computed: 0,
    dominance_flags_computed: 0,
    converged_events: 0,
    converged_psa: 0,
    snapshot_ready_marked: 0,
    duration_ms: 0,
    ...over,
  }
}

function player(gamertag: string, ps: Partial<PostSyncCounters>): SchedulerPlayerOutcome {
  return {
    gamertag,
    xuid: gamertag,
    outcome: 'ok',
    reason: '',
    attempted_at: '2026-07-24T00:00:00Z',
    duration_ms: 0,
    post_sync: postSync(ps),
  }
}

const PLAYERS: SchedulerPlayerOutcome[] = [
  player('Delta', { perf_scores_computed: 4, lusr_updated: 2, career_synced: true, fatal_errors: ['x'] }),
  player('alpha', { perf_scores_computed: 9, sessions_assigned: 3, achievements_synced: true }),
  player('Charlie', { perf_scores_computed: 4, citations_computed: 5, matches_promoted_friends: 2, fatal_errors: ['a', 'b'] }),
  player('bravo', { perf_scores_computed: 0, views_refreshed: 8, engagement_scores_computed: 1, dominance_flags_computed: 6 }),
  { gamertag: 'ghost', xuid: 'ghost', outcome: 'ok', reason: '', attempted_at: '2026-07-24T00:00:00Z', duration_ms: 0 },
]

describe('PostSyncMatrix — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<PostSyncMatrix players={PLAYERS} />)
    const recording = recordSortOrders(container, [
      { name: 'Joueur' },
      { name: 'Perf' },
      { name: 'LUSR' },
      { name: 'Sessions' },
      { name: 'Citations' },
      { name: 'Dominance' },
      { name: 'Engagement' },
      { name: 'Amis' },
      { name: 'Vues' },
      { name: 'Carrière' },
      { name: 'Succès' },
      { name: 'Erreurs fatales' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "Amis": "1: Charlie > Delta > alpha > bravo | 2: Delta > alpha > bravo > Charlie | 3: Charlie > Delta > alpha > bravo | aria-sort: none,-,none,none,none,none,none,none,descending,none,none,none,none",
        "Carrière": "1: Delta > alpha > Charlie > bravo | 2: alpha > Charlie > bravo > Delta | 3: Delta > alpha > Charlie > bravo | aria-sort: none,-,none,none,none,none,none,none,none,none,descending,none,none",
        "Citations": "1: Charlie > Delta > alpha > bravo | 2: Delta > alpha > bravo > Charlie | 3: Charlie > Delta > alpha > bravo | aria-sort: none,-,none,none,none,descending,none,none,none,none,none,none,none",
        "Dominance": "1: bravo > Delta > alpha > Charlie | 2: Delta > alpha > Charlie > bravo | 3: bravo > Delta > alpha > Charlie | aria-sort: none,-,none,none,none,none,descending,none,none,none,none,none,none",
        "Engagement": "1: bravo > Delta > alpha > Charlie | 2: Delta > alpha > Charlie > bravo | 3: bravo > Delta > alpha > Charlie | aria-sort: none,-,none,none,none,none,none,descending,none,none,none,none,none",
        "Erreurs fatales": "1: Charlie > Delta > alpha > bravo | 2: alpha > bravo > Delta > Charlie | 3: Charlie > Delta > alpha > bravo | aria-sort: none,-,none,none,none,none,none,none,none,none,none,none,descending",
        "Joueur": "1: alpha > bravo > Charlie > Delta | 2: Delta > Charlie > bravo > alpha | 3: alpha > bravo > Charlie > Delta | aria-sort: ascending,-,none,none,none,none,none,none,none,none,none,none,none",
        "LUSR": "1: Delta > alpha > Charlie > bravo | 2: alpha > Charlie > bravo > Delta | 3: Delta > alpha > Charlie > bravo | aria-sort: none,-,none,descending,none,none,none,none,none,none,none,none,none",
        "Perf": "1: alpha > Delta > Charlie > bravo | 2: bravo > Delta > Charlie > alpha | 3: alpha > Delta > Charlie > bravo | aria-sort: none,-,descending,none,none,none,none,none,none,none,none,none,none",
        "Sessions": "1: alpha > Delta > Charlie > bravo | 2: Delta > Charlie > bravo > alpha | 3: alpha > Delta > Charlie > bravo | aria-sort: none,-,none,none,descending,none,none,none,none,none,none,none,none",
        "Succès": "1: alpha > Delta > Charlie > bravo | 2: Delta > Charlie > bravo > alpha | 3: alpha > Delta > Charlie > bravo | aria-sort: none,-,none,none,none,none,none,none,none,none,none,descending,none",
        "Vues": "1: bravo > Delta > alpha > Charlie | 2: Delta > alpha > Charlie > bravo | 3: bravo > Delta > alpha > Charlie | aria-sort: none,-,none,none,none,none,none,none,none,descending,none,none,none",
        "initial": "Delta > alpha > Charlie > bravo",
      }
    `)
  })
})
