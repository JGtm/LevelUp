/**
 * Tests — cardGabarit : les cotes des deux gabarits de fiche, valeur pour valeur.
 *
 * Le gabarit normal est celui du triplet du panneau du joueur et de l'encadré « Éliminé » du bloc
 * « À l'instant » : si une cote change, la fixation DOM du panneau casse — ce test dit LAQUELLE,
 * sans lire le HTML.
 */
import { describe, expect, it } from 'vitest'

import { GABARIT_COMPACT, GABARIT_NORMAL } from './cardGabarit'

describe('cardGabarit', () => {
  it('GABARIT_NORMAL = les cotes d’une fiche à corps de 35 (40 / 30 / 15 / 14 / 16 / 46 / 3)', () => {
    expect(GABARIT_NORMAL).toEqual({
      weaponCellW: 40,
      showScore: true,
      scoreCellW: 30,
      countCellW: 15,
      gaugeShieldPx: 5,
      gaugeHealthPx: 3,
      bodyPx: 35,
      iconGrenadePx: 14,
      iconAbilityPx: 16,
      watermarkPx: 46,
      boltCount: 3,
      namePx: 11.5,
    })
  })

  it('GABARIT_COMPACT = la tuile A2 : une cellule d’arme de 48, corps de 31, deux éclairs, filigrane 34', () => {
    expect(GABARIT_COMPACT.weaponCellW).toBe(48)
    expect(GABARIT_COMPACT.bodyPx).toBe(31)
    expect(GABARIT_COMPACT.gaugeShieldPx + GABARIT_COMPACT.gaugeHealthPx).toBe(6)
    expect(GABARIT_COMPACT.countCellW).toBe(10)
    expect(GABARIT_COMPACT.boltCount).toBe(2)
    expect(GABARIT_COMPACT.watermarkPx).toBe(34)
    // Ce qui quitte la tuile : le score — en infobulle du triplet.
    expect(GABARIT_COMPACT.showScore).toBe(false)
  })

  it('les deux gabarits sont gelés : une cote ne se corrige pas à la volée dans un composant', () => {
    expect(Object.isFrozen(GABARIT_NORMAL)).toBe(true)
    expect(Object.isFrozen(GABARIT_COMPACT)).toBe(true)
  })
})
