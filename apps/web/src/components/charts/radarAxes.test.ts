import { describe, it, expect } from 'vitest'

import { axesByKeys, commonAxisKeys } from './radarAxes'

const ax = (...keys: string[]) => keys.map((axis) => ({ axis, value: axis.length, raw: 0 }))

describe('commonAxisKeys', () => {
  it('garde les axes de toutes les séries, dans l\'ordre de la première', () => {
    expect(commonAxisKeys([ax('a', 'b', 'c'), ax('c', 'a', 'b')])).toEqual(['a', 'b', 'c'])
  })

  it('retire un axe absent d\'une seule série, même si la première le porte', () => {
    expect(commonAxisKeys([ax('a', 'objective', 'c'), ax('a', 'c')])).toEqual(['a', 'c'])
    expect(commonAxisKeys([ax('a', 'c'), ax('a', 'objective', 'c')])).toEqual(['a', 'c'])
  })

  it('aucune série, série sans axes ou axes null : aucun indicateur', () => {
    expect(commonAxisKeys([])).toEqual([])
    expect(commonAxisKeys([ax('a'), null])).toEqual([])
    expect(commonAxisKeys([null])).toEqual([])
  })
})

describe('axesByKeys', () => {
  it('lit les axes par clé, dans l\'ordre demandé', () => {
    expect(axesByKeys(ax('a', 'bb', 'ccc'), ['ccc', 'a']).map((a) => a.axis)).toEqual(['ccc', 'a'])
  })

  it('une clé absente de la série est omise, jamais remplacée par un zéro', () => {
    expect(axesByKeys(ax('a'), ['a', 'x'])).toHaveLength(1)
    expect(axesByKeys(null, ['a'])).toEqual([])
  })
})
