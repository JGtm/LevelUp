/**
 * playerShield.test.ts — LES BRIS ET LES RECHARGES QUE LE SERVEUR DATE, COMPTÉS JUSQU'À L'INSTANT.
 *
 * Le web ne redéfinit rien : il choisit la série d'un joueur par ses clés d'identité et compte les instants
 * (ms de l'axe du document) inférieurs ou égaux à la tête de lecture. Une absence est un « non mesuré ».
 */
import { describe, expect, it } from 'vitest'

import { readShieldSeries, shieldReadingAt } from './playerShield'

const DOC = {
  shieldCycles: {
    A: { breaks: [1_000, 4_000, 4_000, 9_000], recharges: [2_500, 9_000] },
    'bot:Marc': { breaks: [], recharges: [] },
    // Un tableau nul (liste Go sans élément) se lit comme une liste vide.
    N: { breaks: null, recharges: null },
  },
} as never

describe('readShieldSeries', () => {
  it('prend la première clé d’identité connue du calque', () => {
    expect(readShieldSeries(DOC, ['bid(7.0)', 'A'])?.breaks).toHaveLength(4)
    expect(readShieldSeries(DOC, ['bot:Marc'])).toEqual({ breaks: [], recharges: [] })
  })

  it('un joueur absent du calque, ou un document sans calque, n’a pas de série', () => {
    expect(readShieldSeries(DOC, ['Z'])).toBeNull()
    expect(readShieldSeries(DOC, [])).toBeNull()
    expect(readShieldSeries({ shieldCycles: undefined } as never, ['A'])).toBeNull()
  })

  it('une liste nulle est une liste vide : le joueur est mesuré, à zéro', () => {
    expect(readShieldSeries(DOC, ['N'])).toEqual({ breaks: [], recharges: [] })
  })
})

describe('shieldReadingAt', () => {
  const series = readShieldSeries(DOC, ['A'])!

  it('avant le premier événement : zéro partout, les totaux du match déjà connus', () => {
    expect(shieldReadingAt(series, 999)).toEqual({
      breaks: 0, recharges: 0, differential: 0, matchBreaks: 4, matchRecharges: 2, matchDifferential: -2,
    })
  })

  it('compte les événements datés au plus tard à l’instant, instant compris, doublons compris', () => {
    expect(shieldReadingAt(series, 1_000)).toMatchObject({ breaks: 1, recharges: 0, differential: -1 })
    expect(shieldReadingAt(series, 4_000)).toMatchObject({ breaks: 3, recharges: 1, differential: -2 })
    expect(shieldReadingAt(series, 9_000)).toMatchObject({ breaks: 4, recharges: 2, differential: -2 })
  })

  it('après le dernier événement, la valeur à l’instant égale le total du match', () => {
    const fin = shieldReadingAt(series, 10 ** 9)
    expect([fin.breaks, fin.recharges, fin.differential]).toEqual([fin.matchBreaks, fin.matchRecharges, fin.matchDifferential])
  })
})
