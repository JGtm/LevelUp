import { describe, expect, it } from 'vitest'
import { CAROUSEL_NEAR_END_TILES, isNearEnd, itemSpanOf } from './carousel.logic'

const metrics = (over: Partial<Parameters<typeof isNearEnd>[0]> = {}) => ({
  scrollLeft: 0,
  clientWidth: 1000,
  scrollWidth: 9000,
  itemSpan: 300,
  ...over,
})

describe('isNearEnd', () => {
  it('à 5 tuiles de la fin : vrai ; juste avant : faux', () => {
    // reste = 9000 - (scrollLeft + 1000) ; 5 tuiles = 1500
    expect(isNearEnd(metrics({ scrollLeft: 6500 }))).toBe(true) // reste 1500
    expect(isNearEnd(metrics({ scrollLeft: 6499 }))).toBe(false) // reste 1501
    expect(CAROUSEL_NEAR_END_TILES).toBe(5)
  })
  it('tout au bout : vrai', () => {
    expect(isNearEnd(metrics({ scrollLeft: 8000 }))).toBe(true)
  })
  it('début de piste : faux', () => {
    expect(isNearEnd(metrics())).toBe(false)
  })
  it('piste non mesurée (scrollWidth nul) : jamais vrai', () => {
    expect(isNearEnd(metrics({ scrollWidth: 0, clientWidth: 0, itemSpan: 0 }))).toBe(false)
  })
  it('seuil réglable', () => {
    expect(isNearEnd(metrics({ scrollLeft: 7000 }), 2)).toBe(false) // reste 1000 > 600
    expect(isNearEnd(metrics({ scrollLeft: 7500 }), 2)).toBe(true) // reste 500 <= 600
  })
})

describe('itemSpanOf', () => {
  const el = (lefts: number[], width = 288) =>
    ({ children: lefts.map((offsetLeft) => ({ offsetLeft, offsetWidth: width })) }) as unknown as HTMLElement
  it('écart entre les deux premières tuiles (largeur + espace)', () => {
    expect(itemSpanOf(el([0, 300, 600]))).toBe(300)
  })
  it('une seule tuile : sa largeur ; aucune : 0', () => {
    expect(itemSpanOf(el([0]))).toBe(288)
    expect(itemSpanOf(el([]))).toBe(0)
  })
})
