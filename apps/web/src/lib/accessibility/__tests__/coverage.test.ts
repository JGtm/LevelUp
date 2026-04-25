/**
 * coverage.test.ts — Vérifie que chaque palette couvre l'intégralité des tokens.
 *
 * Ce test est un snapshot de sécurité : toute modification d'un token
 * ou d'une palette le fait échouer, forçant une revue consciente.
 */
import { describe, it, expect } from 'vitest'
import { ALL_TOKENS } from '../semantic-tokens'
import { defaultPalette } from '../palettes/default'
import { okabePalette } from '../palettes/okabe-ito'

const PALETTES = {
  default: defaultPalette,
  'okabe-ito': okabePalette,
}

const HEX_RE = /^#[0-9a-fA-F]{3,8}$/

describe.each(Object.entries(PALETTES))('palette "%s"', (_name, palette) => {
  it('définit tous les SemanticToken', () => {
    for (const token of ALL_TOKENS) {
      expect(palette[token], `token manquant : ${token}`).toBeDefined()
      expect(palette[token], `token vide : ${token}`).not.toBe('')
    }
  })

  it('ne contient aucune valeur hors de ALL_TOKENS', () => {
    for (const key of Object.keys(palette)) {
      expect(ALL_TOKENS, `token inconnu dans la palette : ${key}`).toContain(key)
    }
  })

  it('toutes les valeurs sont des hex valides', () => {
    for (const token of ALL_TOKENS) {
      const value = palette[token]
      expect(HEX_RE.test(value), `hex invalide pour ${token} : "${value}"`).toBe(true)
    }
  })

  it('snapshot stable — toute modification doit être intentionnelle', () => {
    expect(palette).toMatchSnapshot()
  })
})
