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

  // Le builder est PUR : il n'a pas de locale, donc aucun libellé par défaut (ce serait un
  // littéral FR, cf. chartEmptyStateCanonical.guard.test.ts). Le défaut bilingue
  // (« Matchs » / « Matches », common.charts.axis_matches) est résolu par le composant,
  // qui lit la locale du shell — couvert par HistogramChart.locale.test.tsx.
  it('sans yAxisLabel le builder ne pose aucun libellé (le composant le fournit)', () => {
    const opt = buildHistogramOption(
      makeSeries([{ binStart: 0, binEnd: 1, count: 1 }]),
    ) as OptionShape
    expect(opt.yAxis?.name).toBe('')
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
  itemStyle: { color?: string; opacity?: number; decal?: Record<string, unknown> }
}

function barres(opt: unknown): Array<number | StyledBar> {
  return (opt as { series: Array<{ data: Array<number | StyledBar> }> }).series[0].data
}

const troisBins: ChartPointHistogram[] = [
  { binStart: 0, binEnd: 1, count: 3 },
  { binStart: 1, binEnd: 2, count: 5 },
  { binStart: 2, binEnd: 3, count: 2 },
]

describe('buildHistogramOption — binHatched', () => {
  it('hachure ET atténue les barres désignées, et laisse les autres intactes', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), {
      binHatched: (p) => p.binStart >= 2,
    })
    const data = barres(opt)
    expect(typeof data[0]).toBe('number')
    expect(typeof data[1]).toBe('number')
    const horsFenetre = data[2] as StyledBar
    expect(horsFenetre.value).toBe(2)
    expect(horsFenetre.itemStyle.opacity).toBeLessThan(1)
    // LA HACHURE EST LE SECOND INDICE, et elle doit EXISTER dans l'option : la version
    // 2026-09-06 promettait deux indices et n'en peignait qu'un (un liseré tireté de la
    // couleur du remplissage). Un `decal` ECharts, lui, se voit.
    expect(horsFenetre.itemStyle.decal).toBeDefined()
  })

  it('active `aria.decal` — sans quoi ECharts IGNORE la hachure en silence', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), { binHatched: () => true }) as {
      aria?: { decal?: { show?: boolean } }
    }
    expect(opt.aria?.decal?.show).toBe(true)
  })

  it('n’introduit AUCUNE seconde teinte : la barre hachurée porte la couleur de série', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), {
      colorToken: 'chart-series-1',
      binHatched: (p) => p.binStart >= 2,
    })
    const data = barres(opt)
    const horsFenetre = data[2] as StyledBar
    const serie = (opt as { series: Array<{ itemStyle: { color: string } }> }).series[0]
    expect(horsFenetre.itemStyle.color).toBe(serie.itemStyle.color)
  })

  it('RÉTRO-COMPAT : sans la prop, `data` reste un tableau de nombres nus', () => {
    // INVERSION JOUÉE : en emballant systématiquement chaque valeur en objet, ce test
    // tombe — c'est ce qui garantit que les appelants historiques ne changent pas de
    // rendu d'un iota.
    const opt = buildHistogramOption(makeSeries(troisBins))
    expect(barres(opt)).toEqual([3, 5, 2])
  })

  it('un prédicat toujours faux équivaut à l’absence de prop', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), { binHatched: () => false })
    expect(barres(opt)).toEqual([3, 5, 2])
  })
})

describe('buildHistogramOption — showValues et windowMark', () => {
  function serie(opt: unknown) {
    return (opt as {
      series: Array<{
        label?: { show?: boolean; position?: string }
        markLine?: { data: Array<{ xAxis: number }>; label: { formatter: string } }
      }>
    }).series[0]
  }

  it('écrit la valeur AU-DESSUS de chaque barre quand on le demande', () => {
    expect(serie(buildHistogramOption(makeSeries(troisBins), { showValues: true })).label).toEqual(
      expect.objectContaining({ show: true, position: 'top' }),
    )
  })

  it('ne montre AUCUNE étiquette par défaut (rétro-compat)', () => {
    expect(serie(buildHistogramOption(makeSeries(troisBins))).label?.show).toBe(false)
  })

  it('pose le repère de fenêtre sur la FRONTIÈRE qui précède l’intervalle visé', () => {
    const opt = buildHistogramOption(makeSeries(troisBins), {
      windowMark: { binIndex: 2, label: 'fenêtre 5 s' },
    })
    const ml = serie(opt).markLine
    // 1,5 = la frontière entre la 2e et la 3e catégorie, JAMAIS le centre d'une barre :
    // une fenêtre est une borne, pas un intervalle.
    expect(ml?.data).toEqual([{ xAxis: 1.5 }])
    expect(ml?.label.formatter).toBe('fenêtre 5 s')
  })

  it('ne pose AUCUN repère hors des bornes (index 0 ou au-delà du dernier)', () => {
    expect(
      serie(buildHistogramOption(makeSeries(troisBins), { windowMark: { binIndex: 0, label: 'x' } }))
        .markLine,
    ).toBeUndefined()
    expect(
      serie(buildHistogramOption(makeSeries(troisBins), { windowMark: { binIndex: 9, label: 'x' } }))
        .markLine,
    ).toBeUndefined()
  })
})

// ─── SEUILS FRACTIONNAIRES, SEULS ET MÊLÉS À LA BORNE DE FENÊTRE ─────────────
//
// `thresholds.at` est une position en INDICE DE CATÉGORIE, fractionnaire : un repère de
// médiane tombe DANS une barre, à sa fraction, jamais sur une frontière. ECharts n'accepte
// qu'un `markLine` par série : quand la borne de fenêtre et les seuils coexistent (la carte
// « Riposte » depuis le 2026-09-22), leurs données sont RÉUNIES — et le formatter doit
// rendre le nom de chaque seuil sans effacer le texte de la borne.

describe('buildHistogramOption — thresholds', () => {
  function markLine(opt: unknown) {
    return (
      opt as {
        series: Array<{
          markLine?: {
            data: Array<{ xAxis: number; name?: string }>
            label: { formatter: string | ((p: { name?: string }) => string) }
          }
        }>
      }
    ).series[0].markLine
  }

  it('pose un seuil à sa position FRACTIONNAIRE, sans l’arrondir à une frontière', () => {
    const ml = markLine(
      buildHistogramOption(makeSeries(troisBins), {
        thresholds: [{ at: 1.6, label: 'médiane 2,1 s' }],
      }),
    )
    expect(ml?.data).toEqual([{ xAxis: 1.6, name: 'médiane 2,1 s' }])
  })

  it('RÉUNIT la borne de fenêtre et les seuils sur une seule markLine', () => {
    const ml = markLine(
      buildHistogramOption(makeSeries(troisBins), {
        windowMark: { binIndex: 2, label: 'fenêtre 5 s' },
        thresholds: [{ at: 1.6, label: 'médiane 2,1 s' }],
      }),
    )
    expect(ml?.data).toEqual([
      { xAxis: 1.5 },
      { xAxis: 1.6, name: 'médiane 2,1 s' },
    ])
    // Le formatter rend le NOM du seuil quand il existe, et le texte de la fenêtre sinon.
    const fmt = ml?.label.formatter as (p: { name?: string }) => string
    expect(fmt({ name: 'médiane 2,1 s' })).toBe('médiane 2,1 s')
    expect(fmt({})).toBe('fenêtre 5 s')
  })

  it('ne pose AUCUNE markLine sans borne ni seuil', () => {
    expect(markLine(buildHistogramOption(makeSeries(troisBins), { thresholds: [] }))).toBeUndefined()
  })
})
