/**
 * playerMatchTotals.test.ts — LES TOTAUX DE FIN DE MATCH LUS SUR LA FEUILLE, sans rien inventer : une valeur
 * que la feuille ne porte pas n'a pas de ligne, un zéro porté par la feuille reste un zéro.
 */
import { describe, expect, it } from 'vitest'

import { scoreboardRow } from '../test/scoreboardRow'
import { playerMatchTotals } from './playerMatchTotals'

const NO_TOTALS = {
  accuracy: undefined, shots_hit: undefined, shots_fired: undefined,
  damage_dealt: undefined, damage_taken: undefined, score: undefined, headshot_kills: undefined,
}

describe('playerMatchTotals', () => {
  it('une feuille complète : précision avec touches et tirs, dégâts, score, éliminations à la tête', () => {
    const board = scoreboardRow('A', 'Alpha', 'a', {
      accuracy: 41.2, shots_hit: 118, shots_fired: 288,
      damage_dealt: 2140.4, damage_taken: 1610, score: 1450, headshot_kills: 6,
    })
    expect(playerMatchTotals(board)).toEqual({
      accuracy: { pct: 41.2, hit: 118, fired: 288 },
      damage: { dealt: 2140.4, taken: 1610 },
      score: 1450,
      headshotKills: 6,
    })
  })

  it('sans ligne de feuille : aucun total', () => {
    expect(playerMatchTotals(undefined)).toBeNull()
  })

  it('une feuille qui ne porte rien de tout cela : aucun total', () => {
    expect(playerMatchTotals(scoreboardRow('A', 'Alpha', 'a', NO_TOTALS))).toBeNull()
  })

  it('un zéro porté par la feuille est un zéro : la ligne existe', () => {
    const t = playerMatchTotals(scoreboardRow('A', 'Alpha', 'a', { ...NO_TOTALS, score: 0, headshot_kills: 0 }))
    expect(t).toEqual({ accuracy: null, damage: null, score: 0, headshotKills: 0 })
  })

  it('la précision sans touches ni tirs : le pour-cent seul', () => {
    const t = playerMatchTotals(scoreboardRow('A', 'Alpha', 'a', { ...NO_TOTALS, accuracy: 40, shots_fired: 100 }))
    expect(t?.accuracy).toEqual({ pct: 40, hit: null, fired: null })
  })

  it('les dégâts n’ont de ligne que si la feuille porte les deux', () => {
    const seul = playerMatchTotals(scoreboardRow('A', 'Alpha', 'a', { ...NO_TOTALS, score: 10, damage_dealt: 500 }))
    expect(seul?.damage).toBeNull()
  })

  it('une valeur non finie n’est pas lue', () => {
    const t = playerMatchTotals(scoreboardRow('A', 'Alpha', 'a', { ...NO_TOTALS, score: Number.NaN, headshot_kills: 2 }))
    expect(t).toEqual({ accuracy: null, damage: null, score: null, headshotKills: 2 })
  })
})
