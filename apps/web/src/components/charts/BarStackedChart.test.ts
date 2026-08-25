import { describe, it, expect, vi } from 'vitest'

import {
  buildBarStackedOption,
  type ChartPointStacked,
} from './BarStackedChart'
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

  it('n\'installe AUCUN formateur sans tooltipHideZero ni tooltipComponentNote', () => {
    const opt = buildBarStackedOption(series) as TooltipOpt
    expect(opt.tooltip.formatter).toBeUndefined()
  })

  it('tooltipComponentNote : la note tombe sur la bonne paire (catégorie, composant)', () => {
    const vues: Array<[string, string]> = []
    const opt = buildBarStackedOption(series, {
      tooltipComponentNote: (category, component) => {
        vues.push([category, component])
        return category === 'Aquarius' && component === 'win' ? 'dont 2 volées' : undefined
      },
    }) as TooltipOpt

    const html = opt.tooltip.formatter!([
      param('win', 5, 'Aquarius'),
      param('loss', 2, 'Aquarius'),
    ])
    // La note est demandée pour CHAQUE segment de la catégorie survolée, et pour elle seule.
    expect(vues).toEqual([
      ['Aquarius', 'win'],
      ['Aquarius', 'loss'],
    ])
    expect(html).toContain('Aquarius')
    expect(html).toContain('dont 2 volées')
    // `loss` reste affiché, sans note.
    expect(html).toContain('loss')
    expect(html.match(/dont 2 volées/g)).toHaveLength(1)
  })

  it('tooltipComponentNote seul : les zéros restent affichés (pas de masquage implicite)', () => {
    const opt = buildBarStackedOption(series, {
      tooltipComponentNote: () => undefined,
    }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('win', 0, 'Aquarius'), param('loss', 2, 'Aquarius')])
    expect(html).toContain('win')
    expect(html).toContain('loss')
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

  it('la note et le nom de segment sont échappés (pas d\'injection HTML)', () => {
    const opt = buildBarStackedOption(series, {
      tooltipComponentNote: () => '<img src=x>',
    }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('<b>win</b>', 5, 'Aquarius')])
    expect(html).not.toContain('<img src=x>')
    expect(html).not.toContain('<b>win</b>')
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
})
