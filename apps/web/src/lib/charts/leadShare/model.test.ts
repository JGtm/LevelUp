import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/locale'

import { formatLeadChanges, formatLeadShare } from './model'
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

describe('formatLeadChanges — la règle de pluriel de la langue', () => {
  const fr = LEAD_SHARE_TEXT.fr
  const en = LEAD_SHARE_TEXT.en

  it('français : zéro et un sont au singulier, deux et plus au pluriel', () => {
    expect(formatLeadChanges(0, fr, 'fr')).toBe('0 retournement')
    expect(formatLeadChanges(1, fr, 'fr')).toBe('1 retournement')
    expect(formatLeadChanges(2, fr, 'fr')).toBe('2 retournements')
    expect(formatLeadChanges(3, fr, 'fr')).toBe('3 retournements')
  })

  it('anglais : zéro est au pluriel, un au singulier', () => {
    expect(formatLeadChanges(0, en, 'en')).toBe('0 lead changes')
    expect(formatLeadChanges(1, en, 'en')).toBe('1 lead change')
    expect(formatLeadChanges(2, en, 'en')).toBe('2 lead changes')
  })

  it('absent : rien à dire', () => {
    expect(formatLeadChanges(null, fr, 'fr')).toBe('')
    expect(formatLeadChanges(undefined, fr, 'fr')).toBe('')
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

  it('l’infobulle du tiret est générique : elle ne nomme aucune cause particulière', () => {
    expect(LEAD_SHARE_TEXT.fr.notMeasured).toBe('Pas de chiffre pour ce match.')
    expect(LEAD_SHARE_TEXT.en.notMeasured).toBe('No figure for this match.')
    for (const locale of KNOWN_LOCALES) {
      expect(LEAD_SHARE_TEXT[locale].notMeasured, locale).not.toMatch(/trop peu|too rarely|avance|moves/i)
    }
  })

  it('l’aide garde l’explication des modes à peu de points, comme exemple et non comme cause unique', () => {
    expect(LEAD_SHARE_TEXT.fr.help).toMatch(/par exemple/)
    expect(LEAD_SHARE_TEXT.en.help).toMatch(/for example/)
  })

  it('le nom anglais n’est pas « dominance » (déjà pris par un badge narratif)', () => {
    expect(LEAD_SHARE_TEXT.en.name.toLowerCase()).not.toContain('dominance')
  })
})
