import { beforeEach, describe, expect, it } from 'vitest'

import {
  counterDelta,
  readCountersSnapshot,
  writeCountersSnapshot,
  type CountersSnapshot,
} from './countersTrend'

const KEY = 'test-counters-snapshot'

describe('countersTrend', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trip localStorage : write puis read', () => {
    writeCountersSnapshot(KEY, { 'JGtm|events': 12, total: 30 })
    expect(readCountersSnapshot(KEY)).toEqual({ 'JGtm|events': 12, total: 30 })
  })

  it('read sur clé absente ou JSON corrompu → snapshot vide', () => {
    expect(readCountersSnapshot('absent')).toEqual({})
    localStorage.setItem(KEY, '{pas du json')
    expect(readCountersSnapshot(KEY)).toEqual({})
  })

  it('counterDelta : baisse, hausse, inchangé, première apparition', () => {
    const prev: CountersSnapshot = { events: 10, weapons: 3 }
    expect(counterDelta(prev, 'events', 4)).toBe(-6)
    expect(counterDelta(prev, 'weapons', 8)).toBe(5)
    expect(counterDelta(prev, 'events', 10)).toBeUndefined()
    expect(counterDelta(prev, 'nouvelle_cle', 7)).toBeUndefined()
  })
})
