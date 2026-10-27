/**
 * Tests i18n.ts — formatUnlockedDate.
 */
import { describe, expect, it } from 'vitest'
import { formatUnlockedDate } from './i18n'

describe('formatUnlockedDate', () => {
  it('retourne null si iso absent', () => {
    expect(formatUnlockedDate(undefined, 'fr')).toBeNull()
  })

  it('retourne null si iso invalide', () => {
    expect(formatUnlockedDate('not-a-date', 'fr')).toBeNull()
  })

  it('formate FR (jour court)', () => {
    const out = formatUnlockedDate('2026-04-15T10:00:00Z', 'fr')
    expect(out).toBeTruthy()
    expect(out).toMatch(/2026/)
  })

  it('formate EN (jour court)', () => {
    const out = formatUnlockedDate('2026-04-15T10:00:00Z', 'en')
    expect(out).toBeTruthy()
    expect(out).toMatch(/2026/)
  })
})
