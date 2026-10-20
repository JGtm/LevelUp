/**
 * Tests du builder des courbes d'« Évolution » : axes, repères, légende, courbe en pointillé,
 * couleurs issues des jetons, fenêtre de l'axe X, infobulle.
 */
import { describe, expect, it, vi } from 'vitest'

import type { TrendsPoint } from '@/lib/api/types'

import {
  AXIS_INK,
  buildTendancesLinesOption,
  type TendancesLinesInput,
} from './tendancesLines.logic'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

const FROM = Date.parse('2026-07-07T12:00:00Z')
const TO = Date.parse('2026-10-05T12:00:00Z')

function points(values: number[]): TrendsPoint[] {
  return values.map((value, i) => ({
    t: new Date(TO - (values.length - i) * 86_400_000).toISOString(),
    value,
    matches: i + 1,
  }))
}

function input(overrides: Partial<TendancesLinesInput> = {}): TendancesLinesInput {
  return {
    curves: [{ name: 'FDA', color: 'chart-series-1', points: points([1, 2, 3]) }],
    from: FROM,
    to: TO,
    unit: 'number',
    decimals: 2,
    step: 'day',
    locale: 'fr',
    timeZone: 'UTC',
    labels: {
      tooltipLine: (name, value, matches) => `${name} = ${value} / ${matches}`,
      weekOf: (date) => `sem. ${date}`,
    },
    ...overrides,
  }
}

/* eslint-disable @typescript-eslint/no-explicit-any -- lecture d'une option ECharts non typée */
type Opt = {
  series: Array<Record<string, any>>
  yAxis: Array<Record<string, any>>
  xAxis: Record<string, any>
  legend?: Record<string, any>
  grid: Record<string, any>
  tooltip: { formatter: (p: unknown) => string } & Record<string, any>
}
/* eslint-enable @typescript-eslint/no-explicit-any */
const build = (i: TendancesLinesInput) => buildTendancesLinesOption(i) as unknown as Opt

const DEUX_COURBES: TendancesLinesInput['curves'] = [
  { name: 'A', color: 'chart-series-1', points: points([1, 2]) },
  { name: 'B', color: 'chart-series-2', points: points([1, 2]) },
]

describe('buildTendancesLinesOption — axes', () => {
  it('sans courbe de droite : un seul axe Y, marge droite normale', () => {
    const o = build(input())
    expect(o.yAxis).toHaveLength(1)
    expect(o.yAxis[0].scale).toBe(true)
    expect(o.grid.right).toBe(16)
    expect(o.series).toHaveLength(1)
  })

  it('avec le MMR adverse : second axe sans grille, courbe pointillée en losange, couleur 4', () => {
    const o = build(input({ right: { name: 'MMR adverse', points: points([1500, 1510, 1520]) } }))
    expect(o.yAxis).toHaveLength(2)
    expect(o.yAxis[1].splitLine).toEqual({ show: false })
    expect(o.grid.right).toBeGreaterThan(16)
    const droite = o.series[1]
    expect(droite.yAxisIndex).toBe(1)
    expect(droite.symbol).toBe('diamond')
    expect(droite.lineStyle.type).toBe('dotted')
    expect(droite.lineStyle.color).toBe('color:chart-series-5')
  })

  it('axe X de type temps, borné à la fenêtre', () => {
    const o = build(input())
    expect(o.xAxis.type).toBe('time')
    expect(o.xAxis.min).toBe(FROM)
    expect(o.xAxis.max).toBe(TO)
  })

  it('étiquettes de l’axe Y formatées par unité (ratio en pourcentage)', () => {
    const o = build(input({ unit: 'ratio', decimals: 3 }))
    expect(o.yAxis[0].axisLabel.formatter(0.5)).toBe('50,0 %')
  })

  it('sans courbe : option vide', () => {
    expect(buildTendancesLinesOption(input({ curves: [] }))).toEqual({ backgroundColor: 'transparent' })
  })
})

