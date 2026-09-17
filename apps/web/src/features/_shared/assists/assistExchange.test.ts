import { describe, expect, it } from 'vitest'

import type { RelationAssists } from '@/lib/api/types'

import {
  assistSegments,
  assistShare,
  assistSortValue,
  assistVolumeLengthPct,
  assistVolumeMax,
  givenShare,
  receivedShare,
} from './assistExchange'

function assists(over: Partial<RelationAssists> = {}): RelationAssists {
  return {
    matches_measured: 4,
    my_frags: 100,
    partner_frags: 50,
    received: { total: 20, low: 5, mid: 10, high: 5 },
    given: { total: 10, low: 0, mid: 4, high: 6 },
    ...over,
  }
}

describe('assistExchange', () => {
  it('rapporte chaque sens aux frags du bon joueur', () => {
    const a = assists()
    expect(receivedShare(a)).toBeCloseTo(0.2) // 20 de mes 100 frags
    expect(givenShare(a)).toBeCloseTo(0.2) // 10 de ses 50 frags
  })

  it("n'invente pas de part sans frag", () => {
    expect(assistShare(3, 0)).toBeNull()
  })

  it("prend le plus gros volume d'un sens comme borne, en ignorant les non mesurés", () => {
    expect(assistVolumeMax([assists(), null, assists({ given: { total: 42, low: 0, mid: 42, high: 0 } })])).toBe(42)
    expect(assistVolumeMax([undefined])).toBe(0)
  })

  it("donne la longueur au VOLUME : 117 assistances dépassent 12, et 3 ne remplissent pas la barre", () => {
    const max = 422
    expect(assistVolumeLengthPct(117, max)).toBeGreaterThan(assistVolumeLengthPct(12, max))
    expect(assistVolumeLengthPct(3, max)).toBeGreaterThan(15)
    expect(assistVolumeLengthPct(3, max)).toBeLessThan(30)
    expect(assistVolumeLengthPct(max, max)).toBe(100)
    expect(assistVolumeLengthPct(0, max)).toBe(0)
  })

  it("découpe une demi-barre du centre vers l’extérieur, au prorata des tranches", () => {
    const segs = assistSegments({ total: 10, low: 0, mid: 4, high: 6 }, 10)
    expect(segs.map((s) => s.tier)).toEqual(["mid", "high"])
    expect(segs[0].widthPct).toBeCloseTo(40)
    expect(segs[1].widthPct).toBeCloseTo(60)
  })

  it("ne dessine rien sans assistance ou sans borne", () => {
    expect(assistSegments({ total: 0, low: 0, mid: 0, high: 0 }, 10)).toEqual([])
    expect(assistSegments({ total: 1, low: 1, mid: 0, high: 0 }, 0)).toEqual([])
  })

  it('trie sur les assistances échangées, non mesuré en undefined', () => {
    expect(assistSortValue(assists())).toBe(30)
    expect(assistSortValue(undefined)).toBeUndefined()
  })
})
