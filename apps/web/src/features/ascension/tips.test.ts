/**
 * Tests unitaires — buildAscensionTips (tips de jeu, source coachingTipsManifest).
 */
import { describe, it, expect } from 'vitest'
import { buildAscensionTips } from './tips'

const CATEGORIES_FR = new Set(['Combat', 'Impact', 'Objectif', 'Score', 'Support', 'Survie'])
const CATEGORIES_EN = new Set(['Combat', 'Impact', 'Objective', 'Score', 'Support', 'Survival'])

describe('buildAscensionTips', () => {
  it('returns game tips bounded by MAX_TIPS (FR)', () => {
    const tips = buildAscensionTips('fr')
    expect(tips.length).toBeGreaterThan(0)
    expect(tips.length).toBeLessThanOrEqual(14)
  })

  it('returns the same bound for EN', () => {
    const tips = buildAscensionTips('en')
    expect(tips.length).toBeGreaterThan(0)
    expect(tips.length).toBeLessThanOrEqual(14)
  })

  it('uses a coaching category as the term (FR)', () => {
    for (const tip of buildAscensionTips('fr')) {
      expect(CATEGORIES_FR.has(tip.term)).toBe(true)
    }
  })

  it('uses a coaching category as the term (EN)', () => {
    for (const tip of buildAscensionTips('en')) {
      expect(CATEGORIES_EN.has(tip.term)).toBe(true)
    }
  })

  it('every tip carries non-empty advice and a coaching id, no glossary link', () => {
    for (const tip of buildAscensionTips('fr')) {
      expect(tip.id).toMatch(/^coaching_tips\./)
      expect(tip.shortDef.length).toBeGreaterThan(0)
      expect(tip.href).toBeUndefined()
    }
  })

  it('collapses internal whitespace and newlines in the advice', () => {
    for (const tip of buildAscensionTips('fr')) {
      expect(tip.shortDef).not.toMatch(/\s{2,}/)
      expect(tip.shortDef).not.toMatch(/\n/)
    }
  })

  it('excludes category meta-keys (title, related_signals)', () => {
    for (const tip of buildAscensionTips('fr')) {
      expect(tip.id).not.toMatch(/\.title$/)
      expect(tip.id).not.toMatch(/\.related_signals$/)
    }
  })

  it('does not produce duplicate ids', () => {
    const tips = buildAscensionTips('fr')
    const ids = new Set(tips.map((t) => t.id))
    expect(ids.size).toBe(tips.length)
  })
})
