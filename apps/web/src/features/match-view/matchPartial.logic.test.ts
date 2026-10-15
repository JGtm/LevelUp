/**
 * matchPartial.logic — choix du bandeau de la vue match et fraîcheur d'une vue en cache
 * (E4, D-9 / D-10).
 */
import { describe, expect, it } from 'vitest'

import { matchViewStaleTime, selectMatchPartialBanner } from './matchPartial.logic'

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

  it('vue complète, partielle (source vide) ou absente : fraîcheur normale', () => {
    expect(matchViewStaleTime({ is_partial: false })).toBe(DIX_MINUTES)
    expect(matchViewStaleTime({ is_partial: true, partial_reasons: ['events_empty'] })).toBe(DIX_MINUTES)
    expect(matchViewStaleTime(undefined)).toBe(DIX_MINUTES)
  })
})
