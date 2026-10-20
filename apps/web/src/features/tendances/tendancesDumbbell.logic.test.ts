/**
 * Tests du builder d'haltères : deux points et segment, point seul, repère, étiquettes,
 * légende en bas centrée et limitée aux séries non vides, couleurs par jeton, infobulle.
 */
import { describe, expect, it, vi } from 'vitest'

import {
  buildTendancesDumbbellOption,
  dumbbellHeight,
  dumbbellXBounds,
  type DumbbellRow,
  type TendancesDumbbellInput,
} from './tendancesDumbbell.logic'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `color:${token}`,
  resolveToken: (token: string) => `color:${token}`,
}))

function row(overrides: Partial<DumbbellRow> = {}): DumbbellRow {
  return { label: 'FDA', a: -0.5, b: 0.8, aText: '0,9', bText: '1,4', tooltip: ['ligne 1'], ...overrides }
}

function input(overrides: Partial<TendancesDumbbellInput> = {}): TendancesDumbbellInput {
  return {
    rows: [row(), row({ label: 'Précision', a: 0.1, b: 0.4, aText: '44 %', bText: '47 %' })],
    nameA: 'Défaites',
    nameB: 'Victoires',
    colorA: 'outcome-loss',
    colorB: 'outcome-win',
    xAxisLabel: null,
    ...overrides,
  }
}

/* eslint-disable @typescript-eslint/no-explicit-any -- lecture d'une option ECharts non typée */
type Opt = {
  series: Array<Record<string, any>>
  xAxis: Record<string, any>
  yAxis: Record<string, any>
  legend: Record<string, any>
  tooltip: { formatter: (p: unknown) => string }
}
/* eslint-enable @typescript-eslint/no-explicit-any */
const build = (i: TendancesDumbbellInput) => buildTendancesDumbbellOption(i) as unknown as Opt
const valuesOf = (s: { data: Array<{ value: number[] }> }) => s.data.map((d) => d.value)

