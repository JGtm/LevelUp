import { describe, it, expect, vi } from 'vitest'

import {
  buildBarStackedOption,
  categoryLabelBandPx,
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

  // ─── RÔLES NOMMÉS (Larbin / Patron, 2026-09-17) ───────────────────────────────────
  //
  // Sur les assistances d'escouade, barre et segment sont deux gamertags : sans les
  // nommer, rien ne dit qui a assisté et qui a eu le frag. Le titre d'axe porte le rôle
  // de la barre, l'infobulle les deux — et les appelants qui ne demandent rien gardent
  // exactement le graphe d'avant.
  type AxisOpt = {
    xAxis: { name?: string; nameLocation?: string; nameGap?: number }
    yAxis: { name?: string; nameLocation?: string; nameGap?: number }
    grid: { bottom: number; left: number }
  }

  it('sans titre d\'axe : aucun `name`, marges inchangées', () => {
    const opt = buildBarStackedOption(series) as AxisOpt
    expect(opt.xAxis.name).toBeUndefined()
    expect(opt.yAxis.name).toBeUndefined()
    expect(opt.grid.bottom).toBe(40)
    expect(opt.grid.left).toBe(8)
  })

  it('categoryAxisName : titre posé au MILIEU de l\'axe des catégories, marge basse élargie', () => {
    const opt = buildBarStackedOption(series, { categoryAxisName: 'Larbin' }) as AxisOpt
    expect(opt.xAxis.name).toBe('Larbin')
    expect(opt.xAxis.nameLocation).toBe('middle')
    expect(opt.xAxis.nameGap).toBeGreaterThan(0)
    expect(opt.yAxis.name).toBeUndefined()
    expect(opt.grid.bottom).toBeGreaterThan(40)
    expect(opt.grid.left).toBe(8)
  })

  it('valueAxisName : titre sur l\'axe des valeurs (Y en vertical), marge gauche élargie', () => {
    const opt = buildBarStackedOption(series, { valueAxisName: 'Assistances par patron' }) as AxisOpt
    expect(opt.yAxis.name).toBe('Assistances par patron')
    expect(opt.yAxis.nameLocation).toBe('middle')
    expect(opt.xAxis.name).toBeUndefined()
    expect(opt.grid.left).toBeGreaterThan(8)
    expect(opt.grid.bottom).toBe(40)
  })

  it('orientation horizontale : chaque titre suit son axe (catégories sur Y, valeurs sur X)', () => {
    const opt = buildBarStackedOption(series, {
      categoryAxisName: 'Larbin',
      valueAxisName: 'Assistances par patron',
      orientation: 'horizontal',
    }) as AxisOpt
    expect(opt.yAxis.name).toBe('Larbin')
    expect(opt.xAxis.name).toBe('Assistances par patron')
    expect(opt.grid.left).toBeGreaterThan(8)
    expect(opt.grid.bottom).toBeGreaterThan(40)
  })

  // LE TITRE DE L'AXE VERTICAL DOIT FRANCHIR SES ÉTIQUETTES (2026-09-22). `nameGap` se mesure
  // depuis la ligne d'axe, et `containLabel` ne réserve que les étiquettes : un écart fixe de
  // 28 px posait « Larbin » PAR-DESSUS les gamertags, donc illisible. L'écart suit désormais la
  // largeur du plus long libellé.
  it('barres horizontales : le titre des catégories franchit la bande des libellés', () => {
    const longs: ChartSeries<ChartPointStacked>[] = [
      {
        key: 'test.stack',
        datapoints: [
          { category: 'UnGamertagTresLong', components: { win: 5 } },
          { category: 'Court', components: { win: 3 } },
        ],
      },
    ]
    const opt = buildBarStackedOption(longs, {
      categoryAxisName: 'Larbin',
      orientation: 'horizontal',
    }) as AxisOpt
    expect(opt.yAxis.nameGap).toBeGreaterThan(categoryLabelBandPx(['UnGamertagTresLong']))
    // Et il grandit avec le libellé : deux nuages de gamertags ne se valent pas.
    const courts = buildBarStackedOption(series, {
      categoryAxisName: 'Larbin',
      orientation: 'horizontal',
    }) as AxisOpt
    expect(opt.yAxis.nameGap!).toBeGreaterThan(courts.yAxis.nameGap!)
  })

  it('barres verticales : le titre des catégories garde un écart fixe (libellés sous la ligne)', () => {
    const opt = buildBarStackedOption(series, { categoryAxisName: 'Larbin' }) as AxisOpt
    expect(opt.xAxis.nameGap).toBe(28)
  })

  it('tooltipRoles seul : le formateur s\'installe et nomme les deux rôles', () => {
    const opt = buildBarStackedOption(series, {
      tooltipRoles: { category: 'Larbin', component: 'Patron' },
    }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('Chocoboflor', 3, 'JGtm')])
    expect(html).toContain('Larbin · JGtm')
    expect(html).toContain('Patron · Chocoboflor')
    expect(html).toContain('<strong>3</strong>')
  })

  it('tooltipRoles + tooltipComponentNote : la note reçoit le nom NU, pas le rôle', () => {
    const vues: Array<[string, string]> = []
    const opt = buildBarStackedOption(series, {
      tooltipRoles: { category: 'Larbin', component: 'Patron' },
      tooltipComponentNote: (category, component) => {
        vues.push([category, component])
        return 'part 50,0 %'
      },
    }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('Chocoboflor', 3, 'JGtm')])
    expect(vues).toEqual([['JGtm', 'Chocoboflor']])
    expect(html).toContain('Patron · Chocoboflor')
    expect(html).toContain('part 50,0 %')
  })

  it('tooltipRoles : les rôles sont échappés comme le reste', () => {
    const opt = buildBarStackedOption(series, {
      tooltipRoles: { category: '<b>L</b>', component: '<i>P</i>' },
    }) as TooltipOpt
    const html = opt.tooltip.formatter!([param('X', 1, 'Y')])
    expect(html).not.toContain('<b>L</b>')
    expect(html).not.toContain('<i>P</i>')
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
