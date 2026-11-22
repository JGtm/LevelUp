/**
 * Tests — reliefPlan.logic : ce que la légende du plan dit des calques que la coupe par étage laisse entiers.
 */
import { describe, expect, it } from 'vitest'

import { getTacticalText } from './i18n'
import { calquesSansHauteur, mentionSansHauteur } from './reliefPlan.logic'

const TOUT_AFFICHE = { chaleur: true, murs: true, tueurs: true, coequipiers: true }

describe('calquesSansHauteur', () => {
  it('sans coupe (« Tout »), aucun calque n’est à nommer', () => {
    expect(calquesSansHauteur(null, TOUT_AFFICHE)).toEqual([])
  })

  it('avec une coupe, chaque calque affiché qui ne porte pas la hauteur est nommé, dans l’ordre', () => {
    expect(calquesSansHauteur(2.2, TOUT_AFFICHE)).toEqual(['chaleur', 'murs', 'tueurs', 'coequipiers'])
  })

  it('un calque absent de l’écran n’est pas nommé', () => {
    expect(calquesSansHauteur(0, { chaleur: true, murs: false, tueurs: false, coequipiers: false })).toEqual(['chaleur'])
  })

  it('la coupe « Bas » (0) est une coupe : ce n’est pas « Tout »', () => {
    expect(calquesSansHauteur(0, TOUT_AFFICHE)).toHaveLength(4)
  })

  it('rien d’affiché, rien à dire', () => {
    expect(calquesSansHauteur(2.2, { chaleur: false, murs: false, tueurs: false, coequipiers: false })).toEqual([])
  })
})

describe('mentionSansHauteur : le libellé de la maquette D.1, planche 3', () => {
  it('en français', () => {
    expect(mentionSansHauteur(getTacticalText('fr'), 'chaleur')).toBe(
      'Chaleur : ce calque ne porte pas la hauteur, il reste affiché sur tous les étages.',
    )
  })

  it('en anglais', () => {
    expect(mentionSansHauteur(getTacticalText('en'), 'chaleur')).toBe(
      'Heat: this layer carries no height and stays shown on every floor.',
    )
  })

  it('chaque calque a son nom', () => {
    const t = getTacticalText('fr')
    const noms = (['chaleur', 'murs', 'tueurs', 'coequipiers'] as const).map((c) => mentionSansHauteur(t, c).split(' : ')[0])
    expect(new Set(noms).size).toBe(4)
  })
})
