/**
 * Tests — buildHistogramOption (Phase 3 P3.B).
 */
import { describe, it, expect } from 'vitest'
import { buildHistogramOption, type ChartPointHistogram } from './HistogramChart'
import type { ChartSeries } from './ChartCard'

function makeSeries(points: ChartPointHistogram[]): ChartSeries<ChartPointHistogram>[] {
  return [
    {
      key: 'test.histogram',
      meta: { gamertag: 'test' },
      datapoints: points,
    },
  ]
}

interface OptionShape {
  series?: Array<{ type?: string; data?: number[]; itemStyle?: { color?: string } }>
  xAxis?: { type?: string; data?: string[]; name?: string }
  yAxis?: { type?: string; name?: string }
  backgroundColor?: string
}

describe('buildHistogramOption', () => {
  it('retourne option vide si aucune série', () => {
    const opt = buildHistogramOption([]) as OptionShape
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('retourne option vide si série sans datapoints', () => {
    const opt = buildHistogramOption(makeSeries([])) as OptionShape
    expect(opt.series).toBeUndefined()
  })

  it('génère une série bar avec les counts dans l\'ordre', () => {
    const opt = buildHistogramOption(
      makeSeries([
        { binStart: 0, binEnd: 1, count: 3 },
        { binStart: 1, binEnd: 2, count: 7 },
        { binStart: 2, binEnd: 3, count: 5 },
      ]),
    ) as OptionShape
    expect(opt.series?.[0].type).toBe('bar')
    expect(opt.series?.[0].data).toEqual([3, 7, 5])
  })

  it('mappe binStart/binEnd en categories format integer', () => {
    const opt = buildHistogramOption(
      makeSeries([
        { binStart: 0, binEnd: 5, count: 1 },
        { binStart: 5, binEnd: 10, count: 2 },
      ]),
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['0–5', '5–10'])
  })

  it('formate les bornes float avec 2 décimales', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0.5, binEnd: 1.25, count: 1 }]),
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['0.50–1.25'])
  })

  it('respecte un formatBin custom', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
      { formatBin: (p) => `bucket-${p.count}` },
    ) as OptionShape
    expect(opt.xAxis?.data).toEqual(['bucket-1'])
  })

  it('expose xAxisLabel + yAxisLabel sur les axes', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
      { xAxisLabel: 'K/D', yAxisLabel: 'Matchs' },
    ) as OptionShape
    expect(opt.xAxis?.name).toBe('K/D')
    expect(opt.yAxis?.name).toBe('Matchs')
  })

  it('par défaut yAxisLabel = "Matchs"', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
    ) as OptionShape
    expect(opt.yAxis?.name).toBe('Matchs')
  })
})

// ─── BARRES ATTÉNUÉES (correction W2/W5, revue ronde 1 du 2026-09-06) ─────────
//
// « Montrées, jamais comptées » : les barres hors périmètre gardent la COULEUR DE
// SÉRIE et perdent en opacité — UN seul indice graphique. Pas de seconde teinte — aucun
// token sémantique du dépôt n'est achromatique dans les quatre palettes
// (`divergent-neutral` vaut blue-400 dans la palette par défaut), et une seconde
// couleur aurait donc dépendu de la palette pour rester neutre.

interface StyledBar {
  value: number
  itemStyle: { color?: string; opacity?: number }
}

function barres(opt: unknown): Array<number | StyledBar> {
  return (opt as { series: Array<{ data: Array<number | StyledBar> }> }).series[0].data
}

const troisBins: ChartPointHistogram[] = [
  { binStart: 0, binEnd: 1, count: 3 },
  { binStart: 1, binEnd: 2, count: 5 },
  { binStart: 2, binEnd: 3, count: 2 },
]

describe('buildHistogramOption — binAttenuated', () => {
  it('atténue les barres désignées et laisse les autres intactes', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), {
      binAttenuated: (p) => p.binStart >= 2,
    })
    const data = barres(opt)
    expect(typeof data[0]).toBe('number')
    expect(typeof data[1]).toBe('number')
    const attenuee = data[2] as StyledBar
    expect(attenuee.value).toBe(2)
    // L'opacité est le SEUL indice graphique : le liseré tireté qu'assertait cette
    // ligne n'a jamais été visible (même couleur que le remplissage, sous la même
    // opacité globale). Cadenasser une promesse que l'écran ne tient pas est pire que
    // ne rien cadenasser — correction R2 du 2026-09-06.
    expect(attenuee.itemStyle.opacity).toBeLessThan(1)
  })

  it('n’introduit AUCUNE seconde teinte : la barre atténuée porte la couleur de série', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), {
      colorToken: 'chart-series-1',
      binAttenuated: (p) => p.binStart >= 2,
    })
    const data = barres(opt)
    const attenuee = data[2] as StyledBar
    const serie = (opt as { series: Array<{ itemStyle: { color: string } }> }).series[0]
    expect(attenuee.itemStyle.color).toBe(serie.itemStyle.color)
  })

  it('RÉTRO-COMPAT : sans la prop, `data` reste un tableau de nombres nus', () => {
    // INVERSION JOUÉE : en emballant systématiquement chaque valeur en objet, ce test
    // tombe — c'est ce qui garantit que les appelants historiques ne changent pas de
    // rendu d'un iota.
    const opt = buildHistogramOption(makeSeries(troisBins))
    expect(barres(opt)).toEqual([3, 5, 2])
  })

  it('un prédicat toujours faux équivaut à l’absence de prop', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), { binAttenuated: () => false })
    expect(barres(opt)).toEqual([3, 5, 2])
  })
})
