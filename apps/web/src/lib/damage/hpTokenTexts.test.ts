/**
 * Les textes affichés qui citent la vie effective (formules des infobulles Rendement / Résistance,
 * aide des réglages) portent le jeton `{{HP}}` dans CHAQUE langue : la valeur vient du bootstrap, jamais
 * du dictionnaire.
 */
import { describe, expect, it } from 'vitest'

import { KNOWN_LOCALES } from '@/lib/i18n/generated/locales'
import { MATCH_VIEW_TEXT } from '@/features/match-view/i18n'
import { getSettingsText } from '@/features/settings/i18n'
import { getSquadText } from '@/features/squad/i18n'
import { HP_PENDING, substituteHpToken } from './effectiveHp'

describe('textes affichés avec le jeton {{HP}}', () => {
  for (const locale of KNOWN_LOCALES) {
    const texts: Record<string, string> = {
      'match radar rendement': MATCH_VIEW_TEXT[locale].radarTooltipImpact,
      'match radar résistance': MATCH_VIEW_TEXT[locale].radarTooltipSurvival,
      'escouade radar rendement': getSquadText(locale).synergyRadar.tooltip.impact,
      'escouade radar résistance': getSquadText(locale).synergyRadar.tooltip.survival,
      'réglages rendement sans assistances': getSettingsText(locale).rendementExcludeAssistsHint,
    }
    for (const [name, text] of Object.entries(texts)) {
      it(`${locale} : ${name} porte le jeton et se rend avec la vie du bootstrap`, () => {
        expect(text).toContain('{{HP}}')
        expect(substituteHpToken(text, 115)).toMatch(/115 ×|× 115|115 ?\)/)
        expect(substituteHpToken(text, 115)).not.toContain('{{HP}}')
      })
      it(`${locale} : ${name} n'affiche aucune valeur avant le bootstrap`, () => {
        const before = substituteHpToken(text, undefined)
        expect(before).toContain(HP_PENDING)
        expect(before).not.toMatch(/\b22[59]\b|\b115\b/)
      })
    }
  }
})
