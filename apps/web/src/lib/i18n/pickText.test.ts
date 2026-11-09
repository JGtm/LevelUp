import { describe, expect, it } from 'vitest'

import { DEFAULT_LOCALE, FALLBACK_LOCALE } from './locale'
import { allTexts, localeTextsOf, pickText } from './pickText'

describe('pickText', () => {
  it('rend le texte de la langue demandée', () => {
    expect(pickText({ en: 'Rocket Launcher', fr: 'Lance-roquettes' }, 'fr')).toBe('Lance-roquettes')
    expect(pickText({ en: 'Rocket Launcher', fr: 'Lance-roquettes' }, 'en')).toBe('Rocket Launcher')
  })

  it('replie sur la langue de repli quand la langue demandée est vide ou absente', () => {
    expect(pickText({ en: 'Sentinel', fr: '' }, 'fr')).toBe('Sentinel')
    expect(pickText({ en: 'Sentinel' }, 'fr')).toBe('Sentinel')
    expect(pickText({ en: 'Sentinel', fr: null }, 'fr')).toBe('Sentinel')
  })

  it('ne replie PAS sur une autre langue que celle du repli', () => {
    // Seul le français est renseigné : demander l'anglais (qui est le repli) ne rend pas le français.
    expect(FALLBACK_LOCALE).toBe('en')
    expect(pickText({ fr: 'Sentinelle' }, 'en')).toBe('')
  })

  it('rend la chaîne vide pour un libellé absent ou sans aucune langue', () => {
    expect(pickText(undefined, 'fr')).toBe('')
    expect(pickText(null, 'en')).toBe('')
    expect(pickText({}, DEFAULT_LOCALE)).toBe('')
  })

  it('accepte un objet qui porte d’autres propriétés que les langues', () => {
    const label = { en: 'Camo', fr: 'Camouflage', img: '/x.png', tinted: true, family: 'powerup_camo' }
    expect(pickText(label, 'fr')).toBe('Camouflage')
  })
})

describe('allTexts', () => {
  it('rend les textes renseignés, dans l’ordre de la liste des langues', () => {
    expect(allTexts({ en: 'Thruster', fr: 'Propulseur' }).sort()).toEqual(['Propulseur', 'Thruster'])
  })

  it('ignore les langues vides et un libellé absent', () => {
    expect(allTexts({ en: '', fr: 'Seul' })).toEqual(['Seul'])
    expect(allTexts(undefined)).toEqual([])
  })
})

describe('localeTextsOf', () => {
  it('ne garde que les propriétés de langue', () => {
    expect(localeTextsOf({ en: 'A', fr: 'B', x: 1, img: 'i' })).toEqual({ en: 'A', fr: 'B' })
  })

  it('rend un objet vide pour une source absente', () => {
    expect(localeTextsOf(null)).toEqual({})
  })
})
