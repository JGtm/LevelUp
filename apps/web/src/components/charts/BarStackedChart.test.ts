import { describe, it, expect, vi } from 'vitest'

import { buildBarStackedOption, type ChartPointStacked } from './BarStackedChart'
import type { ChartSeries } from './ChartCard'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
}))

describe('buildBarStackedOption', () => {
  const series: ChartSeries<ChartPointStacked>[] = [
    {
      key: 'test.stack',
      datapoints: [
        { category: 'Aquarius', components: { win: 5, loss: 2 } },
        { category: 'Recharge', components: { win: 3, loss: 4 } },
      ],
    },
  ]

  it('génère 1 ECharts series par component', () => {
    const opt = buildBarStackedOption(series) as { series: { name: string }[] }
    expect(opt.series).toHaveLength(2)
    expect(opt.series.map((s) => s.name).sort()).toEqual(['loss', 'win'])
  })

  it('catégories sur xAxis en orientation vertical (default)', () => {
    const opt = buildBarStackedOption(series) as { xAxis: { data: string[] } }
    expect(opt.xAxis.data).toEqual(['Aquarius', 'Recharge'])
  })

  it('catégories sur yAxis en orientation horizontal', () => {
    const opt = buildBarStackedOption(series, { orientation: 'horizontal' }) as {
      yAxis: { data: string[] }
    }
    expect(opt.yAxis.data).toEqual(['Aquarius', 'Recharge'])
  })

  it('toutes les bars partagent stack="total"', () => {
    const opt = buildBarStackedOption(series) as { series: { stack: string }[] }
    for (const s of opt.series) {
      expect(s.stack).toBe('total')
    }
  })

  it('respecte componentOrder', () => {
    const opt = buildBarStackedOption(series, {
      componentOrder: ['win', 'loss'],
    }) as { series: { name: string }[] }
    expect(opt.series.map((s) => s.name)).toEqual(['win', 'loss'])
  })

  it('applique componentColors via resolveToken', () => {
    const opt = buildBarStackedOption(series, {
      componentColors: { win: 'outcome-win', loss: 'outcome-loss' },
      componentOrder: ['win', 'loss'],
    }) as { series: { itemStyle: { color: string } }[] }
    expect(opt.series[0].itemStyle.color).toBe('var(outcome-win)')
    expect(opt.series[1].itemStyle.color).toBe('var(outcome-loss)')
  })

  it('series vide retourne option minimal', () => {
    const opt = buildBarStackedOption([])
    expect(opt).toEqual({ backgroundColor: 'transparent' })
  })

  // ─── L'INFOBULLE, INVOQUÉE POUR DE VRAI ────────────────────────────────────────────
  //
  // Le formateur personnalisé (masquage des zéros, note par segment) n'était couvert
  // que par sa PRÉSENCE : une option `tooltip.formatter` non nulle. Or c'est son SORTIE
  // qui porte le contrat — la note « dont N volées » doit atterrir sur la bonne paire
  // (assistant, tueur), et le masquage des zéros doit rester intact pour les appelants
  // qui l'utilisaient avant l'ajout de la note.
  type TooltipOpt = {
    tooltip: { formatter?: (raw: unknown) => string }
  }
  /** Un paramètre ECharts d'axe, tel que l'infobulle en reçoit un tableau. */
  const param = (seriesName: string, value: number | null, axisValueLabel: string) => ({
    seriesName,
    value,
    marker: '',
    axisValueLabel,
  })

  it('n\'installe AUCUN formateur sans tooltipHideZero', () => {
    const opt = buildBarStackedOption(series) as TooltipOpt
    expect(opt.tooltip.formatter).toBeUndefined()
  })

  it('tooltipHideZero : les segments à zéro disparaissent, le comportement d\'avant', () => {
    const opt = buildBarStackedOption(series, { tooltipHideZero: true }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('win', 0, 'Aquarius'), param('loss', 2, 'Aquarius')])
    expect(html).not.toContain('win')
    expect(html).toContain('loss')
  })

  it('tooltipHideZero : aucune ligne restante → infobulle VIDE (pas de cadre orphelin)', () => {
    const opt = buildBarStackedOption(series, { tooltipHideZero: true }) as TooltipOpt
    expect(opt.tooltip.formatter!([param('win', 0, 'Aquarius')])).toBe('')
  })

  it('le nom de segment et la catégorie sont échappés (pas d\'injection HTML)', () => {
    const opt = buildBarStackedOption(series, { tooltipHideZero: true }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('<b>win</b>', 5, '<i>Aquarius</i>')])
    expect(html).not.toContain('<b>win</b>')
    expect(html).not.toContain('<i>Aquarius</i>')
  })

  it('component absent d\'un dp → 0 (pas crash)', () => {
    const partial: ChartSeries<ChartPointStacked>[] = [
      {
        key: 'test',
        datapoints: [
          { category: 'A', components: { win: 5 } },
          { category: 'B', components: { loss: 3 } },
        ],
      },
    ]
    const opt = buildBarStackedOption(partial) as { series: { name: string; data: number[] }[] }
    const winSeries = opt.series.find((s) => s.name === 'win')!
    expect(winSeries.data).toEqual([5, 0]) // B sans win → 0
  })

  it('aucun titre d\'axe : marges fixes', () => {
    const opt = buildBarStackedOption(series) as {
      xAxis: { name?: string }
      yAxis: { name?: string }
      grid: { bottom: number; left: number }
    }
    expect(opt.xAxis.name).toBeUndefined()
    expect(opt.yAxis.name).toBeUndefined()
    expect(opt.grid.bottom).toBe(40)
    expect(opt.grid.left).toBe(8)
  })
})

