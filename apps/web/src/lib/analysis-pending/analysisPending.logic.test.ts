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
  ANALYSIS_STATE,
  isAnalysisInProgress,
  homeStaleTime,
  tilesPageRefetchInterval,
  tilesPageStaleTime,
  matchViewHasAnalysisPending,
} from './analysisPending.logic'

const tile = (pending?: boolean): RecentMatchItem => ({ match_id: 'm', analysis_pending: pending }) as RecentMatchItem

function home(recent: RecentMatchItem[], total = 12) {
  return {
    recent_matches: recent,
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
    expect(homeRefetchInterval(home([tile(true)]))).toBe(ANALYSIS_PENDING_REFETCH_MS)
  })
  it('s arrête (false) quand plus rien n est en cours', () => {
    expect(homeRefetchInterval(home([tile(false), tile()]))).toBe(false)
  })
  it('garde le poll de première synchro (30 s) quand aucun match n existe', () => {
    expect(homeRefetchInterval(home([], 0))).toBe(FIRST_SYNC_REFETCH_MS)
    expect(homeRefetchInterval(undefined)).toBe(FIRST_SYNC_REFETCH_MS)
  })
})

describe('homeStaleTime', () => {
  it('raccourcie tant qu une analyse est en cours, normale sinon', () => {
    expect(homeStaleTime(home([tile(true)]), 300_000)).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(homeStaleTime(home([tile(false)]), 300_000)).toBe(300_000)
    expect(homeHasAnalysisPending(undefined)).toBe(false)
  })
})

describe('matchViewHasAnalysisPending', () => {
  it('vrai seulement quand l’en-tête le dit', () => {
    expect(matchViewHasAnalysisPending(view(true))).toBe(true)
    expect(matchViewHasAnalysisPending(view(false))).toBe(false)
    expect(matchViewHasAnalysisPending(view())).toBe(false)
    expect(matchViewHasAnalysisPending(undefined)).toBe(false)
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

describe('isAnalysisInProgress', () => {
  it("vrai seulement pour l'état « in_progress » servi par le serveur", () => {
    expect(isAnalysisInProgress({ analysis_state: ANALYSIS_STATE.inProgress })).toBe(true)
    expect(isAnalysisInProgress({ analysis_state: ANALYSIS_STATE.queued })).toBe(false)
    expect(isAnalysisInProgress({ analysis_state: '' })).toBe(false)
    expect(isAnalysisInProgress({})).toBe(false)
    expect(isAnalysisInProgress(null)).toBe(false)
    expect(isAnalysisInProgress(undefined)).toBe(false)
  })
})

describe('pages de tuiles (pages suivantes, favoris) : chacune se recharge seule', () => {
  it('60 s tant que CETTE page contient une tuile en cours, sinon false', () => {
    expect(tilesPageRefetchInterval({ items: [tile(false), tile(true)] })).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(tilesPageRefetchInterval({ items: [tile(false), tile()] })).toBe(false)
    expect(tilesPageRefetchInterval(undefined)).toBe(false)
  })
  it('fraîcheur courte seulement quand une tuile de la page est en cours', () => {
    expect(tilesPageStaleTime({ items: [tile(true)] }, 300_000)).toBe(ANALYSIS_PENDING_REFETCH_MS)
    expect(tilesPageStaleTime({ items: [tile(false)] }, 300_000)).toBe(300_000)
    expect(tilesPageStaleTime(undefined, 300_000)).toBe(300_000)
  })
})
