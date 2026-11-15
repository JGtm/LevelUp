/**
 * Tests — le texte des valeurs du panneau de joueur : entier, pourcentage, et la lacune qui n'est
 * jamais un zéro.
 */
import { describe, expect, it } from 'vitest'

import { formatPanelValue, NO_VALUE } from './playerPanelFormat'

describe('formatPanelValue', () => {
  it('une valeur non lue est un tiret, pas un zéro', () => {
    expect(formatPanelValue({ format: 'integer', value: null }, 'fr')).toBe(NO_VALUE)
    expect(formatPanelValue({ format: 'percent', value: null }, 'en')).toBe(NO_VALUE)
  })

  it('un zéro MESURÉ s’écrit 0', () => {
    expect(formatPanelValue({ format: 'integer', value: 0 }, 'fr')).toBe('0')
  })

  it('un entier suit la langue (séparateur de milliers)', () => {
    expect(formatPanelValue({ format: 'integer', value: 3457 }, 'en')).toBe('3,457')
    expect(formatPanelValue({ format: 'integer', value: 3457 }, 'fr').replace(/\s/g, ' ')).toBe('3 457')
  })

  it('un pourcentage a une décimale et une espace insécable avant le signe', () => {
    expect(formatPanelValue({ format: 'percent', value: 42.5 }, 'en')).toBe('42.5\u00a0%')
    expect(formatPanelValue({ format: 'percent', value: 42.5 }, 'fr')).toBe('42,5\u00a0%')
  })
})