/**
 * LES COLONNES GROUPÉES (2026-09-21, D18 — contrôle des armes spéciales). Cinq options
 * ajoutées, toutes par défaut inertes : sans elles l'option est exactement celle d'avant.
 */
describe('buildBarStackedOption — groupes, étiquettes de valeur, opacité', () => {
  const series: ChartSeries<ChartPointStacked>[] = [
    {
      key: 'pads',
      datapoints: [
        { category: 'Sniper', components: { Alpha: 5, Charlie: 3 } },
        { category: 'Épée', components: { Alpha: 0, Charlie: 2 } },
        { category: 'Hydra', components: { Alpha: 1, Charlie: 0 } },
      ],
    },
  ]
  const groupes = [
    { label: 'Puissance · 10 prises', span: 2 },
    { label: 'Terrain · 1 prise', span: 1 },
  ]

  it('par défaut : ni graphic, ni markLine, ni étiquette, légende présente', () => {
    const opt = buildBarStackedOption(series) as {
      graphic?: unknown
      legend: { data?: string[]; show?: boolean }
      series: { markLine?: unknown; label?: unknown }[]
    }
    expect(opt.graphic).toBeUndefined()
    expect(opt.legend.data).toEqual(['Alpha', 'Charlie'])
    expect(opt.series[0].markLine).toBeUndefined()
    expect(opt.series[0].label).toBeUndefined()
  })

  it('categoryGroups : UN seul trait, porté par la PREMIÈRE série, et deux titres', () => {
    const opt = buildBarStackedOption(series, { categoryGroups: groupes }) as {
      graphic: { children: { style: { text: string } }[] }
      series: { markLine?: { data: { xAxis: number }[] } }[]
    }
    expect(opt.series[0].markLine?.data).toEqual([{ xAxis: 1.5 }])
    expect(opt.series[1].markLine).toBeUndefined()
    expect(opt.graphic.children.map((c) => c.style.text)).toEqual([
      'Puissance · 10 prises',
      'Terrain · 1 prise',
    ])
  })

  it('showLegend false : la légende ECharts disparaît (une seule légende, celle du DOM)', () => {
    const opt = buildBarStackedOption(series, { showLegend: false }) as {
      legend: { show?: boolean }
    }
    expect(opt.legend.show).toBe(false)
  })

  it('valueLabels.segments : le compte DANS le segment, jamais un zéro, sans chevauchement', () => {
    const opt = buildBarStackedOption(series, { valueLabels: { segments: true } }) as {
      series: {
        label?: { show: boolean; position: string; formatter: (p: { value: number }) => string }
        labelLayout?: { hideOverlap: boolean }
      }[]
    }
    expect(opt.series[0].label?.position).toBe('inside')
    expect(opt.series[0].labelLayout?.hideOverlap).toBe(true)
    expect(opt.series[0].label?.formatter({ value: 5 })).toBe('5')
    expect(opt.series[0].label?.formatter({ value: 0 })).toBe('')
  })

  it('valueLabels.segments : le compte est blanc à contour sombre fin sur tous les segments, centré', () => {
    const opt = buildBarStackedOption(series, { valueLabels: { segments: true }, componentHexColors: { Alpha: '#fde68a', Charlie: '#1e3a8a' } }) as {
      series: { name: string; label?: { color: string; textBorderColor: string; verticalAlign: string } }[]
    }
    for (const name of ['Alpha', 'Charlie']) {
      expect(opt.series.find((x) => x.name === name)?.label).toMatchObject({ color: '#ffffff', textBorderColor: '#0b1220', verticalAlign: 'middle' })
    }
  })

  it('valueLabels.totals : une série muette de hauteur nulle écrit le total au sommet', () => {
    const opt = buildBarStackedOption(series, { valueLabels: { totals: [8, 2, 1] } }) as {
      series: {
        name: string
        silent?: boolean
        data: number[]
        label?: { formatter: (p: { dataIndex: number }) => string }
      }[]
    }
    const totaux = opt.series[opt.series.length - 1]
    expect(totaux.silent).toBe(true)
    expect(totaux.data).toEqual([0, 0, 0])
    expect(totaux.label?.formatter({ dataIndex: 0 })).toBe('8')
    expect(totaux.label?.formatter({ dataIndex: 2 })).toBe('1')
  })

  it('categoryNote : seconde ligne sous l’étiquette, et la catégorie reste nue sans note', () => {
    const opt = buildBarStackedOption(series, {
      categoryNote: (c) => (c === 'Sniper' ? '+ 4 sans nom' : undefined),
    }) as { xAxis: { axisLabel: { formatter: (v: string) => string } } }
    expect(opt.xAxis.axisLabel.formatter('Sniper')).toBe('Sniper\n{note|+ 4 sans nom}')
    expect(opt.xAxis.axisLabel.formatter('Hydra')).toBe('Hydra')
  })

  it('componentOpacity : l’opacité par sous-clé, opaque par défaut', () => {
    const opt = buildBarStackedOption(series, { componentOpacity: { Charlie: 0.6 } }) as {
      series: { name: string; itemStyle: { opacity: number } }[]
    }
    const par = Object.fromEntries(opt.series.map((s) => [s.name, s.itemStyle.opacity]))
    expect(par.Alpha).toBe(1)
    expect(par.Charlie).toBe(0.6)
  })
})
