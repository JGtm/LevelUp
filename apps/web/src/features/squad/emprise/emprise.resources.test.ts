/**
 * emprise.resources.test.ts — GARDE-RAIL « une ressource = une entrée de liste » (lot L7.4 du plan
 * PLAN_EMPRISE_VEHICULES_2026-09-28) : toute ressource de `RESOURCE_ORDER` a ses textes dans les deux
 * langues (toutes les chaînes non vides), son jeton de couleur propre, et les unités d'exposition
 * que le Go publie (`effect_ms`, `pickups`, `aboard_ms`). Une ressource ajoutée à l'ordre sans ses
 * textes planterait au rendu d'une carte : ce test la refuse avant.
 */
import { describe, expect, it } from 'vitest'

import { RESOURCE_ORDER } from './emprise.logic'
import { EMPRISE_TEXT } from './empriseStrings'
import { resourceInk } from './resourceColors'

const LOCALES = ['fr', 'en'] as const

describe('chaque ressource de l’onglet a ses textes et sa couleur', () => {
  for (const locale of LOCALES) {
    it(`${locale} : textes complets pour ${RESOURCE_ORDER.join(', ')}`, () => {
      const t = EMPRISE_TEXT[locale]
      for (const r of RESOURCE_ORDER) {
        const res = t.resources[r]
        expect(res, `textes de la ressource ${r}`).toBeDefined()
        for (const [field, value] of Object.entries(res)) expect(value, `${r}.${field}`).not.toBe('')
      }
    })

    it(`${locale} : les trois expositions publiées par le Go`, () => {
      const exposure = EMPRISE_TEXT[locale].production.exposure
      expect(Object.keys(exposure).sort()).toEqual(['aboard_ms', 'effect_ms', 'pickups'])
      expect(exposure.aboard_ms.fmt(210_000)).toBe('3 min 30')
      expect(exposure.aboard_ms.fmt(45_000)).toBe('45 s')
    })
  }

  it('une couleur propre par ressource (jeton, jamais le repli neutre)', () => {
    const inks = RESOURCE_ORDER.map(resourceInk)
    expect(new Set(inks).size).toBe(RESOURCE_ORDER.length)
    expect(resourceInk('vehicle')).toContain('resource-vehicle')
    expect(inks).not.toContain(resourceInk('ressource_inconnue'))
  })
})
