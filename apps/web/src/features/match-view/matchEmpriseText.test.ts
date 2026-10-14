/**
 * Tests — matchEmpriseText : même carte que la page sœur → même titre ; aide portée « le match » ;
 * textes propres à la page, FR et EN. Les mots de personne sont tenus par la garde unique
 * (`timeseries/usages/textesSansPersonne.test.ts`).
 */
import { describe, expect, it } from 'vitest'

import { EMPRISE_TEXT } from '@/features/squad/emprise/empriseStrings'
import { getSquadText } from '@/features/squad/i18n'
import { USAGES_TEXT } from '@/features/timeseries/usages/usagesText'

import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'

describe('MATCH_EMPRISE_TEXT — titres des cartes partagées', () => {
  for (const locale of ['fr', 'en'] as const) {
    const t = MATCH_EMPRISE_TEXT[locale]
    const base = EMPRISE_TEXT[locale]

    it(`${locale} : le contrôle prend le titre « par match » de l’Escouade ; fiches, frags et rendement gardent le leur`, () => {
      expect(t.emprise.control.title).toBe(base.grid.title)
      expect(t.emprise.sheets.title).toBe(base.sheets.title)
      expect(t.emprise.production.title).toBe(base.production.title)
      expect(t.emprise.yield.title).toBe(base.yield.title)
      expect(t.squad.weaponKills.title).toBe(getSquadText(locale).weaponKills.title)
      expect(t.squad.performanceCharts.fragBreakdownTitle).toBe(getSquadText(locale).performanceCharts.fragBreakdownTitle)
    })

    it(`${locale} : les aides disent la portée du match`, () => {
      const scope = locale === 'fr' ? 'sur le match' : 'in the match'
      for (const info of [t.emprise.control.info, t.emprise.sheets.info, t.emprise.production.info, t.emprise.yield.info, t.squad.weaponKills.info, t.squad.performanceCharts.fragBreakdownInfo]) {
        expect(info).toContain(scope)
      }
    })

    it(`${locale} : « Isolement » une ligne par joueur, aide de la carte des Séries temporelles précédée de la portée`, () => {
      expect(t.cards.lives.title).toBe(locale === 'fr' ? 'Isolement, par joueur' : 'Isolation, by player')
      expect(t.cards.lives.info.endsWith(USAGES_TEXT[locale].cards.lives.info)).toBe(true)
    })
  }
})

describe('MATCH_EMPRISE_TEXT — textes propres à la page', () => {
  it('sous-titre de l’intertitre d’un match filmé', () => {
    expect(MATCH_EMPRISE_TEXT.fr.own.coverage(8)).toBe('film décodé · 8 joueurs présents à la fin')
    expect(MATCH_EMPRISE_TEXT.en.own.coverage(1)).toBe('film decoded · 1 player present at the end')
  })

  it('constats de H, nommant l’équipe ou l’adversaire, sans jamais dire un inconnu', () => {
    const y = MATCH_EMPRISE_TEXT.fr.own.yield
    expect(y.noEffect(false, '1:12', 2)).toBe('Aucun temps d’effet pour l’adversaire (équipe : 1:12, 2 frags)')
    expect(y.noPickup(true, 0, 2)).toBe('Aucune arme spéciale prise par l’équipe (0 contre 2)')
  })
})
