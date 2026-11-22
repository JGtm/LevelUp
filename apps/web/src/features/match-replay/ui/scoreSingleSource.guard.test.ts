/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : UN SEUL SCORE À L'INSTANT. L'infobulle de la tuile et la fiche du joueur lisent la même valeur,
 * rendue par `scoreOfReading` (`model/playerInstantScore.ts`) sur une lecture rangée une fois par document.
 *
 * POURQUOI. La tuile écrivait le score du calque du film (`live.score`), la fiche la somme des points par action
 * gardée par la feuille : les deux différaient pour plus de la moitié des joueurs mesurés, par un décalage d'une
 * image ou durablement. Le test numérique (`model/playerInstantScore.test.ts`) prouve l'égalité sur un document ;
 * ce garde empêche qu'un des deux lecteurs reprenne une autre source ou une autre fonction.
 */
import { describe, expect, it } from 'vitest'

import { fichierNomme, lire } from '../test/featureFiles'

describe('garde-rail : un seul score à l’instant', () => {
  it('le triplet de la tuile ne lit pas le score du calque de compteurs', () => {
    expect(lire(fichierNomme('ReplayCountersBadge.tsx'))).not.toMatch(/\blive\.score\b/)
  })

  it('la tuile lit sa valeur par `playerScoreAt`, qui passe par `scoreOfReading`', () => {
    expect(lire(fichierNomme('ReplayPlayerCard.tsx'))).toMatch(/\bplayerScoreAt\(/)
    const foyer = lire(fichierNomme('playerInstantScore.ts'))
    expect(foyer).toMatch(/export function playerScoreAt[\s\S]*?return scoreOfReading\(/)
  })

  it('la fiche lit la même fonction, sur les lectures de score du même foyer', () => {
    const fiche = lire(fichierNomme('playerInstantEstimates.ts'))
    expect(fiche).toMatch(/\bscoreOfReading\(/)
    expect(fiche).toMatch(/\bscoreReaderOf\(/)
  })
})
