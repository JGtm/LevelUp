import { describe, expect, it } from 'vitest'

import { withoutNullRows } from './withoutNullRows'

describe('withoutNullRows', () => {
  it.each([
    ['sans enregistrement', undefined, {}],
    ['enregistrement nul', null, {}],
    ['enregistrement vide', {}, {}],
    ['une entrée nulle est retirée', { a: [1, 2], b: null }, { a: [1, 2] }],
    ['une liste vide est gardée', { a: [] as number[], b: [3] }, { a: [], b: [3] }],
  ])('%s', (_label, input, expected) => {
    expect(withoutNullRows<number>(input)).toEqual(expected)
  })

  it('ne partage pas les listes avec l’entrée', () => {
    const rows = [1]
    expect(withoutNullRows({ a: rows }).a).not.toBe(rows)
  })
})
