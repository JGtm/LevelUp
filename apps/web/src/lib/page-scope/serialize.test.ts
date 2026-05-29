import { describe, it, expect } from 'vitest'

import { csvToSet, setToCsv, strOrUndef } from './serialize'

describe('setToCsv', () => {
  it('joint les valeurs par une virgule', () => {
    expect(setToCsv(new Set(['a', 'b', 'c']))).toBe('a,b,c')
  })

  it('Set vide → undefined (param omis)', () => {
    expect(setToCsv(new Set())).toBeUndefined()
  })

  it('null/undefined → undefined', () => {
    expect(setToCsv(undefined)).toBeUndefined()
    expect(setToCsv(null)).toBeUndefined()
  })

  it('préserve les valeurs avec espaces (vocab Halo)', () => {
    expect(setToCsv(new Set(['Ranked Arena', 'Big Team Battle']))).toBe(
      'Ranked Arena,Big Team Battle',
    )
  })
})

describe('csvToSet', () => {
  it('découpe sur la virgule', () => {
    expect(csvToSet('a,b,c')).toEqual(new Set(['a', 'b', 'c']))
  })

  it('chaîne vide → Set vide', () => {
    expect(csvToSet('')).toEqual(new Set())
  })

  it('valeur non-chaîne → Set vide', () => {
    expect(csvToSet(undefined)).toEqual(new Set())
    expect(csvToSet(42)).toEqual(new Set())
    expect(csvToSet(['a'])).toEqual(new Set())
  })

  it('ignore les segments vides (virgules superflues)', () => {
    expect(csvToSet('a,,b,')).toEqual(new Set(['a', 'b']))
  })
})

describe('round-trip setToCsv ∘ csvToSet', () => {
  it('préserve un Set non vide', () => {
    const original = new Set(['Aquarius', 'Live Fire', 'Streets'])
    expect(csvToSet(setToCsv(original))).toEqual(original)
  })
})

describe('strOrUndef', () => {
  it('chaîne non vide → elle-même', () => {
    expect(strOrUndef('2026-05-01')).toBe('2026-05-01')
  })

  it('vide / null / undefined → undefined', () => {
    expect(strOrUndef('')).toBeUndefined()
    expect(strOrUndef(null)).toBeUndefined()
    expect(strOrUndef(undefined)).toBeUndefined()
  })
})
