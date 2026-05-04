/**
 * Tests i18n.ts — pickLocalized et formatUnlockedDate.
 *
 * pickLocalized doit gérer correctement tous les cas de fallback bilingue,
 * notamment quand l'API Xbox renvoie un seul des deux champs (FR-only ou
 * EN-only) ou les deux vides.
 */
import { describe, expect, it } from 'vitest'
import { formatUnlockedDate, pickLocalized } from './i18n'

describe('pickLocalized', () => {
  it('locale=fr : retourne fr quand fr non vide', () => {
    expect(pickLocalized('Hello', 'Bonjour', 'fr')).toBe('Bonjour')
  })

  it('locale=fr : fallback sur en quand fr vide', () => {
    expect(pickLocalized('Hello', '', 'fr')).toBe('Hello')
  })

  it('locale=fr : fallback sur en quand fr undefined', () => {
    expect(pickLocalized('Hello', undefined, 'fr')).toBe('Hello')
  })

  it('locale=en : retourne en quand en non vide', () => {
    expect(pickLocalized('Hello', 'Bonjour', 'en')).toBe('Hello')
  })

  it('locale=en : fallback sur fr quand en vide', () => {
    expect(pickLocalized('', 'Bonjour', 'en')).toBe('Bonjour')
  })

  it('locale=en : fallback sur fr quand en undefined', () => {
    expect(pickLocalized(undefined, 'Bonjour', 'en')).toBe('Bonjour')
  })

  it('les deux vides → string vide', () => {
    expect(pickLocalized('', '', 'fr')).toBe('')
    expect(pickLocalized('', '', 'en')).toBe('')
  })

  it('les deux undefined → string vide', () => {
    expect(pickLocalized(undefined, undefined, 'fr')).toBe('')
    expect(pickLocalized(undefined, undefined, 'en')).toBe('')
  })
})

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
