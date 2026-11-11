import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'

import {
  averageOfMeasured,
  formatLeadChanges,
  formatLeadShare,
  type LeadSharePoint,
} from './model'
import { LEAD_SHARE_TEXT, type LeadShareText } from './text'

describe('formatLeadShare — pourcentage entier, tiret quand non mesuré', () => {
  it('arrondit à l’entier', () => {
    expect(formatLeadShare(0.624)).toBe('62 %')
    expect(formatLeadShare(0.625)).toBe('63 %')
    expect(formatLeadShare(1)).toBe('100 %')
  })

  it('zéro MESURÉ est « 0 % » ; absent ou null est un tiret, jamais « 0 % »', () => {
    expect(formatLeadShare(0)).toBe('0 %')
    expect(formatLeadShare(undefined)).toBe('—')
    expect(formatLeadShare(null)).toBe('—')
  })
})

describe('formatLeadChanges', () => {
  const t = LEAD_SHARE_TEXT.fr
  it('singulier, pluriel, zéro (pluriel) et absent', () => {
    expect(formatLeadChanges(1, t)).toBe('1 retournement')
    expect(formatLeadChanges(3, t)).toBe('3 retournements')
    expect(formatLeadChanges(0, t)).toBe('0 retournements')
    expect(formatLeadChanges(null, t)).toBe('')
    expect(formatLeadChanges(undefined, t)).toBe('')
  })
})

describe('averageOfMeasured — un match non mesuré n’entre ni au numérateur ni au dénominateur', () => {
  const pt = (share: number | null): LeadSharePoint => ({ key: String(share), label: 'x', share })
  it('moyenne des seules parts mesurées, zéro mesuré compris', () => {
    expect(averageOfMeasured([pt(0.2), pt(null), pt(0)])).toEqual({ avg: 0.1, n: 2 })
  })
  it('aucune part mesurée : pas de moyenne', () => {
    expect(averageOfMeasured([pt(null)])).toEqual({ avg: null, n: 0 })
    expect(averageOfMeasured([])).toEqual({ avg: null, n: 0 })
  })
})

describe('LEAD_SHARE_TEXT — une langue complète ou refusée', () => {
  const keys = Object.keys(LEAD_SHARE_TEXT.fr) as Array<keyof LeadShareText>
  const variables = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort()

  it('chaque langue connue a ses textes, tous non vides', () => {
    for (const locale of KNOWN_LOCALES) {
      for (const key of keys) expect(LEAD_SHARE_TEXT[locale][key].trim().length, `${locale}.${key}`).toBeGreaterThan(0)
    }
  })

  it('les variables des gabarits sont les mêmes dans toutes les langues', () => {
    for (const locale of KNOWN_LOCALES) {
      for (const key of keys) expect(variables(LEAD_SHARE_TEXT[locale][key]), `${locale}.${key}`).toEqual(variables(LEAD_SHARE_TEXT.fr[key]))
    }
  })

  it('le nom anglais n’est pas « dominance » (déjà pris par un badge narratif)', () => {
    expect(LEAD_SHARE_TEXT.en.name.toLowerCase()).not.toContain('dominance')
  })
})
