/**
 * engagementGap.test.ts — contribution en événements d'un point d'engagement.
 *
 * engagementGapEvents(résidu évén./min, durée s) = résidu × (durée/60). null si
 * un terme est absent/non-fini (report D5).
 */
import { describe, it, expect } from 'vitest'

import { engagementGapEvents } from './engagementGap'

describe('engagementGapEvents', () => {
  it('événements = résidu (évén./min) × durée (min)', () => {
    // 3 évén./min de plus que l'attendu sur 10 min (600 s) → 30 événements.
    expect(engagementGapEvents(3, 600)).toBe(30)
    // Déficit : -2 évén./min sur 5 min (300 s) → -10 événements.
    expect(engagementGapEvents(-2, 300)).toBe(-10)
  })

  it('résidu nul → 0 événement (pas null)', () => {
    expect(engagementGapEvents(0, 600)).toBe(0)
  })

  it('null si résidu ou durée absent/non-fini (report D5)', () => {
    expect(engagementGapEvents(null, 600)).toBeNull()
    expect(engagementGapEvents(3, null)).toBeNull()
    expect(engagementGapEvents(undefined, 600)).toBeNull()
    expect(engagementGapEvents(Infinity, 600)).toBeNull()
    expect(engagementGapEvents(3, NaN)).toBeNull()
  })
})
