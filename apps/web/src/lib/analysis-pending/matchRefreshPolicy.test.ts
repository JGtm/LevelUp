import { describe, expect, it } from 'vitest'

import {
  MATCH_DATA_STALE_MS,
  MATCH_SOURCE_RETRY_MS,
  matchRefetchInterval,
  matchStaleTime,
} from './matchRefreshPolicy'

describe('matchRefreshPolicy', () => {
  it('30 s sur lecture en échec (prime sur l’analyse), 60 s pendant l’analyse, arrêt sinon', () => {
    expect(MATCH_SOURCE_RETRY_MS).toBe(30_000)
    expect(matchRefetchInterval({ sourceUnavailable: true, analysisPending: false })).toBe(30_000)
    expect(matchRefetchInterval({ sourceUnavailable: true, analysisPending: true })).toBe(30_000)
    expect(matchRefetchInterval({ sourceUnavailable: false, analysisPending: true })).toBe(60_000)
    expect(matchRefetchInterval({ sourceUnavailable: false, analysisPending: false })).toBe(false)
  })

  it('fraîcheur nulle dans les deux états transitoires, 10 min sinon', () => {
    expect(MATCH_DATA_STALE_MS).toBe(600_000)
    expect(matchStaleTime({ sourceUnavailable: true, analysisPending: false })).toBe(0)
    expect(matchStaleTime({ sourceUnavailable: false, analysisPending: true })).toBe(0)
    expect(matchStaleTime({ sourceUnavailable: false, analysisPending: false })).toBe(600_000)
  })
})
