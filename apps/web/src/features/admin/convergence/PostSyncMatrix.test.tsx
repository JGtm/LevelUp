/**
 * PostSyncMatrix.test.tsx — tri CLIENT par clic sur les en-têtes (I16).
 */
import { describe, it, expect } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { PostSyncCounters, SchedulerPlayerOutcome } from '@/lib/api/types'
import { PostSyncMatrix } from './PostSyncMatrix'

function postSync(overrides: Partial<PostSyncCounters> = {}): PostSyncCounters {
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
    duration_ms: 0,
    ...overrides,
  }
}

function makePlayer(overrides: Partial<SchedulerPlayerOutcome> & { gamertag: string; perf: number }): SchedulerPlayerOutcome {
  return {
    gamertag: overrides.gamertag,
    xuid: overrides.gamertag,
    outcome: 'ok',
    reason: '',
    attempted_at: '2026-07-24T00:00:00Z',
    duration_ms: 0,
    post_sync: postSync({ perf_scores_computed: overrides.perf }),
  }
}

/** Ordre des lignes du tbody, identifiées par le gamertag qu'elles contiennent. */
function rowOrder(names: string[]): string[] {
  const tbody = document.querySelector('tbody')
  return Array.from(tbody?.querySelectorAll('tr') ?? []).map((tr) => {
    const txt = tr.textContent ?? ''
    return names.find((n) => txt.includes(n)) ?? '?'
  })
}

describe('PostSyncMatrix — tri CLIENT par en-têtes (I16)', () => {
  const names = ['Alpha', 'Bravo', 'Charlie']

  function players(): SchedulerPlayerOutcome[] {
    return [
      makePlayer({ gamertag: 'Alpha', perf: 5 }),
      makePlayer({ gamertag: 'Bravo', perf: 20 }),
      makePlayer({ gamertag: 'Charlie', perf: 10 }),
    ]
  }

  it('sans clic : ordre serveur conservé', () => {
    render(<PostSyncMatrix players={players()} />)
    expect(rowOrder(names)).toEqual(['Alpha', 'Bravo', 'Charlie'])
  })

  it('clic sur « Perf » trie numériquement (desc au 1er clic)', () => {
    render(<PostSyncMatrix players={players()} />)
    const header = screen.getByText('Perf')
    fireEvent.click(header)
    expect(rowOrder(names)).toEqual(['Bravo', 'Charlie', 'Alpha'])
  })
})
