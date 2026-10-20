/**
 * usageFormat.test.ts — le formatage des parts des formes partagées.
 */
import { describe, expect, it } from 'vitest'

import { formatUsagePct } from './usageFormat'

describe('formatage des parts', () => {
  it('formate une part au dixième, virgule en FR, point en EN', () => {
    expect(formatUsagePct(45.62, 'fr')).toBe('45,6 %')
    expect(formatUsagePct(45.62, 'en')).toBe('45.6%')
  })

  it('nil ≠ 0 : une part absente rend un tiret, une part nulle rend 0,0 %', () => {
    expect(formatUsagePct(null, 'fr')).toBe('—')
    expect(formatUsagePct(undefined, 'fr')).toBe('—')
    expect(formatUsagePct(0, 'fr')).toBe('0,0 %')
  })
})
