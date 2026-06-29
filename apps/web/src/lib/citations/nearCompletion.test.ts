import { describe, it, expect } from 'vitest'
import {
  allCitationsMastered,
  selectNearCompletion,
  NEAR_COMPLETION_DEFAULT_LIMIT,
  NEAR_COMPLETION_MIN_PCT,
} from './nearCompletion'
import type { CitationDisplayItem } from './types'

/** Fabrique un CitationDisplayItem tieré avec des défauts « entamé, non maîtrisé ». */
function item(over: Partial<CitationDisplayItem>): CitationDisplayItem {
  return {
    key: 'k',
    name: 'Citation',
    pct: 80,
    tierIndex: 1,
    tierCount: 3,
    total: 8,
    nextTierTarget: 10,
    isMastered: false,
    isNewlyMastered: false,
    source: 'infinite',
    ...over,
  }
}

describe('selectNearCompletion', () => {
  it('garde les citations tierées, entamées, non maîtrisées et proches du palier', () => {
    const res = selectNearCompletion([item({ key: 'a', pct: 90 })])
    expect(res).toHaveLength(1)
    expect(res[0].item.key).toBe('a')
    expect(res[0].remaining).toBe(2) // 10 - 8
    expect(res[0].isFinalTier).toBe(false) // tierIndex 1 < tierCount-1 (2)
  })

  it('exclut les maîtrisées, non entamées, non tierées et sous le seuil', () => {
    const res = selectNearCompletion([
      item({ key: 'mastered', isMastered: true, pct: 100 }),
      item({ key: 'untouched', total: 0, pct: 0 }),
      item({ key: 'untiered', tierCount: 0 }),
      item({ key: 'far', pct: NEAR_COMPLETION_MIN_PCT - 1 }),
    ])
    expect(res).toHaveLength(0)
  })

  it('exclut une donnée incohérente où le prochain palier est <= total', () => {
    const res = selectNearCompletion([item({ pct: 95, total: 10, nextTierTarget: 10 })])
    expect(res).toHaveLength(0)
  })

  it('trie par proximité décroissante (pct) en premier', () => {
    const res = selectNearCompletion([
      item({ key: 'low', pct: 75 }),
      item({ key: 'high', pct: 99 }),
      item({ key: 'mid', pct: 85 }),
    ])
    expect(res.map((r) => r.item.key)).toEqual(['high', 'mid', 'low'])
  })

  it('à proximité égale, privilégie le dernier palier (franchir = maîtriser)', () => {
    const res = selectNearCompletion([
      item({ key: 'midTier', pct: 90, tierIndex: 1, tierCount: 3 }),
      item({ key: 'finalTier', pct: 90, tierIndex: 2, tierCount: 3 }),
    ])
    expect(res[0].item.key).toBe('finalTier')
    expect(res[0].isFinalTier).toBe(true)
  })

  it('à proximité et palier égaux, privilégie le moins d’unités restantes', () => {
    const res = selectNearCompletion([
      item({ key: 'far', pct: 90, total: 80, nextTierTarget: 100 }), // remaining 20
      item({ key: 'near', pct: 90, total: 9, nextTierTarget: 10 }), // remaining 1
    ])
    expect(res[0].item.key).toBe('near')
  })

  it('respecte la limite', () => {
    const many = Array.from({ length: 10 }, (_, i) => item({ key: `c${i}`, pct: 70 + i }))
    expect(selectNearCompletion(many, 6)).toHaveLength(6)
    expect(selectNearCompletion(many, 0)).toHaveLength(0)
  })

  it('plafonne à 5 tuiles par défaut (une seule ligne)', () => {
    expect(NEAR_COMPLETION_DEFAULT_LIMIT).toBe(5)
    const many = Array.from({ length: 10 }, (_, i) => item({ key: `c${i}`, pct: 70 + i }))
    expect(selectNearCompletion(many)).toHaveLength(5)
  })

  it('couvre la source native H5 (mêmes champs view-model)', () => {
    const res = selectNearCompletion([
      item({ key: 'h5', source: 'native', pct: 88, tierIndex: 3, tierCount: 5, total: 440, nextTierTarget: 500 }),
    ])
    expect(res).toHaveLength(1)
    expect(res[0].remaining).toBe(60)
  })
})

describe('allCitationsMastered', () => {
  it('vrai quand toutes les citations tiérées sont maîtrisées', () => {
    expect(
      allCitationsMastered([
        item({ key: 'a', isMastered: true }),
        item({ key: 'b', isMastered: true }),
      ]),
    ).toBe(true)
  })

  it('faux dès qu’une citation tiérée reste en cours', () => {
    expect(
      allCitationsMastered([
        item({ key: 'a', isMastered: true }),
        item({ key: 'b', isMastered: false }),
      ]),
    ).toBe(false)
  })

  it('ignore les citations non tiérées (tierCount 0) pour juger la maîtrise', () => {
    expect(
      allCitationsMastered([
        item({ key: 'tiered', isMastered: true }),
        item({ key: 'untiered', tierCount: 0, isMastered: false }),
      ]),
    ).toBe(true)
  })

  it('faux quand aucune citation tiérée n’existe (rien à célébrer)', () => {
    expect(allCitationsMastered([])).toBe(false)
    expect(allCitationsMastered([item({ key: 'untiered', tierCount: 0 })])).toBe(false)
  })
})
