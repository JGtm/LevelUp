/**
 * _divergentStepOption.test.ts — l'option ECharts commune aux deux graphes divergents de
 * l'onglet Chronologie (Écart de score, Supériorité numérique).
 */
import { describe, expect, it, vi } from 'vitest'

import type { EChartsThemeColors } from '@/components/charts/_utils'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const { buildDivergentStepOption, symmetricExtent } = await import('./_divergentStepOption')

const TC: EChartsThemeColors = {
  axisLabel: 'LABEL',
  axisLine: 'LINE',
  splitLine: 'SPLIT',
  splitAreaA: 'A',
  splitAreaB: 'B',
  text: 'TEXT',
  tooltipBg: 'BG',
  tooltipBorder: 'BORDER',
  card: 'CARD',
  isDark: false,
}

function option(over: Partial<Parameters<typeof buildDivergentStepOption>[0]> = {}) {
  return buildDivergentStepOption(
    {
      points: [
        { ms: 0, value: 0 },
        { ms: 10_000, value: 3 },
        { ms: 20_000, value: -2 },
        { ms: 30_000, value: -2 },
      ],
      endMs: 30_000,
      positiveName: 'Équipe en tête',
      negativeName: 'Adversaires en tête',
      tooltipHtml: (ms) => `INFOBULLE ${ms}`,
      ...over,
    },
    TC,
  ) as Record<string, any> // eslint-disable-line @typescript-eslint/no-explicit-any
}

describe('buildDivergentStepOption — la forme', () => {
  it('pose DEUX séries d’aire en escalier : l’une au-dessus de zéro, l’autre en dessous', () => {
    const o = option()
    expect(o.series).toHaveLength(2)
    expect(o.series.map((s: { step: string }) => s.step)).toEqual(['end', 'end'])
    expect(o.series[0].data).toEqual([[0, 0], [10_000, 3], [20_000, 0], [30_000, 0]])
    expect(o.series[1].data).toEqual([[0, 0], [10_000, 0], [20_000, -2], [30_000, -2]])
  })

  it('peint chaque moitié d’un jeton DIVERGENT du projet, jamais d’une valeur en dur', () => {
    const o = option()
    expect(o.series[0].areaStyle.color).toBe('var(--ac-divergent-pos)')
    expect(o.series[1].areaStyle.color).toBe('var(--ac-divergent-neg)')
    expect(JSON.stringify(o)).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
  })

  it('UN SEUL axe des ordonnées, symétrique autour de zéro, et un seul axe des temps', () => {
    const o = option()
    expect(Array.isArray(o.yAxis)).toBe(false)
    expect(Array.isArray(o.xAxis)).toBe(false)
    expect(o.yAxis.min).toBe(-o.yAxis.max)
    expect(o.xAxis).toMatchObject({ type: 'value', min: 0, max: 30_000 })
  })

  it('étiquette l’axe des ordonnées avec le signe : « +2 », « 0 », « −2 »', () => {
    const f = option().yAxis.axisLabel.formatter as (v: number) => string
    expect(f(2)).toBe('+2')
    expect(f(0)).toBe('0')
    expect(f(-2)).toBe('−2')
  })

  it('trace la LIGNE DE ZÉRO, une seule fois, sur la première série', () => {
    const o = option()
    expect(o.series[0].markLine.data).toEqual([{ yAxis: 0 }])
    expect(o.series[1].markLine).toBeUndefined()
  })

  it('met la légende EN BAS, aux couleurs des aires, avec les deux noms de série', () => {
    const o = option()
    expect(o.legend.bottom).toBe(0)
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual([
      'Équipe en tête',
      'Adversaires en tête',
    ])
    expect(o.legend.data[0].itemStyle.color).toBe('var(--ac-divergent-pos)')
    expect(o.legend.data[1].itemStyle.color).toBe('var(--ac-divergent-neg)')
  })

  it('partage les marges et l’axe des temps du graphe « Score dans le temps » (alignement)', () => {
    const o = option()
    expect(o.grid).toMatchObject({ bottom: 44, left: 40, right: 16, top: 12 })
    expect(o.xAxis.splitLine.show).toBe(false)
    expect(o.xAxis.axisLabel.formatter(65_000)).toBe('1:05')
  })

  it('le texte des axes et de la légende est à l’encre du thème, jamais à celle d’une série', () => {
    const o = option()
    expect(o.xAxis.axisLabel.color).toBe('LABEL')
    expect(o.yAxis.axisLabel.color).toBe('LABEL')
    expect(o.legend.textStyle.color).toBe('LABEL')
  })
})

describe('buildDivergentStepOption — l’infobulle', () => {
  it('lit l’instant de la souris (axisValue), pas le point le plus proche', () => {
    const f = option().tooltip.formatter as (p: unknown) => string
    expect(f([{ axisValue: 12_345, value: [10_000, 3] }])).toBe('INFOBULLE 12345')
  })

  it('retombe sur l’abscisse du point quand l’instant de la souris manque', () => {
    const f = option().tooltip.formatter as (p: unknown) => string
    expect(f([{ value: [20_000, -2] }])).toBe('INFOBULLE 20000')
  })

  it('déclenche sur l’axe : une infobulle par instant, pas par aire', () => {
    expect(option().tooltip.trigger).toBe('axis')
  })
})

describe('symmetricExtent — un axe symétrique aux graduations rondes', () => {
  it('garde le plancher quand l’écart est petit : ±4, par pas de 2', () => {
    expect(symmetricExtent(1, 4)).toEqual({ max: 4, interval: 2 })
    expect(symmetricExtent(4, 4)).toEqual({ max: 4, interval: 2 })
  })

  it('arrondit au pas supérieur quand l’écart dépasse le plancher', () => {
    expect(symmetricExtent(12, 4)).toEqual({ max: 15, interval: 5 })
    expect(symmetricExtent(37, 4)).toEqual({ max: 40, interval: 20 })
  })

  it('ne rend jamais un maximum inférieur à l’écart lu', () => {
    for (const v of [0, 1, 3, 5, 9, 17, 49, 51, 130, 999]) {
      expect(symmetricExtent(v, 4).max).toBeGreaterThanOrEqual(v)
    }
  })
})
