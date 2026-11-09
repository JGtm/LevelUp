import { describe, expect, it } from 'vitest'
import { intlLocale } from '@/lib/formatters/intlLocale'
import { INTL_TAG } from './generated/locales'
import { DEFAULT_LOCALE, FALLBACK_LOCALE, KNOWN_LOCALES, isKnownLocale, normalizeLocale } from './locale'

describe('normalizeLocale', () => {
  it('garde une locale connue telle quelle', () => {
    expect(normalizeLocale('fr')).toBe('fr')
    expect(normalizeLocale('en')).toBe('en')
  })

  it('retient la sous-étiquette primaire, casse et séparateurs tolérés', () => {
    expect(normalizeLocale('en-US')).toBe('en')
    expect(normalizeLocale('en_us')).toBe('en')
    expect(normalizeLocale(' EN ')).toBe('en')
    expect(normalizeLocale('fr-FR')).toBe('fr')
    expect(normalizeLocale('en-GB,fr;q=0.8')).toBe('en')
  })

  it('vide, absente ou inconnue → DEFAULT_LOCALE (jamais un repli binaire)', () => {
    expect(normalizeLocale(undefined)).toBe(DEFAULT_LOCALE)
    expect(normalizeLocale(null)).toBe(DEFAULT_LOCALE)
    expect(normalizeLocale('')).toBe(DEFAULT_LOCALE)
    expect(normalizeLocale('de-DE')).toBe(DEFAULT_LOCALE)
    expect(normalizeLocale('zz')).toBe(DEFAULT_LOCALE)
  })
})

describe('liste des langues générée', () => {
  it('DEFAULT_LOCALE et FALLBACK_LOCALE sont des langues listées', () => {
    expect(isKnownLocale(DEFAULT_LOCALE)).toBe(true)
    expect(isKnownLocale(FALLBACK_LOCALE)).toBe(true)
  })

  it('chaque langue a une étiquette Intl, lue par intlLocale', () => {
    for (const locale of KNOWN_LOCALES) {
      expect(INTL_TAG[locale]).toMatch(/^[a-z]{2,3}-[A-Z0-9]{2,}/)
      expect(intlLocale(locale)).toBe(INTL_TAG[locale])
    }
  })

  it('intlLocale : fr → fr-FR, en → en-GB', () => {
    expect(intlLocale('fr')).toBe('fr-FR')
    expect(intlLocale('en')).toBe('en-GB')
  })
})
