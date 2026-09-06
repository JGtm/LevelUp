/**
 * exportOverlayPanels.test.ts — LE PANNEAU DE FIN REPEINT DANS LA VIDÉO dit-il la même
 * chose que celui affiché à l'écran ?
 *
 * Les deux surfaces annoncent le MÊME match : si elles tranchaient séparément, la vidéo
 * exportée afficherait les points de la dernière manche (« 100 - 43 ») pendant que la page
 * dit « 2 - 1 ». Le jumeau DOM est couvert par ReplayVictoryOverlay.test.tsx ; ce fichier
 * couvre le choix côté export, qui ne l'était pas (constat de revue adversariale du
 * 2026-08-29).
 */
import { describe, expect, it } from 'vitest'

import { exportFinalScore } from './exportOverlayPanels'

const film = { ally: { score: 100 }, enemy: { score: 43 } }

describe('exportFinalScore', () => {
  it("préfère le score servi par l'API quand il existe", () => {
    expect(exportFinalScore({ ally: 2, enemy: 1 }, film)).toEqual({ ally: 2, enemy: 1 })
  })

  it('retombe sur la lecture du calque du film quand l’API ne dit rien', () => {
    expect(exportFinalScore(null, film)).toEqual({ ally: 100, enemy: 43 })
    expect(exportFinalScore(undefined, film)).toEqual({ ally: 100, enemy: 43 })
  })

  it("n'invente rien quand aucune des deux sources ne parle", () => {
    expect(exportFinalScore(null, null)).toBeNull()
  })

  it('accepte un zéro : 2 manches à 0 est une mesure, pas une absence', () => {
    expect(exportFinalScore({ ally: 2, enemy: 0 }, film)).toEqual({ ally: 2, enemy: 0 })
  })
})
