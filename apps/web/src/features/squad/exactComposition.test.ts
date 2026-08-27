/**
 * Tests — défaut de l'option « composition stricte » (page Escouade).
 *
 * Le défaut a basculé sur « cochée » : la clé absente doit donner `true`, et
 * un choix explicitement stocké (les deux sens) doit rester respecté.
 */
import { describe, it, expect } from 'vitest'
import { exactCompositionDefault } from './exactComposition'

describe('exactCompositionDefault', () => {
  it('aucun choix stocké (clé absente) → cochée par défaut', () => {
    expect(exactCompositionDefault(null)).toBe(true)
  })

  it("choix explicite « true » (cochée) → respecté", () => {
    expect(exactCompositionDefault('true')).toBe(true)
  })

  it("choix explicite « false » (décochée) → respecté, le défaut ne le réécrit pas", () => {
    expect(exactCompositionDefault('false')).toBe(false)
  })

  it('valeur illisible en localStorage → repli sur le défaut coché', () => {
    expect(exactCompositionDefault('')).toBe(true)
    expect(exactCompositionDefault('oui')).toBe(true)
  })
})
