/**
 * Tests — cardGabarit : le gabarit NORMAL porte les cotes d'aujourd'hui, valeur pour valeur.
 *
 * Ce test est le pendant chiffré de la fixation DOM 4v4 : si une cote du gabarit normal
 * change, la fixture casse — mais ce test dit LAQUELLE, sans lire 42 Ko de HTML.
 */
import { describe, expect, it } from 'vitest'

import { GABARIT_COMPACT, GABARIT_NORMAL } from './cardGabarit'

describe('cardGabarit', () => {
  it('GABARIT_NORMAL = les constantes de la fiche au 2026-09-06 (40 / 32 / 56 / 30 / 15 / 14 / 16 / 46 / 3)', () => {
    expect(GABARIT_NORMAL).toEqual({
      weaponCells: 2,
      weaponCellW: 40,
      showAmmo: true,
      ammoCellW: 32,
      showGrenadeStock: true,
      grenadesBoxW: 56,
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
      seatGrid: false,
    })
  })

  it('GABARIT_COMPACT = la tuile A2 : une cellule d’arme de 48, corps de 31, deux éclairs, filigrane 34, grille', () => {
    expect(GABARIT_COMPACT.weaponCells).toBe(1)
    expect(GABARIT_COMPACT.weaponCellW).toBe(48)
    expect(GABARIT_COMPACT.bodyPx).toBe(31)
    expect(GABARIT_COMPACT.gaugeShieldPx + GABARIT_COMPACT.gaugeHealthPx).toBe(6)
    expect(GABARIT_COMPACT.countCellW).toBe(10)
    expect(GABARIT_COMPACT.boltCount).toBe(2)
    expect(GABARIT_COMPACT.watermarkPx).toBe(34)
    expect(GABARIT_COMPACT.seatGrid).toBe(true)
    // Ce qui quitte la tuile : munitions, stock de grenades, score — en infobulle (étape 3).
    expect(GABARIT_COMPACT.showAmmo).toBe(false)
    expect(GABARIT_COMPACT.showGrenadeStock).toBe(false)
    expect(GABARIT_COMPACT.showScore).toBe(false)
  })

  it('les deux gabarits sont gelés : une cote ne se corrige pas à la volée dans un composant', () => {
    expect(Object.isFrozen(GABARIT_NORMAL)).toBe(true)
    expect(Object.isFrozen(GABARIT_COMPACT)).toBe(true)
  })
})
