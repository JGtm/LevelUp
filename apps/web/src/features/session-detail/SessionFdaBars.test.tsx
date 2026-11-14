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

function damageSeries() {
  return [
    {
      key: 'fda',
      datapoints: [
        { key: 'frags' as const, label: 'Frags', value: 1.2 },
        { key: 'deaths' as const, label: 'Morts', value: 0.7 },
        { key: 'assists' as const, label: 'Assists', value: 0.3 },
        { key: 'damageDealt' as const, label: 'Infligés', value: 312.6 },
        { key: 'damageTaken' as const, label: 'Subis', value: 240.4 },
      ],
    },
  ]
}

interface DamageOpt {
  series: Array<{ yAxisIndex: number; barGap: string; data: Array<{ value: number } | null> }>
  yAxis: Array<{ min?: number; max?: number; position?: string; splitLine?: { show: boolean } }>
  xAxis: { data: string[] }
}

describe('buildSessionFdaBarsOption — dégâts par minute', () => {
  it('deux séries superposées : comptes sur l axe de gauche, dégâts sur l axe de droite', () => {
    const opt = buildSessionFdaBarsOption(damageSeries(), { decimals: 2 }) as unknown as DamageOpt
    expect(opt.xAxis.data).toEqual(['Frags', 'Morts', 'Assists', 'Infligés', 'Subis'])
    expect(opt.series).toHaveLength(2)
    const [left, right] = opt.series
    expect(left.yAxisIndex).toBe(0)
    expect(right.yAxisIndex).toBe(1)
    // Chaque colonne n est portée que par UNE série ; subis et morts sous la ligne zéro.
    expect(left.data.map((d) => d?.value ?? null)).toEqual([1.2, -0.7, 0.3, null, null])
    expect(right.data.map((d) => d?.value ?? null)).toEqual([null, null, null, 313, -240])
    // Les deux séries occupent la même colonne.
    expect(left.barGap).toBe('-100%')
    expect(right.barGap).toBe('-100%')
  })

  it('axe de droite sans grille, zéros alignés sur celui de gauche', () => {
    const opt = buildSessionFdaBarsOption(damageSeries(), { decimals: 2 }) as unknown as DamageOpt
    expect(opt.yAxis).toHaveLength(2)
    const [l, r] = opt.yAxis
    expect(r.position).toBe('right')
    expect(r.splitLine).toEqual({ show: false })
    const share = (a: { min?: number; max?: number }) => -a.min! / (a.max! - a.min!)
    expect(share(l)).toBeCloseTo(share(r), 10)
    expect(r.max!).toBeGreaterThanOrEqual(312.6)
    expect(r.min!).toBeLessThanOrEqual(-240.4)
  })

  it('domaines partagés de la comparaison : repris tels quels', () => {
    const opt = buildSessionFdaBarsOption(damageSeries(), {
      decimals: 2,
      yDomain: [-2, 4],
      yDomainRight: [-500, 1000],
    }) as unknown as DamageOpt
    expect([opt.yAxis[0].min, opt.yAxis[0].max]).toEqual([-2, 4])
    expect([opt.yAxis[1].min, opt.yAxis[1].max]).toEqual([-500, 1000])
  })

  it('sans dégâts : une seule série, un seul axe', () => {
    const opt = buildSessionFdaBarsOption(fdaSeries(1, 1, 1), { decimals: 2 }) as unknown as DamageOpt
    expect(opt.series).toHaveLength(1)
    expect(opt.yAxis).toHaveLength(1)
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
