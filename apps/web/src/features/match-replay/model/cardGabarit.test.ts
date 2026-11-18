/**
 * Tests — cardGabarit : les cotes de la tuile compacte, valeur pour valeur.
 *
 * Si une cote change, la fixation DOM de la tuile casse — ce test dit LAQUELLE, sans lire le HTML. L'encadré
 * « Éliminé » du bloc « À l'instant » a sa hauteur dans `COTES_BLOC_INSTANT`.
 */
import { describe, expect, it } from 'vitest'

import { COTES_BLOC_INSTANT, GABARIT_COMPACT } from './cardGabarit'

describe('cardGabarit', () => {
  it('GABARIT_COMPACT = la tuile A2 : une cellule d’arme de 48, corps de 31, deux éclairs, filigrane 34', () => {
    expect(GABARIT_COMPACT.weaponCellW).toBe(48)
    expect(GABARIT_COMPACT.bodyPx).toBe(31)
    expect(GABARIT_COMPACT.gaugeShieldPx + GABARIT_COMPACT.gaugeHealthPx).toBe(6)
    expect(GABARIT_COMPACT.countCellW).toBe(10)
    expect(GABARIT_COMPACT.boltCount).toBe(2)
    expect(GABARIT_COMPACT.watermarkPx).toBe(34)
  })

  it('l’encadré « Éliminé » du bloc « À l’instant » tient sur un corps de 35, cote propre au bloc', () => {
    expect(COTES_BLOC_INSTANT.corpsEliminePx).toBe(35)
  })

  it('le gabarit et les cotes du bloc sont gelés : une cote ne se corrige pas à la volée dans un composant', () => {
    expect(Object.isFrozen(GABARIT_COMPACT)).toBe(true)
    expect(Object.isFrozen(COTES_BLOC_INSTANT)).toBe(true)
  })
})
