import { beforeEach, describe, expect, it } from 'vitest'

import { commonManifest } from './generated/common'
import { formatMessage, resetFormatterCache } from './format'

describe('formatMessage', () => {
  beforeEach(() => {
    resetFormatterCache()
  })

  it('resout une cle simple en FR', () => {
    expect(formatMessage(commonManifest, 'common.period.last_1y', 'fr')).toBe('Dernière année')
  })

  it('resout une cle simple en EN', () => {
    expect(formatMessage(commonManifest, 'common.period.last_1y', 'en')).toBe('Last year')
  })

  it('applique pluralisation ICU one en FR', () => {
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'fr', { n: 1 })).toBe('1 match')
  })

  it('applique pluralisation ICU other en FR', () => {
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'fr', { n: 5 })).toBe('5 matchs')
  })

  it('applique pluralisation ICU one en EN', () => {
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 1 })).toBe('1 match')
  })

  it('applique pluralisation ICU other en EN', () => {
    expect(formatMessage(commonManifest, 'common.kpi.matches_count', 'en', { n: 12 })).toBe('12 matches')
  })

  it('retourne la cle si elle est absente du manifest', () => {
    // @ts-expect-error : on teste volontairement une cle hors du type.
    expect(formatMessage(commonManifest, 'unknown.key', 'fr')).toBe('unknown.key')
  })

  it('memoize les formatters (pas de re-compilation a chaque appel)', () => {
    // Verifie indirectement : 1000 appels avec la meme cle doivent etre rapides.
    // Le formatter est cache par (locale, message).
    const start = performance.now()
    for (let i = 0; i < 1000; i++) {
      formatMessage(commonManifest, 'common.kpi.matches_count', 'fr', { n: i })
    }
    const duration = performance.now() - start
    // Avec cache, on s'attend a < 50ms pour 1000 appels. Sans cache (re-compile
    // chaque fois) ca prendrait plusieurs centaines de ms. On laisse une marge
    // confortable pour les CI lentes.
    expect(duration).toBeLessThan(500)
  })

  it('court-circuite MessageFormat si pas d\'accolades et pas de vars', () => {
    // Tres simple : la string ne contient pas d'accolades, pas d'interpolation.
    // Le code prend le chemin rapide (return message) sans creer de formatter.
    expect(formatMessage(commonManifest, 'common.outcome.win', 'fr')).toBe('Victoire')
    expect(formatMessage(commonManifest, 'common.outcome.win', 'en')).toBe('Win')
  })

  it('retourne la cle si la locale demandee est manquante (defensive)', () => {
    const partialManifest = {
      'test.partial': { fr: 'FR uniquement', en: '' },
    } as const
    expect(formatMessage(partialManifest, 'test.partial', 'en')).toBe('test.partial')
    expect(formatMessage(partialManifest, 'test.partial', 'fr')).toBe('FR uniquement')
  })
})

// ─── Test de coherence cross-reference du manifest commonManifest ────────────
//
// Conformement au PLAN_META_FOUNDATIONS_GO § 3.1.11 : on verifie que le
// manifest a bien fr ET en pour chaque cle (le build step le verifie deja
// mais on laisse un test runtime pour la garde defensive).

describe('commonManifest integrity', () => {
  it('toutes les cles ont fr ET en non vides', () => {
    for (const [key, entry] of Object.entries(commonManifest)) {
      expect(entry.fr, `cle "${key}" : fr manquant`).toBeTruthy()
      expect(entry.en, `cle "${key}" : en manquant`).toBeTruthy()
    }
  })

  it('aucune cle vide', () => {
    expect(Object.keys(commonManifest).length).toBeGreaterThan(0)
  })
})