describe('buildTendancesDumbbellOption', () => {
  it('sans ligne : option vide', () => {
    expect(Object.keys(buildTendancesDumbbellOption(input({ rows: [] })))).toEqual(['backgroundColor'])
  })

  it('segment + deux séries de points, axe Y de catégories inversé dans l’ordre reçu', () => {
    const o = build(input())
    expect(o.series.map((s) => s.type)).toEqual(['custom', 'scatter', 'scatter'])
    expect(o.series[0].data).toEqual([
      [0, -0.5, 0.8],
      [1, 0.1, 0.4],
    ])
    expect(o.series[1].name).toBe('Défaites')
    expect(o.series[2].name).toBe('Victoires')
    expect(valuesOf(o.series[1] as never)).toEqual([
      [-0.5, 0],
      [0.1, 1],
    ])
    expect(o.yAxis.type).toBe('category')
    expect(o.yAxis.inverse).toBe(true)
    expect(o.yAxis.data).toEqual(['FDA', 'Précision'])
  })

  it('ligne sans point A : un point B seul, pas de segment, indices de lignes conservés', () => {
    const o = build(
      input({ rows: [row({ a: null, aText: undefined }), row({ label: 'Autre', a: 0.2, b: 0.6 })] }),
    )
    expect(o.series[0].data).toEqual([[1, 0.2, 0.6]])
    expect(valuesOf(o.series[1] as never)).toEqual([[0.2, 1]])
    expect(valuesOf(o.series[2] as never)).toEqual([
      [0.8, 0],
      [0.6, 1],
    ])
  })

  it('couleurs : les jetons sont résolus dans le builder', () => {
    const o = build(input())
    expect(o.series[1].itemStyle.color).toBe('color:outcome-loss')
    expect(o.series[2].itemStyle.color).toBe('color:outcome-win')
    expect(o.series[0].renderItem).toBeTypeOf('function')
  })

  it('étiquettes au-dessus des points : le texte de la ligne', () => {
    const o = build(input())
    expect(o.series[1].label.position).toBe('top')
    const fa = o.series[1].label.formatter as (p: unknown) => string
    const fb = o.series[2].label.formatter as (p: unknown) => string
    expect(fa({ data: o.series[1].data[0] })).toBe('0,9')
    expect(fb({ data: o.series[2].data[1] })).toBe('47 %')
  })

  it('repère vertical en pointillé quand il est fourni, absent sinon', () => {
    const avec = build(input({ reference: 0 }))
    const repere = avec.series.find((s) => s.markLine)?.markLine
    expect(repere.data).toEqual([{ xAxis: 0 }])
    expect(repere.lineStyle.type).toBe('dashed')
    expect(build(input()).series.some((s) => s.markLine)).toBe(false)
  })

  it('axe X : étiquettes masquées, ou formatées par la fonction fournie', () => {
    expect(build(input()).xAxis.axisLabel).toEqual({ show: false })
    const o = build(input({ xAxisLabel: (v) => `${v} /match` }))
    expect(o.xAxis.axisLabel.formatter(0.5)).toBe('0.5 /match')
    expect(o.xAxis.type).toBe('value')
  })

  it('axe X tiré des SEULS points : le segment encode ses valeurs sur X, la ligne sur Y', () => {
    // Sans `encode`, la première dimension du segment (le numéro de ligne) étirait l'axe X
    // jusqu'au nombre de lignes : tous les points tassés dans un coin.
    expect(build(input()).series[0].encode).toEqual({ x: [1, 2], y: 0 })
  })

  it('axe X avec repère : symétrique autour de lui, le point le plus éloigné près du bord', () => {
    const o = build(input({ reference: 0 }))
    // Point le plus éloigné du repère : 0,8 ; marge de 12 %.
    expect(o.xAxis.min).toBeCloseTo(-0.896)
    expect(o.xAxis.max).toBeCloseTo(0.896)
    expect(o.xAxis.scale).toBeUndefined()
  })

  it('axe X sans repère : ajusté aux points, sans bornes imposées', () => {
    const o = build(input())
    expect(o.xAxis.scale).toBe(true)
    expect(o.xAxis.min).toBeUndefined()
    expect(dumbbellXBounds(input().rows, undefined)).toBeNull()
    expect(dumbbellXBounds([], 0)).toBeNull()
  })

  it('légende en bas et centrée, avec les deux séries', () => {
    const o = build(input())
    expect(o.legend.bottom).toBe(0)
    expect(o.legend.left).toBe('center')
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual(['Défaites', 'Victoires'])
    expect(o.legend.data[0].itemStyle.color).toBe('color:outcome-loss')
  })

  it('légende limitée aux séries qui ont au moins un point', () => {
    const o = build(input({ rows: [row({ a: null }), row({ a: null })] }))
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual(['Victoires'])
  })

  it('infobulle : libellé de la ligne puis ses lignes, texte échappé', () => {
    const o = build(input({ rows: [row({ label: 'A<b>', tooltip: ['x < y', 'ok'] })] }))
    const html = o.tooltip.formatter([
      { seriesType: 'custom', value: [0, -0.5, 0.8] },
      { seriesType: 'scatter', value: [0.8, 0] },
    ])
    expect(html).toBe('<b>A&lt;b&gt;</b><br/>x &lt; y<br/>ok')
    expect(o.tooltip.formatter([])).toBe('')
  })
})

describe('dumbbellHeight', () => {
  it('proportionnelle au nombre de lignes, avec un plancher', () => {
    expect(dumbbellHeight(1)).toBe(160)
    expect(dumbbellHeight(10)).toBeGreaterThan(dumbbellHeight(5))
    expect(dumbbellHeight(10) - dumbbellHeight(9)).toBe(dumbbellHeight(9) - dumbbellHeight(8))
  })
})
