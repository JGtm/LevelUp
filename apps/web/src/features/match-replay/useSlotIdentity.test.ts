/**
 * Tests — distinctSlotColors (le mode « couleurs distinctes par joueur » du calque).
 *
 * Ce qu'ils protègent : la couleur d'un joueur est STABLE — la même pour toutes ses vies,
 * indexée sur son rang dans la jointure — et la palette CYCLE au-delà de sa taille plutôt
 * que de laisser des joueurs sans couleur.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from './rosterLogic'
import { distinctSlotColors } from './useSlotIdentity'

function player(xuid: string, slots: number[]): ReplayPlayer {
  return {
    xuid,
    lives: slots.map((slot) => ({ slot, team: -1, points: [] })) as ReplayPlayer['lives'],
  }
}

describe('distinctSlotColors', () => {
  it('toutes les vies d’un joueur portent SA couleur, indexée sur son rang', () => {
    const table = distinctSlotColors(
      [player('A', [512, 514]), player('B', [513])],
      ['c1', 'c2'],
    )
    expect(table.get(512)).toBe('c1')
    expect(table.get(514)).toBe('c1')
    expect(table.get(513)).toBe('c2')
  })

  it('la palette CYCLE au-delà de sa taille : le troisième joueur reprend la première couleur', () => {
    const table = distinctSlotColors(
      [player('A', [512]), player('B', [513]), player('C', [514])],
      ['c1', 'c2'],
    )
    expect(table.get(514)).toBe('c1')
  })

  it('un slot hors des vies jointes n’a pas de couleur — la convention « ne rien dessiner »', () => {
    const table = distinctSlotColors([player('A', [512])], ['c1'])
    expect(table.get(999)).toBeUndefined()
  })
})
