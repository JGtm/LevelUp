import { describe, it, expect } from 'vitest'

import { formatGamertag, isBotGamertag } from './gamertag'

describe('formatGamertag', () => {
  it('returns gamertag unchanged for normal players', () => {
    expect(formatGamertag('Spartan-117')).toBe('Spartan-117')
    expect(formatGamertag('Bungie Lover 99')).toBe('Bungie Lover 99')
  })

  it('formats raw bot ids bid(N.0) as "343 Bot N"', () => {
    expect(formatGamertag('bid(5.0)')).toBe('343 Bot 5')
    expect(formatGamertag('bid(12.0)')).toBe('343 Bot 12')
  })

  it('handles bid without decimal or closing paren (defensive)', () => {
    expect(formatGamertag('bid(7)')).toBe('343 Bot 7')
    expect(formatGamertag('bid(7.0')).toBe('343 Bot 7')
  })

  it('case-insensitive on the BID prefix', () => {
    expect(formatGamertag('BID(3.0)')).toBe('343 Bot 3')
  })

  it('returns dash for null / undefined / empty', () => {
    expect(formatGamertag(null)).toBe('—')
    expect(formatGamertag(undefined)).toBe('—')
    expect(formatGamertag('')).toBe('—')
    expect(formatGamertag('   ')).toBe('—')
  })

  it('trims whitespace before formatting', () => {
    expect(formatGamertag('  bid(2.0)  ')).toBe('343 Bot 2')
  })
})

describe('isBotGamertag', () => {
  it('detects bid pattern', () => {
    expect(isBotGamertag('bid(1.0)')).toBe(true)
    expect(isBotGamertag('bid(99.0)')).toBe(true)
  })

  it('returns false for normal gamertags', () => {
    expect(isBotGamertag('Spartan-117')).toBe(false)
    expect(isBotGamertag('343 Bot 5')).toBe(false)
  })

  it('returns false for null / empty', () => {
    expect(isBotGamertag(null)).toBe(false)
    expect(isBotGamertag(undefined)).toBe(false)
    expect(isBotGamertag('')).toBe(false)
  })
})
