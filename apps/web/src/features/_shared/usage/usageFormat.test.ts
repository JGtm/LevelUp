/**
 * usageFormat.test.ts — le formatage des parts, cadences et comptes du bloc « usages
 * d'équipement, armes spéciales et objectifs » (S3). Extrait de `usageLogic.test.ts` le
 * 2026-09-09 (étape E5.1bis, scission de taille — CLAUDE.md n°5) au moment du déménagement
 * du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import { formatUsageCount, formatUsagePct, formatUsageRate } from './usageFormat'

describe('formatage des parts, cadences et comptes', () => {
  it('formate une part au dixième, virgule en FR, point en EN', () => {
    expect(formatUsagePct(45.62, 'fr')).toBe('45,6 %')
    expect(formatUsagePct(45.62, 'en')).toBe('45.6%')
  })

  it('nil ≠ 0 : une part absente rend un tiret, une part nulle rend 0,0 %', () => {
    expect(formatUsagePct(null, 'fr')).toBe('—')
    expect(formatUsagePct(undefined, 'fr')).toBe('—')
    expect(formatUsagePct(0, 'fr')).toBe('0,0 %')
  })

  it('formate une cadence à une décimale, tiret quand absente', () => {
    expect(formatUsageRate(1.26, 'fr')).toBe('1,3')
    expect(formatUsageRate(undefined, 'fr')).toBe('—')
  })

  it('formate une durée en m:ss — un 0 MESURÉ s écrit 0:00, jamais un tiret', () => {
    expect(formatUsageCount(125, 'fr', true)).toBe('2:05')
    expect(formatUsageCount(0, 'fr', true)).toBe('0:00')
    expect(formatUsageCount(null, 'fr', true)).toBe('—')
  })
})
