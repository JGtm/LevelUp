/**
 * bonusColor.test.ts — non-régression : la couleur du segment "Bonus"
 * (assistances) doit rester distincte des 8 couleurs verrouillées de
 * l'escouade — les 4 couleurs joueurs ET leurs opposés colorimétriques
 * (hexComplement, hue +180°, utilisés pour les barres "morts").
 *
 * Historique : le bonus utilisait `chart-series-7` (#F59E0B ambre), identique
 * à `perf-tier-3` (#F59E0B), la couleur du 3ᵉ joueur → indiscernable.
 *
 * NB : fichier séparé de colors.test.ts qui mocke tout `@/lib/accessibility`.
 * Ici on importe les vraies fonctions (hexComplement, palette) — tests purs.
 */
import { describe, it, expect } from 'vitest'

import { hexComplement } from '@/lib/accessibility'
import { defaultPalette } from '@/lib/accessibility/palettes/default'
import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from './colors'

const PLAYER_TOKENS = [SQUAD_MAIN_PLAYER_TOKEN, ...SQUAD_TEAMMATE_COLOR_TOKENS]

describe('couleur Bonus (palette default)', () => {
  it('est distincte des 4 couleurs joueurs et de leurs opposés colorimétriques', () => {
    const playerHexes = PLAYER_TOKENS.map((t) => defaultPalette[t].toUpperCase())
    const forbidden = new Set<string>([
      ...playerHexes,
      ...playerHexes.map((h) => hexComplement(h).toUpperCase()),
    ])

    expect(forbidden.size).toBe(8) // 4 joueurs + 4 opposés, tous distincts
    expect(forbidden.has(defaultPalette.bonus.toUpperCase())).toBe(false)
  })

  it("n'utilise plus chart-series-7 (source de la collision ambre)", () => {
    expect(defaultPalette.bonus.toUpperCase()).not.toBe(
      defaultPalette['chart-series-7'].toUpperCase(),
    )
  })
})
