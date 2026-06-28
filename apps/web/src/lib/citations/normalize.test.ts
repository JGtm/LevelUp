import { describe, it, expect } from 'vitest'
import { normalizeInfinitePage, normalizeNativeTotals } from './normalize'
import type {
  CitationsPageResponse,
  NativeCommendationsTotalsResponse,
} from '@/lib/api/types'

describe('normalizeInfinitePage', () => {
  it('maps fields, passes group.completed, uses citations.length as denominator', () => {
    const resp: CitationsPageResponse = {
      categories: ['multikill'],
      citations: [{} as never, {} as never, {} as never], // 3 → itemsTotal
      total_count: 99,
      citations_by_category: [
        {
          category: 'multikill',
          completed: 1,
          total: 2,
          items: [
            {
              name_norm: 'a', name_display: 'Alpha', category: 'multikill',
              total: 45, tier_count: 5, earned_tiers: 3, next_tier_target: 50,
              mastery_pct: 60, image_url: 'img', description: 'desc',
            },
            {
              name_norm: 'b', name_display: 'Bravo', category: 'multikill',
              total: 120, tier_count: 5, earned_tiers: 5, next_tier_target: 0,
              mastery_pct: 100,
            },
          ],
        },
      ],
    }
    const vm = normalizeInfinitePage(resp)
    expect(vm.source).toBe('infinite')
    expect(vm.hasFilters).toBe(true)
    expect(vm.itemsTotal).toBe(3)
    expect(vm.masteredTotal).toBe(1)
    const [a, b] = vm.categories[0].items
    expect(a).toMatchObject({ key: 'a', name: 'Alpha', pct: 60, tierIndex: 3, tierCount: 5, nextTierTarget: 50, imageUrl: 'img', description: 'desc', isMastered: false })
    expect(b.isMastered).toBe(true)
  })

  it('guards null arrays', () => {
    const vm = normalizeInfinitePage({ categories: null, citations: null, citations_by_category: null, total_count: 0 })
    expect(vm.categories).toEqual([])
    expect(vm.itemsTotal).toBe(0)
    expect(vm.masteredTotal).toBe(0)
  })
})

describe('normalizeNativeTotals', () => {
  it('maps fields, derives completed from isMastered, uses total_count denominator', () => {
    const resp: NativeCommendationsTotalsResponse = {
      total_count: 3,
      categories: [
        {
          category: 'MULTIPLAYER',
          items: [
            { id: 'a', name: 'Kills', category: 'MULTIPLAYER', total: 45, progress_pct: 60, tier_index: 3, tier_count: 5, next_tier_target: 50, icon_url: 'i' },
            { id: 'b', name: 'Assists', category: 'MULTIPLAYER', total: 120, progress_pct: 100, tier_index: 5, tier_count: 5, is_mastered: true },
          ],
        },
      ],
    }
    const vm = normalizeNativeTotals(resp)
    expect(vm.source).toBe('native')
    expect(vm.hasFilters).toBe(false)
    expect(vm.itemsTotal).toBe(3)
    expect(vm.categories[0].completed).toBe(1) // derived from isMastered
    expect(vm.masteredTotal).toBe(1)
    const [a, b] = vm.categories[0].items
    expect(a).toMatchObject({ key: 'a', name: 'Kills', pct: 60, tierIndex: 3, tierCount: 5, imageUrl: 'i', isMastered: false })
    expect(b.isMastered).toBe(true)
  })

  it('falls back to #id8 name and guards null arrays', () => {
    const vm = normalizeNativeTotals({ total_count: 1, categories: [{ category: 'X', items: [{ id: '0123456789', name: '', category: 'X', total: 0, progress_pct: 0 }] }] })
    expect(vm.categories[0].items[0].name).toBe('#01234567')
    const empty = normalizeNativeTotals({ total_count: 0, categories: null })
    expect(empty.categories).toEqual([])
  })
})
