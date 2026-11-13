import { describe, expect, it } from 'vitest'

import { meanOfValid, medianOfValid, movingMean } from './stats'

type Maybe = number | null | undefined

/** Ce que les sites écrivaient à la main : la somme dans l'ordre, de zéro, divisée par le nombre. */
const handMean = (xs: number[]): number => xs.reduce((s, v) => s + v, 0) / xs.length

describe('meanOfValid', () => {
  it.each<[string, Maybe[], number | null]>([
    ['vide', [], null],
    ['uniquement des absentes', [null, undefined, NaN, Infinity, -Infinity], null],
    ['une valeur', [4], 4],
    ['des absentes ignorées', [1, null, 3, undefined, NaN], 2],
    ['zéros et négatifs', [0, -2, 2, 0], 0],
  ])('%s', (_label, input, expected) => {
    expect(meanOfValid(input)).toBe(expected)
  })

  it('est, au bit près, la somme dans l’ordre divisée par le nombre sur une série valide', () => {
    for (const xs of [[0.1, 0.2, 0.3], [1e-9, 1e9, -1e9, 0.7], [3.3, 3.3, 3.3, 3.3, 3.3, 3.3, 3.3], [Math.PI, Math.E, Math.SQRT2]]) {
      expect(Object.is(meanOfValid(xs), handMean(xs))).toBe(true)
    }
  })
})

describe('medianOfValid', () => {
  it.each<[string, Maybe[], number | null]>([
    ['vide', [], null],
    ['absentes seulement', [null, NaN], null],
    ['impair', [9, 1, 5], 5],
    ['pair : moyenne des deux centraux', [4, 1, 3, 2], 2.5],
    ['absentes ignorées', [10, null, 2, Infinity, 6, undefined], 6],
    ['une valeur', [7], 7],
    ['doublons', [2, 2, 2, 5], 2],
  ])('%s', (_label, input, expected) => {
    expect(medianOfValid(input)).toBe(expected)
  })

  it('ne modifie pas la série reçue', () => {
    const xs = [3, 1, 2]
    medianOfValid(xs)
    expect(xs).toEqual([3, 1, 2])
  })

  it('médiane à égalité avec la formule des sites : (a + b) / 2 au bit près', () => {
    const xs = [0.1, 0.7, 0.2, 0.3]
    const sorted = [...xs].sort((a, b) => a - b)
    expect(Object.is(medianOfValid(xs), (sorted[1] + sorted[2]) / 2)).toBe(true)
  })
})

describe('movingMean', () => {
  it('fenêtre qui se termine au point, tronquée au début (moyenne mobile expansive)', () => {
    expect(movingMean([2, 4, 6, 8], 3)).toEqual([2, 3, 4, 6])
  })

  it('fenêtre pleine seulement : null sur les window − 1 premiers points', () => {
    expect(movingMean([2, 4, 6, 8], 3, { fullWindowOnly: true })).toEqual([null, null, 4, 6])
  })

  it('les absentes ne comptent ni au numérateur ni au dénominateur ; sans valeur valide : null', () => {
    expect(movingMean([2, null, 6, null, null], 2)).toEqual([2, 2, 6, 6, null])
  })

  it('minValid : une fenêtre trop pauvre en valeurs valides ne rend rien', () => {
    expect(movingMean([1, null, 3, 5, null], 3, { minValid: 2 })).toEqual([null, null, 2, 4, 4])
  })

  it('fenêtre pleine ET minValid égal à la fenêtre : seule une fenêtre entièrement valide rend une moyenne', () => {
    expect(movingMean([1, 2, null, 4, 5, 6], 3, { fullWindowOnly: true, minValid: 3 })).toEqual([null, null, null, null, null, 5])
  })

  it('fenêtre centrée, tronquée aux bords', () => {
    expect(movingMean([1, 2, 3, 4, 5], 3, { align: 'centered' })).toEqual([1.5, 2, 3, 4, 4.5])
  })

  it('centrée, une fenêtre de 1 point rend la série telle quelle (absentes en null)', () => {
    expect(movingMean([1, null, undefined, 4], 1, { align: 'centered' })).toEqual([1, null, null, 4])
  })

  it('série vide : tableau vide', () => {
    expect(movingMean([], 3)).toEqual([])
  })

  it('même somme dans l’ordre que le calcul à la main, au bit près', () => {
    const xs = [0.1, 0.2, 0.3, 0.4, 0.5]
    const out = movingMean(xs, 3)
    expect(Object.is(out[4], handMean([0.3, 0.4, 0.5]))).toBe(true)
    expect(Object.is(out[1], handMean([0.1, 0.2]))).toBe(true)
  })
})
