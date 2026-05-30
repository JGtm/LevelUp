/**
 * Tests unitaires — buildAscensionTips.
 */
import { describe, it, expect } from 'vitest'
import { buildAscensionTips } from './tips'

describe('buildAscensionTips', () => {
  it('returns the 14 entries of the Ascension section for FR', () => {
    const tips = buildAscensionTips('fr')
    expect(tips).toHaveLength(14)
  })

  it('returns the same count for EN', () => {
    const tips = buildAscensionTips('en')
    expect(tips).toHaveLength(14)
  })

  it('every tip has a stable href pointing to /help glossary anchor', () => {
    const tips = buildAscensionTips('fr')
    for (const tip of tips) {
      expect(tip.href).toMatch(/^\/help\?tab=glossary#glossary-entry-/)
    }
  })

  it('shortDef collapses newlines and is bounded', () => {
    const tips = buildAscensionTips('fr')
    for (const tip of tips) {
      expect(tip.shortDef.length).toBeLessThanOrEqual(181) // 180 + ellipsis
      expect(tip.shortDef).not.toMatch(/\n/)
    }
  })

  it('includes core Ascension concepts (FR)', () => {
    const tips = buildAscensionTips('fr')
    const terms = tips.map((t) => t.term)
    expect(terms).toContain('Série (Streak)')
    expect(terms).toContain('Multiplicateur PP')
    expect(terms).toContain('Coach proactif')
  })

  it('includes core Ascension concepts (EN)', () => {
    const tips = buildAscensionTips('en')
    const terms = tips.map((t) => t.term)
    expect(terms).toContain('Streak')
    expect(terms).toContain('PP Multiplier')
    expect(terms).toContain('Proactive Coach')
  })

  it('does not produce duplicate ids', () => {
    const tips = buildAscensionTips('fr')
    const ids = new Set(tips.map((t) => t.id))
    expect(ids.size).toBe(tips.length)
  })
})
