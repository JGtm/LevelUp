import { describe, expect, it } from 'vitest'
import type { HomePageResponse, MatchViewResponse, RecentMatchItem } from '@/lib/api/types'
import {
  ANALYSIS_PENDING_REFETCH_MS,
  ANALYSIS_PART,
  FIRST_SYNC_REFETCH_MS,
  type AnalysisPendingCarrier,
  anyAnalysisPending,
  homeHasAnalysisPending,
  homeRefetchInterval,
  hasPendingPart,
  homeStaleTime,
  matchViewHasAnalysisPending,
  matchViewRefetchInterval,
} from './analysisPending.logic'

const tile = (pending?: boolean): RecentMatchItem => ({ match_id: 'm', analysis_pending: pending }) as RecentMatchItem

function home(recent: RecentMatchItem[], favorites: RecentMatchItem[], total = 12) {
  return {
    recent_matches: recent,
    favorite_matches: favorites,
    hero: { kpis: { total_matches: total } },
  } as unknown as HomePageResponse
}

const view = (pending?: boolean) => ({ header: { analysis_pending: pending } }) as unknown as MatchViewResponse

describe('anyAnalysisPending', () => {
  it('vrai dès qu un élément porte le signal', () => {
    expect(anyAnalysisPending([tile(false), tile(true)])).toBe(true)
  })
  it('faux sur absent, faux, null ou undefined', () => {
    expect(anyAnalysisPending([tile(), tile(false), null, undefined])).toBe(false)
    expect(anyAnalysisPending(undefined)).toBe(false)
    expect(anyAnalysisPending(null)).toBe(false)
  })
})

describe('homeRefetchInterval', () => {
  it('60 s tant qu une tuile récente est en cours', () => {
    expect(homeRefetchInterval(home([tile(true)], []))).toBe(ANALYSIS_PENDING_REFETCH_MS)
  })
  it('60 s si seule une tuile favorite est en cours', () => {
    expect(homeRefetchInterval(home([tile(false)], [tile(true)]))).toBe(ANALYSIS_PENDING_REFETCH_MS)
  })
  it('s arrête (false) quand plus rien n est en cours', () => {
    expect(homeRefetchInterval(home([tile(false)], [tile()]))).toBe(false)
  })
  it('garde le poll de première synchro (30 s) quand aucun match n existe', () => {
    expect(homeRefetchInterval(home([], [], 0))).toBe(FIRST_SYNC_REFETCH_MS)
    expect(homeRefetchInterval(undefined)).toBe(FIRST_SYNC_REFETCH_MS)
  })
})

describe('homeStaleTime', () => {
  it('raccourcie tant qu une analyse est en cours, normale sinon', () => {
    expect(homeStaleTime(home([tile(true)], []), 300_000)).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(homeStaleTime(home([tile(false)], []), 300_000)).toBe(300_000)
    expect(homeHasAnalysisPending(undefined)).toBe(false)
  })
})

describe('matchViewRefetchInterval', () => {
  it('60 s tant que l analyse est en cours puis false', () => {
    expect(matchViewRefetchInterval(view(true))).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(matchViewRefetchInterval(view(false))).toBe(false)
    expect(matchViewRefetchInterval(view())).toBe(false)
    expect(matchViewRefetchInterval(undefined)).toBe(false)
    expect(matchViewHasAnalysisPending(view(true))).toBe(true)
  })
})

describe('hasPendingPart', () => {
  it('vrai seulement pour une cause présente dans la liste', () => {
    const item = { analysis_pending_parts: ['kill_source', 'replay'] }
    expect(hasPendingPart(item, ANALYSIS_PART.killSource)).toBe(true)
    expect(hasPendingPart(item, ANALYSIS_PART.replay)).toBe(true)
    expect(hasPendingPart(item, ANALYSIS_PART.citations)).toBe(false)
  })
  it('faux sur liste vide, absente, null ou élément absent', () => {
    expect(hasPendingPart({ analysis_pending_parts: [] }, ANALYSIS_PART.replay)).toBe(false)
    expect(hasPendingPart({}, ANALYSIS_PART.replay)).toBe(false)
    expect(hasPendingPart({ analysis_pending_parts: null }, ANALYSIS_PART.replay)).toBe(false)
    expect(hasPendingPart(null, ANALYSIS_PART.replay)).toBe(false)
    expect(hasPendingPart(undefined, ANALYSIS_PART.replay)).toBe(false)
  })
  it('ne lit pas le booléen : analysis_pending seul ne fait aucune cause', () => {
    expect(hasPendingPart({ analysis_pending: true } as AnalysisPendingCarrier, ANALYSIS_PART.killSource)).toBe(false)
  })
  it('les trois chaînes sont celles du serveur (domain.AnalysisPart*)', () => {
    expect(ANALYSIS_PART).toEqual({ killSource: 'kill_source', citations: 'citations', replay: 'replay' })
  })
})
