import { describe, it, expect } from 'vitest'
import { offensiveLabel, defensiveLabel, activityLabel } from './combatProfileLabels'

describe('combatProfileLabels — locale-aware', () => {
  it('offensiveLabel rend FR et EN', () => {
    expect(offensiveLabel('disperse', 'fr')).toBe('Dispersé')
    expect(offensiveLabel('disperse', 'en')).toBe('Scattered')
    expect(offensiveLabel('chirurgical', 'fr')).toBe('Chirurgical')
    expect(offensiveLabel('chirurgical', 'en')).toBe('Surgical')
  })

  it('defensiveLabel rend FR et EN', () => {
    expect(defensiveLabel('inebranlable', 'fr')).toBe('Inébranlable')
    expect(defensiveLabel('inebranlable', 'en')).toBe('Unshakable')
    expect(defensiveLabel('fragile', 'en')).toBe('Fragile')
  })

  it('activityLabel rend FR et EN', () => {
    expect(activityLabel('agressif', 'fr')).toBe('Agressif')
    expect(activityLabel('agressif', 'en')).toBe('Aggressive')
    expect(activityLabel('passif', 'en')).toBe('Passive')
  })

  it('null/undefined → null', () => {
    expect(offensiveLabel(null, 'fr')).toBeNull()
    expect(defensiveLabel(undefined, 'en')).toBeNull()
    expect(activityLabel(null, 'en')).toBeNull()
  })
})
