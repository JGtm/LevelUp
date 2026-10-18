/**
 * matchPartial.logic — choix du bandeau de la vue match et fraîcheur d'une vue en cache
 * (E4, D-9 / D-10).
 */
import { describe, expect, it } from 'vitest'

import {
  MATCH_VIEW_RETRY_MS,
  matchViewPollInterval,
  matchViewStaleTime,
  selectMatchPartialBanner,
} from './matchPartial.logic'

const DIX_MINUTES = 10 * 60 * 1000

describe('selectMatchPartialBanner', () => {
  it('vue complète ou sans raison : aucun bandeau', () => {
    expect(selectMatchPartialBanner({ is_partial: false, partial_reasons: [] })).toBeNull()
    expect(selectMatchPartialBanner({})).toBeNull()
    expect(selectMatchPartialBanner({ is_partial: true, partial_reasons: [] })).toBeNull()
    expect(selectMatchPartialBanner({ is_partial: false, partial_reasons: ['events_empty'] })).toBeNull()
  })

  it('sources vides seules : bandeau partiel avec ses raisons', () => {
    expect(
      selectMatchPartialBanner({ is_partial: true, partial_reasons: ['scoreboard_empty', 'medals_empty'] }),
    ).toEqual({ kind: 'partial', reasons: ['scoreboard_empty', 'medals_empty'] })
  })

  it.each([
    'scoreboard_unavailable',
    'events_unavailable',
    'player_stats_unavailable',
    'medals_unavailable',
    'section_unavailable',
  ])('%s : bandeau « indisponible »', (code) => {
    expect(selectMatchPartialBanner({ is_partial: true, partial_reasons: [code] })).toEqual({
      kind: 'unavailable',
    })
  })

  it('panne et vide simultanés : « indisponible » prime', () => {
    expect(
      selectMatchPartialBanner({ is_partial: true, partial_reasons: ['events_empty', 'medals_unavailable'] }),
    ).toEqual({ kind: 'unavailable' })
  })

  it('un code inconnu garde le bandeau partiel', () => {
    expect(
      selectMatchPartialBanner({ is_partial: true, partial_reasons: ['autre_code'] }),
    ).toEqual({ kind: 'partial', reasons: ['autre_code'] })
  })
})

describe('matchViewStaleTime', () => {
  it("vue en panne : périmée d'emblée, le cache ne la rejoue pas", () => {
    expect(matchViewStaleTime({ is_partial: true, partial_reasons: ['scoreboard_unavailable'] })).toBe(0)
    expect(matchViewStaleTime({ is_partial: true, partial_reasons: ['section_unavailable'] })).toBe(0)
  })

  it("analyse du film en cours : la vue ne reste pas figée par la fraîcheur de 10 min", () => {
    expect(matchViewStaleTime({ is_partial: false, header: { analysis_pending: true } } as never)).toBe(0)
    expect(matchViewStaleTime({ is_partial: false, header: { analysis_pending: false } } as never)).toBe(DIX_MINUTES)
    expect(matchViewStaleTime({ is_partial: false, header: {} } as never)).toBe(DIX_MINUTES)
  })

  it('vue complète, partielle (source vide) ou absente : fraîcheur normale', () => {
    expect(matchViewStaleTime({ is_partial: false })).toBe(DIX_MINUTES)
    expect(matchViewStaleTime({ is_partial: true, partial_reasons: ['events_empty'] })).toBe(DIX_MINUTES)
    expect(matchViewStaleTime(undefined)).toBe(DIX_MINUTES)
  })
})

describe('matchViewPollInterval', () => {
  it("30 s tant qu'une source est en panne, même si l'analyse est aussi en cours", () => {
    expect(matchViewPollInterval({ is_partial: true, partial_reasons: ['scoreboard_unavailable'] })).toBe(
      MATCH_VIEW_RETRY_MS,
    )
    expect(
      matchViewPollInterval({
        is_partial: true,
        partial_reasons: ['section_unavailable'],
        header: { analysis_pending: true },
      } as never),
    ).toBe(MATCH_VIEW_RETRY_MS)
    expect(MATCH_VIEW_RETRY_MS).toBe(30_000)
  })

  it("60 s tant que l'analyse est en cours, sinon plus de rechargement", () => {
    expect(matchViewPollInterval({ is_partial: false, header: { analysis_pending: true } } as never)).toBe(60_000)
    expect(matchViewPollInterval({ is_partial: false, header: { analysis_pending: false } } as never)).toBe(false)
    expect(matchViewPollInterval({ is_partial: true, partial_reasons: ['events_empty'] })).toBe(false)
    expect(matchViewPollInterval(undefined)).toBe(false)
  })
})
