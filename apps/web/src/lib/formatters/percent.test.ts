/**
 * Tests table-driven sur formatPercent (ADR 0006).
 *
 * Politique transverse : tests de non-régression sur le format public.
 * Toute modification de la convention décimale doit ré-évaluer ce fichier.
 */
import { describe, it, expect } from 'vitest'

import { formatPercent, formatPercentValue } from './percent'

describe('formatPercent', () => {
  it('formate les ratios canoniques avec 1 décimale par défaut', () => {
    expect(formatPercent(0.5)).toBe('50.0 %')
    expect(formatPercent(0.5532)).toBe('55.3 %')
    expect(formatPercent(1)).toBe('100.0 %')
    expect(formatPercent(0)).toBe('0.0 %')
  })

  it('respecte le paramètre decimals', () => {
    expect(formatPercent(0.5532, 0)).toBe('55 %')
    expect(formatPercent(0.5532, 2)).toBe('55.32 %')
    expect(formatPercent(0.5532, 3)).toBe('55.320 %')
  })

  it('renvoie le fallback "—" pour null/undefined/NaN', () => {
    expect(formatPercent(null)).toBe('—')
    expect(formatPercent(undefined)).toBe('—')
    expect(formatPercent(NaN)).toBe('—')
  })

  it('respecte un fallback custom', () => {
    expect(formatPercent(null, 1, 'N/A')).toBe('N/A')
    expect(formatPercent(NaN, 0, '')).toBe('')
  })

  it('gère les valeurs hors borne sans coercition', () => {
    // L'helper formate ce qu'on lui passe — pas de clamp.
    // Le clamp métier doit se faire en amont si nécessaire.
    expect(formatPercent(1.5)).toBe('150.0 %')
    expect(formatPercent(-0.1)).toBe('-10.0 %')
  })
})

describe('formatPercentValue', () => {
  it('renvoie la valeur sans suffixe %', () => {
    expect(formatPercentValue(0.5532)).toBe('55.3')
    expect(formatPercentValue(0.5532, 2)).toBe('55.32')
    expect(formatPercentValue(1)).toBe('100.0')
  })

  it('respecte le fallback', () => {
    expect(formatPercentValue(null)).toBe('—')
    expect(formatPercentValue(NaN, 1, '')).toBe('')
  })
})
