/**
 * Tests — cardDensity : compacte si et seulement si `mode_category === 'BTB'` (D1).
 */
import { describe, expect, it } from 'vitest'

import { cardDensity } from './cardDensity'
import { GABARIT_COMPACT, GABARIT_NORMAL } from './cardGabarit'

describe('cardDensity — la densité se lit sur la catégorie de mode, et sur rien d’autre', () => {
  it('`BTB` → gabarit compact', () => {
    expect(cardDensity({ mode_category: 'BTB' })).toBe(GABARIT_COMPACT)
  })

  it.each(['Arena', 'Ranked', 'Fiesta', 'Firefight', 'Other', ''])(
    'catégorie %j → gabarit normal',
    (categorie) => {
      expect(cardDensity({ mode_category: categorie })).toBe(GABARIT_NORMAL)
    },
  )

  it('catégorie absente, nulle, en-tête nul ou absent → normal', () => {
    expect(cardDensity({})).toBe(GABARIT_NORMAL)
    expect(cardDensity({ mode_category: undefined })).toBe(GABARIT_NORMAL)
    expect(cardDensity({ mode_category: null })).toBe(GABARIT_NORMAL)
    expect(cardDensity(null)).toBe(GABARIT_NORMAL)
    expect(cardDensity(undefined)).toBe(GABARIT_NORMAL)
  })

  it('la casse et les variantes ne sont pas devinées : « btb », « BTB Heavies » → normal (la taxonomie du serveur a déjà classé)', () => {
    // `BTB Heavies` est un PRÉFIXE de pair_name ; la catégorie qu'il donne est `BTB`. Si le
    // serveur envoyait le préfixe au lieu de la catégorie, ce serait un défaut de serveur — la
    // fiche ne le rattrape pas en devinant.
    expect(cardDensity({ mode_category: 'btb' })).toBe(GABARIT_NORMAL)
    expect(cardDensity({ mode_category: 'BTB Heavies' })).toBe(GABARIT_NORMAL)
  })

  it('un nom de la chaîne de prototypes n’est pas une catégorie', () => {
    expect(cardDensity({ mode_category: 'constructor' })).toBe(GABARIT_NORMAL)
    expect(cardDensity({ mode_category: '__proto__' })).toBe(GABARIT_NORMAL)
  })

  it('aucun repli sur les effectifs : la fonction ne reçoit que l’en-tête — un 12v12 sans catégorie reste normal', () => {
    // Le seul paramètre est l'en-tête : il n'existe aucun canal par lequel un nombre de
    // sièges, de joueurs ou de lignes de tableau pourrait entrer dans la décision.
    expect(cardDensity.length).toBe(1)
    expect(cardDensity({ start_time: '2026-07-24T20:00:00Z', mode_category: '' })).toBe(GABARIT_NORMAL)
  })
})
