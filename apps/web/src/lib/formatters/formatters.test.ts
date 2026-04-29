/**
 * Tests table-driven sur les helpers formatters (P2.6bis).
 * Note : `formatPercent` a son propre fichier de tests (percent.test.ts).
 */
import { describe, it, expect } from 'vitest'

import {
  formatDate,
  formatDateShort,
  formatDateTime,
  formatNumber,
  formatNumberFixed,
  formatRatio,
  formatKDA,
  formatDurationMMSS,
  formatDurationHMS,
} from './index'

describe('formatDate', () => {
  it('format ISO en medium FR', () => {
    const result = formatDate('2026-04-29T12:00:00Z', 'fr-FR')
    // Le résultat exact dépend du fuseau ; on vérifie qu'il contient avril
    expect(result).toMatch(/avr|avril/i)
  })

  it('format ISO en short EN', () => {
    const result = formatDate('2026-04-29T12:00:00Z', 'en-US', { dateStyle: 'short' })
    expect(result).toMatch(/4\/29\/26|04\/29\/2026/)
  })

  it('renvoie le fallback sur null/undefined/empty/invalide', () => {
    expect(formatDate(null, 'fr-FR')).toBe('—')
    expect(formatDate(undefined, 'fr-FR')).toBe('—')
    expect(formatDate('', 'fr-FR')).toBe('—')
    expect(formatDate('not-a-date', 'fr-FR')).toBe('—')
  })

  it('respecte un fallback custom', () => {
    expect(formatDate(null, 'fr-FR', undefined, 'N/A')).toBe('N/A')
  })
})

describe('formatDateShort', () => {
  it('format DD/MM FR', () => {
    expect(formatDateShort('2026-04-29')).toMatch(/29\/04/)
  })
})

describe('formatDateTime', () => {
  it('format date+time selon la locale', () => {
    const result = formatDateTime('2026-04-29T12:00:00Z', 'fr-FR')
    // Doit contenir une date et une heure
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/)
  })

  it('fallback sur null', () => {
    expect(formatDateTime(null, 'fr-FR')).toBe('—')
  })
})

describe('formatNumber', () => {
  it('format avec séparateurs FR', () => {
    expect(formatNumber(12345, 'fr-FR', 0)).toMatch(/12.345/)
    expect(formatNumber(12345.6, 'fr-FR', 1)).toMatch(/12.345,6/)
  })

  it('format avec séparateurs EN', () => {
    expect(formatNumber(12345, 'en-US', 0)).toBe('12,345')
  })

  it('fallback sur null/NaN', () => {
    expect(formatNumber(null, 'fr-FR')).toBe('—')
    expect(formatNumber(NaN, 'fr-FR')).toBe('—')
  })
})

describe('formatNumberFixed', () => {
  it('toFixed sans séparateurs locale', () => {
    expect(formatNumberFixed(12.345, 1)).toBe('12.3')
    expect(formatNumberFixed(12.345, 2)).toBe('12.35')
  })

  it('fallback sur null/NaN/Infinity', () => {
    expect(formatNumberFixed(null)).toBe('—')
    expect(formatNumberFixed(NaN)).toBe('—')
    expect(formatNumberFixed(Infinity)).toBe('—')
  })
})

describe('formatRatio / formatKDA', () => {
  it('2 décimales locale-sensitive', () => {
    expect(formatRatio(2.345, 'fr-FR')).toMatch(/2,35/)
    expect(formatRatio(2.345, 'en-US')).toBe('2.35')
  })

  it('formatKDA est un alias de formatRatio', () => {
    expect(formatKDA(2.345, 'fr-FR')).toBe(formatRatio(2.345, 'fr-FR'))
  })

  it('fallback sur null', () => {
    expect(formatRatio(null, 'fr-FR')).toBe('—')
    expect(formatKDA(null, 'fr-FR')).toBe('—')
  })
})

describe('formatDurationMMSS', () => {
  it('format MM:SS standard', () => {
    expect(formatDurationMMSS(125)).toBe('2:05')
    expect(formatDurationMMSS(3661)).toBe('61:01') // pas d'heure (M >= 60 ok)
    expect(formatDurationMMSS(0)).toBe('-')
  })

  it('fallback sur invalide', () => {
    expect(formatDurationMMSS(undefined)).toBe('-')
    expect(formatDurationMMSS(null)).toBe('-')
    expect(formatDurationMMSS(-5)).toBe('-')
    expect(formatDurationMMSS(NaN)).toBe('-')
  })

  it('respecte un fallback custom', () => {
    expect(formatDurationMMSS(null, '—')).toBe('—')
  })
})

describe('formatDurationHMS', () => {
  it('format HH:MM:SS standard', () => {
    expect(formatDurationHMS(3661)).toBe('1:01:01')
    expect(formatDurationHMS(125)).toBe('0:02:05')
  })

  it('fallback sur invalide', () => {
    expect(formatDurationHMS(undefined)).toBe('-')
    expect(formatDurationHMS(0)).toBe('-')
  })
})