describe('buildTendancesLinesOption — courbes', () => {
  it('trait plein, points ronds de taille 6, sans lissage', () => {
    const s = build(input()).series[0]
    expect(s.type).toBe('line')
    expect(s.smooth).toBe(false)
    expect(s.symbol).toBe('circle')
    expect(s.symbolSize).toBe(6)
    expect(s.lineStyle.type).toBeUndefined()
    expect(s.data[0].value).toHaveLength(2)
  })

  it('courbe en pointillé : tireté, sans symbole, pastille de légende élargie', () => {
    const o = build(
      input({
        curves: [
          DEUX_COURBES[0],
          { name: 'Parité', color: AXIS_INK, points: points([1, 2]), dashed: true },
        ],
      }),
    )
    expect(o.series[1].lineStyle.type).toBe('dashed')
    expect(o.series[1].symbol).toBe('none')
    expect(o.legend?.itemWidth).toBe(30)
  })

  it('couleurs issues des jetons, résolues dans le builder', () => {
    const o = build(
      input({
        curves: [
          { name: 'Frags', color: 'stat-kills', points: points([1, 2]) },
          { name: 'Morts', color: 'stat-deaths', points: points([1, 2]) },
        ],
      }),
    )
    expect(o.series[0].lineStyle.color).toBe('color:stat-kills')
    expect(o.series[1].itemStyle.color).toBe('color:stat-deaths')
  })
})

describe('buildTendancesLinesOption — repères', () => {
  it('repère horizontal en pointillé sur la première courbe seulement', () => {
    const o = build(input({ reference: 0, curves: DEUX_COURBES }))
    expect(o.series[0].markLine.data).toEqual([expect.objectContaining({ yAxis: 0 })])
    expect(o.series[0].markLine.lineStyle.type).toBe('dashed')
    expect(o.series[1].markLine).toBeUndefined()
  })

  it('moyenne d’avant : repère de la couleur de SA courbe', () => {
    const o = build(
      input({
        curves: [{ name: 'A', color: 'chart-series-3', points: points([1, 2]), prevMean: 1.5 }],
      }),
    )
    expect(o.series[0].markLine.data).toEqual([
      { yAxis: 1.5, lineStyle: { color: 'color:chart-series-3' } },
    ])
  })

  it('référence et moyenne d’avant coexistent sur la première courbe', () => {
    const o = build(
      input({
        reference: 1,
        curves: [{ name: 'A', color: 'chart-series-3', points: points([1, 2]), prevMean: 1.5 }],
      }),
    )
    expect(o.series[0].markLine.data).toHaveLength(2)
  })

  it('aucun repère : pas de markLine', () => {
    expect(build(input()).series[0].markLine).toBeUndefined()
  })
})

describe('buildTendancesLinesOption — légende', () => {
  it('absente pour une courbe seule sans axe de droite', () => {
    expect(build(input()).legend).toBeUndefined()
  })

  it('en bas et centrée dès deux courbes', () => {
    const o = build(input({ curves: DEUX_COURBES }))
    expect(o.legend?.bottom).toBe(0)
    expect(o.legend?.left).toBe('center')
    expect(o.legend?.data.map((e: { name: string }) => e.name)).toEqual(['A', 'B'])
    expect(o.legend?.itemWidth).toBe(12)
  })

  it('présente pour une courbe seule avec axe de droite, pastille élargie (pointillé)', () => {
    const o = build(input({ right: { name: 'MMR', points: points([1, 2]) } }))
    expect(o.legend?.data).toHaveLength(2)
    expect(o.legend?.itemWidth).toBe(30)
  })
})

describe('buildTendancesLinesOption — infobulle', () => {
  const param = (seriesIndex: number, v: number, matches: number) => ({
    seriesIndex,
    marker: '<i></i>',
    value: [TO, v],
    data: { matches },
  })

  it('une ligne par courbe avec les matchs du point, axe de droite en entier', () => {
    const o = build(input({ right: { name: 'MMR', points: points([1500, 1510]) } }))
    const html = o.tooltip.formatter([param(0, 1.5, 4), param(1, 1512.4, 4)])
    expect(html).toContain('FDA = 1,50 / 4')
    expect(html).toMatch(/MMR = 1\D512 \/ 4/)
  })

  it('pas « semaine » : l’en-tête passe par le libellé de semaine', () => {
    const o = build(input({ step: 'week' }))
    expect(o.tooltip.formatter([param(0, 1, 1)])).toContain('sem. ')
  })

  it('échappe les noms de courbe', () => {
    const o = build(
      input({ curves: [{ name: '<i>x</i>', color: 'chart-series-1', points: points([1, 2]) }] }),
    )
    expect(o.tooltip.formatter([param(0, 1, 1)])).toContain('&lt;i&gt;x&lt;/i&gt;')
  })
})
