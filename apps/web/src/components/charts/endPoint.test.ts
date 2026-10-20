/**
 * endPoint.test.ts — le point final grossi (source unique, revue L6.1 constat R7).
 */
import { describe, expect, it } from 'vitest'

import { END_POINT_SIZE, lastPointIndex, withEndPoint } from './endPoint'

describe('withEndPoint', () => {
  it('par défaut, le dernier point NON nul : grossi et liseré à la couleur de la carte', () => {
    const out = withEndPoint([1, 2, null], 'card')
    expect(out).toEqual([
      1,
      { value: 2, symbol: 'circle', symbolSize: END_POINT_SIZE, itemStyle: { borderColor: 'card', borderWidth: 2 } },
      null,
    ])
    expect(lastPointIndex([null, null])).toBe(-1)
    expect(withEndPoint([null, null], 'card')).toEqual([null, null])
  })

  it('`at` désigne le point ; un trou à cet index : rien n’est grossi, jamais un voisin', () => {
    expect(withEndPoint([1, 2, null], 'card', { at: 2 })).toEqual([1, 2, null])
    expect(withEndPoint([1, 2, 3], 'card', { at: -1 })).toEqual([1, 2, 3])
    expect(withEndPoint([1, 2, 3], 'card', { at: 0 })[0]).toMatchObject({ value: 1, symbolSize: END_POINT_SIZE })
  })

  it('un objet de donnée garde ses champs ; taille, liseré, couleur et champs ajoutés s’appliquent', () => {
    const out = withEndPoint([{ value: 5, tip: 't', itemStyle: { color: 'c', borderWidth: 1 } }], 'card', {
      size: 11,
      borderWidth: 1.5,
      color: 'x',
      extra: (i) => ({ index: i }),
    })
    expect(out[0]).toEqual({
      value: 5,
      tip: 't',
      symbol: 'circle',
      symbolSize: 11,
      itemStyle: { color: 'x', borderColor: 'card', borderWidth: 1.5 },
      index: 0,
    })
  })
})
