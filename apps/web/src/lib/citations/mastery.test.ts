import { describe, it, expect } from 'vitest'
import { citationMastery } from './mastery'

describe('citationMastery', () => {
  it('trusts the explicit is_mastered flag (native page)', () => {
    expect(citationMastery({ is_mastered: true, tier_count: 5, tier_index: 0 })).toBe(true)
    expect(citationMastery({ is_mastered: false, tier_count: 5, tier_index: 5 })).toBe(true) // tier path still wins
  })

  it('tiered: mastered iff final tier reached (tier_index)', () => {
    expect(citationMastery({ tier_count: 5, tier_index: 5 })).toBe(true)
    expect(citationMastery({ tier_count: 5, tier_index: 6 })).toBe(true)
    expect(citationMastery({ tier_count: 5, tier_index: 4 })).toBe(false)
    expect(citationMastery({ tier_count: 5, tier_index: 0 })).toBe(false)
  })

  it('tiered: earned_tiers is the Infinite synonym of tier_index', () => {
    expect(citationMastery({ tier_count: 3, earned_tiers: 3 })).toBe(true)
    expect(citationMastery({ tier_count: 3, earned_tiers: 2 })).toBe(false)
  })

  it('untiered (tier_count 0): mastered iff progress_pct >= 100', () => {
    expect(citationMastery({ tier_count: 0, progress_pct: 100 })).toBe(true)
    expect(citationMastery({ tier_count: 0, progress_pct: 99.9 })).toBe(false)
    expect(citationMastery({ progress_pct: 100 })).toBe(true)
  })

  it('untiered: mastery_pct is the Infinite synonym of progress_pct', () => {
    expect(citationMastery({ tier_count: 0, mastery_pct: 100 })).toBe(true)
    expect(citationMastery({ mastery_pct: 50 })).toBe(false)
  })

  it('all-nullish → false', () => {
    expect(citationMastery({})).toBe(false)
    expect(citationMastery({ tier_count: null, tier_index: null, progress_pct: null })).toBe(false)
  })

  // Behavior-preservation vs the OLD inline rules of each surface.
  it('preserves the MatchCitationSnippet rule: (tc>0 && ti>=tc) || progress_pct>=100', () => {
    const old = (c: { tier_count?: number; tier_index?: number; progress_pct: number }) =>
      ((c.tier_count ?? 0) > 0 && (c.tier_index ?? 0) >= (c.tier_count ?? 0)) || c.progress_pct >= 100
    for (const c of [
      { tier_count: 5, tier_index: 5, progress_pct: 100 },
      { tier_count: 5, tier_index: 2, progress_pct: 40 },
      { tier_count: 0, tier_index: 0, progress_pct: 100 },
      { tier_count: 0, tier_index: 0, progress_pct: 30 },
    ]) {
      expect(citationMastery(c)).toBe(old(c))
    }
  })

  it('preserves the Infinite CitationItem rule on consistent data (mastery_pct>=100 ⟺ earned>=count)', () => {
    expect(citationMastery({ tier_count: 5, earned_tiers: 5, mastery_pct: 100 })).toBe(true)
    expect(citationMastery({ tier_count: 5, earned_tiers: 2, mastery_pct: 40 })).toBe(false)
  })
})
