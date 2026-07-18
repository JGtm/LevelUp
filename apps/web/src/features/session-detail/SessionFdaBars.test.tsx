/**
 * Tests buildSessionFdaBarsOption (3 barres FDA) + helpers de delta de rang.
 */
import { describe, expect, it } from 'vitest'

import { buildSessionFdaBarsOption } from './SessionFdaBars'
import { formatRankDelta, rankDeltaToken } from './_shared'

interface OptShape {
  series?: Array<{ type: string; data: Array<{ value: number; itemStyle: { color: string } }> }>
  xAxis?: { type: string; data: string[] }
}

function fdaSeries(frags: number, deaths: number, assists: number) {
  return [
    {
      key: 'fda',
      datapoints: [
        { key: 'frags' as const, label: 'Frags', value: frags },
        { key: 'deaths' as const, label: 'Morts', value: deaths },
        { key: 'assists' as const, label: 'Assists', value: assists },
      ],
    },
  ]
}

describe('buildSessionFdaBarsOption', () => {
  it('produit 3 barres FDA, Morts en NÉGATIF (vers le bas, façon Escouade)', () => {
    const opt = buildSessionFdaBarsOption(fdaSeries(12.34, 5.67, 3.21), { decimals: 1 }) as unknown as OptShape
    const bar = opt.series![0]
    expect(bar.type).toBe('bar')
    // Frags/Assists positifs (haut), Morts négatives (bas) depuis l'axe zéro.
    expect(bar.data.map((d) => d.value)).toEqual([12.3, -5.7, 3.2])
    expect(bar.data.every((d) => 'color' in d.itemStyle)).toBe(true)
    expect(opt.xAxis!.type).toBe('category')
    expect(opt.xAxis!.data).toEqual(['Frags', 'Morts', 'Assists'])
  })

  it('arrondit à 2 décimales en mode minute (Morts négatives)', () => {
    const opt = buildSessionFdaBarsOption(fdaSeries(0.853, 0.41, 0.2), { decimals: 2 }) as unknown as OptShape
    expect(opt.series![0].data.map((d) => d.value)).toEqual([0.85, -0.41, 0.2])
  })

  it('retourne une option vide sans points', () => {
    const opt = buildSessionFdaBarsOption([], { decimals: 1 }) as unknown as OptShape
    expect(opt.series).toBeUndefined()
  })
})

describe('formatRankDelta / rankDeltaToken', () => {
  it('CSR = entier signé', () => {
    expect(formatRankDelta(45, 'csr')).toBe('+45')
    expect(formatRankDelta(-12, 'csr')).toBe('−12')
    expect(formatRankDelta(0, 'csr')).toBe('±0')
  })

  it('LUSR = 2 décimales signées (glyphe − U+2212)', () => {
    expect(formatRankDelta(1.234, 'lusr')).toBe('+1.23')
    expect(formatRankDelta(-0.5, 'lusr')).toBe('−0.50')
  })

  it('token de couleur par signe', () => {
    expect(rankDeltaToken(5)).toBe('divergent-pos')
    expect(rankDeltaToken(-5)).toBe('divergent-neg')
    expect(rankDeltaToken(0)).toBe('divergent-neutral')
  })
})
