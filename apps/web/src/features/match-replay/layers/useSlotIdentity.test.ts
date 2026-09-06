/**
 * Tests — distinctColorResolver (le mode « couleurs distinctes par joueur » du calque).
 *
 * Ce qu'ils protègent : la couleur d'un joueur est indexée sur son rang dans la jointure, la
 * palette CYCLE au-delà de sa taille, ET la couleur est résolue PAR IMAGE — un slot réattribué
 * entre manches suit son propriétaire courant, comme les couleurs d'équipe.
 */
import { describe, expect, it } from 'vitest'

import { buildSlotOwnership, type ReplayPlayer } from '../model/rosterLogic'
import { distinctColorResolver } from './useSlotIdentity'

/** Un joueur avec des vies sur des slots (fenêtre par défaut [start, end], une par slot). */
function player(xuid: string, lives: [slot: number, start: number, end: number][]): ReplayPlayer {
  return {
    xuid,
    lives: lives.map(([slot, startFrame, endFrame]) => ({
      slot, team: -1, startFrame, endFrame, points: [{ t: startFrame, x: 0, y: 0 }],
    })) as ReplayPlayer['lives'],
  }
}

describe('distinctColorResolver', () => {
  it('toutes les vies d’un joueur portent SA couleur, indexée sur son rang', () => {
    const players = [player('A', [[512, 0, 100], [514, 0, 100]]), player('B', [[513, 0, 100]])]
    const color = distinctColorResolver(buildSlotOwnership(players), players, ['c1', 'c2'])
    expect(color(512, 10)).toBe('c1')
    expect(color(514, 10)).toBe('c1')
    expect(color(513, 10)).toBe('c2')
  })

  it('la palette CYCLE au-delà de sa taille : le troisième joueur reprend la première couleur', () => {
    const players = [
      player('A', [[512, 0, 100]]),
      player('B', [[513, 0, 100]]),
      player('C', [[514, 0, 100]]),
    ]
    const color = distinctColorResolver(buildSlotOwnership(players), players, ['c1', 'c2'])
    expect(color(514, 10)).toBe('c1')
  })

  it('un slot sans propriétaire à cette image n’a pas de couleur — la convention « ne rien dessiner »', () => {
    const players = [player('A', [[512, 0, 100]])]
    const color = distinctColorResolver(buildSlotOwnership(players), players, ['c1'])
    expect(color(999, 10)).toBeNull() // slot inconnu
    expect(color(512, 500)).toBeNull() // slot connu mais hors de toute vie à cette image
  })

  it('multi-manche : la couleur suit le propriétaire de LA MANCHE COURANTE', () => {
    // Slot 512 : joueur A en manche 0, joueur B en manche 2. Une couleur figée par slot aurait
    // montré un seul joueur ; résolue par image, elle suit l’occupant.
    const players = [player('A', [[512, 0, 50]]), player('B', [[512, 200, 250]])]
    const color = distinctColorResolver(buildSlotOwnership(players), players, ['c1', 'c2'])
    expect(color(512, 25)).toBe('c1') // A (rang 0)
    expect(color(512, 220)).toBe('c2') // B (rang 1)
  })
})
