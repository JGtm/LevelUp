/**
 * cumulativeSeries.test.ts — cumul signé générique avec report D5 + moyenne.
 *
 * - `cumulativeSigned` : cumul par ordre donné, report carry-forward (null ne fait
 *   pas avancer le cumul, la courbe reporte), arrondi 2 déc.
 * - `meanOfValid` : moyenne sur les contributions valides uniquement.
 * - `finiteOrNull` : garde D5 partagée.
 */
import { describe, it, expect } from 'vitest'

import { cumulativeSigned, meanOfValid, finiteOrNull } from './cumulativeSeries'

describe('finiteOrNull', () => {
  it('renvoie la valeur si finie, sinon null', () => {
    expect(finiteOrNull(1.5)).toBe(1.5)
    expect(finiteOrNull(0)).toBe(0)
    expect(finiteOrNull(null)).toBeNull()
    expect(finiteOrNull(undefined)).toBeNull()
    expect(finiteOrNull(Infinity)).toBeNull()
    expect(finiteOrNull(NaN)).toBeNull()
  })
})

describe('cumulativeSigned', () => {
  it('cumul signé dans l\'ordre donné', () => {
    const out = cumulativeSigned([0.5, -0.4, 1.0])
    expect(out.map((p) => p.cumulative)).toEqual([0.5, 0.1, 1.1])
    expect(out.map((p) => p.value)).toEqual([0.5, -0.4, 1])
  })

  it('report D5 : une contribution null ne fait pas avancer le cumul', () => {
    const out = cumulativeSigned([0.5, null, 1.0])
    expect(out.map((p) => p.value)).toEqual([0.5, null, 1])
    // Le point du milieu reporte 0.5, puis reprend (+1.0 = 1.5).
    expect(out.map((p) => p.cumulative)).toEqual([0.5, 0.5, 1.5])
  })

  it('valeur non-finie traitée comme absente (report)', () => {
    const out = cumulativeSigned([0.5, Infinity, 1.0])
    expect(out[1].value).toBeNull()
    expect(out.map((p) => p.cumulative)).toEqual([0.5, 0.5, 1.5])
  })

  it('arrondi à 2 décimales (cumul et valeur)', () => {
    const out = cumulativeSigned([0.111, 0.222])
    expect(out.map((p) => p.value)).toEqual([0.11, 0.22])
    expect(out.map((p) => p.cumulative)).toEqual([0.11, 0.33])
  })

  it('tableau vide → aucun point', () => {
    expect(cumulativeSigned([])).toEqual([])
  })
})

describe('meanOfValid', () => {
  it('moyenne sur les contributions valides', () => {
    expect(meanOfValid([0.6, 0.4])).toBe(0.5)
  })

  it('ignore null / non-fini (D5)', () => {
    expect(meanOfValid([0.6, null, undefined, Infinity])).toBe(0.6)
  })

  it('null si aucune contribution exploitable', () => {
    expect(meanOfValid([null, undefined])).toBeNull()
    expect(meanOfValid([])).toBeNull()
  })
})
