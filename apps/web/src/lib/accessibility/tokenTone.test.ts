/**
 * Garde-rail de `tokenTone` : la dérivation reste RELATIVE au jeton (même teinte, clarté
 * décalée vers le premier plan du thème) et ne fabrique jamais une couleur littérale.
 */
import { describe, expect, it } from 'vitest'

import { tokenTone } from './tokenTone'

describe('tokenTone', () => {
  it('rend la couleur inchangée pour un écart nul ou négatif', () => {
    expect(tokenTone('var(--c)', 0)).toBe('var(--c)')
    expect(tokenTone('var(--c)', -0.1)).toBe('var(--c)')
  })

  it('décale la clarté dans les deux thèmes et relève la chroma, teinte intacte', () => {
    expect(tokenTone('var(--c)', 0.12)).toBe(
      'light-dark(oklch(from var(--c) calc(l - 0.12) calc(c * 1.12) h), oklch(from var(--c) calc(l + 0.12) calc(c * 1.12) h))',
    )
  })

  it('plafonne l’écart : au-delà, le ton ne se lirait plus comme le même rôle', () => {
    expect(tokenTone('var(--c)', 0.9)).toBe(tokenTone('var(--c)', 0.3))
  })

  it('n’écrit aucune couleur littérale : la sortie ne contient que le jeton reçu', () => {
    const out = tokenTone('var(--c)', 0.24)
    expect(out).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
    expect(out.match(/var\(--[a-z-]+\)/g)).toEqual(['var(--c)', 'var(--c)'])
  })
})
