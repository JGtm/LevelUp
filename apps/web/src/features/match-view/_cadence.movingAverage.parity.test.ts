/**
 * movingAverage (Vue match, cadence) avant et après son passage sur `movingMean` (étape 7.8) : l'ancien
 * calcul, arrondi au dixième compris, est gardé ici comme référence, mêmes nombres au bit près.
 */
import { describe, expect, it } from 'vitest'

import { CADENCE_MA_WINDOW, movingAverage } from './_cadence'

function ancienne(values: readonly number[], window: number): number[] {
  return values.map((_, i) => {
    const start = Math.max(0, i - (window - 1))
    let sum = 0
    for (let j = start; j <= i; j++) sum += values[j]
    return Math.round((sum / (i - start + 1)) * 10) / 10
  })
}

const CAS: Array<[string, number[], number]> = [
  ['série vide', [], CADENCE_MA_WINDOW],
  ['un point', [4], CADENCE_MA_WINDOW],
  ['comptes entiers', [0, 3, 5, 2, 8, 1, 0, 4], CADENCE_MA_WINDOW],
  ['arrondis au dixième à la limite', [1, 2, 2, 0, 0, 1], CADENCE_MA_WINDOW],
  ['fenêtre plus large que la série', [2, 4], 10],
  ['fenêtre de 1', [2, 4, 9], 1],
]

describe('movingAverage (cadence) : mêmes nombres qu’avant 7.8', () => {
  it.each(CAS)('%s', (_label, values, window) => {
    const attendu = ancienne(values, window)
    const obtenu = movingAverage(values, window)
    expect(obtenu).toHaveLength(attendu.length)
    obtenu.forEach((v, i) => expect(Object.is(v, attendu[i])).toBe(true))
  })
})
