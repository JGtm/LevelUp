/**
 * sessionBarsTrendChart — LA frise « une soirée, un bâton », partagée.
 *
 * Hissée depuis `features/squad/charts/squadRiposteSessionsChart.ts` (lot J) le
 * 2026-09-22 : la page Séries temporelles demande la même frise sur deux sujets
 * (Riposte, Appui reçu) et la règle des deux copies interdit de la recopier. Le module
 * de l'Escouade n'est plus qu'un ADAPTATEUR — il traduit sa `FriseRiposte` en séries
 * génériques et n'écrit plus une seule clé d'option ECharts.
 *
 * GRAMMAIRE TENUE (celle de `squadSessionTimelineChart.ts`, désignée comme référence par
 * l'utilisateur) : bâtons à 18 px, courbe pleine 2 px NON lissée, légende nommant chaque
 * série, axes en 10 px gris.
 *
 * TROIS INVARIANTS, quel que soit le nombre de séries :
 *
 *   1. UN SEUL AXE Y. Tout est en points de pourcentage — taux d'une soirée, tendance,
 *      repère d'habituel : deux axes seraient une faute de lecture (et la règle dataviz
 *      « jamais deux échelles dans un graphe »).
 *   2. LE REPÈRE D'HABITUEL EST UNE LIGNE TIRETÉE, jamais un bâton : une valeur de
 *      comparaison n'est pas une mesure de la période. Chaque série porte le sien.
 *   3. LE DÉNOMINATEUR VOYAGE AVEC LE TAUX : soit en second rang d'étiquettes sous l'axe
 *      des dates (`volumeAxis`), soit en lignes d'infobulle (`tooltipLines`). Une frise
 *      qui n'en montre aucun laisse lire une soirée à 3 morts comme une soirée à 80.
 *
 * UN BÂTON CREUX (`hollow`) dit « échantillon faible » : il est rendu en contour, jamais
 * absent — la trame du temps ne doit pas mentir — et l'appelant l'exclut de sa tendance.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from './_utils'

/** Le repère de comparaison d'une série : sa valeur en POURCENTS et son libellé. */
export interface SessionBarsUsual {
  valuePct: number
  label: string
  /** Couleur du trait et de son étiquette (défaut : le gris des axes). */
  color?: string
}

/** La moyenne glissante d'une série, déjà calculée par l'appelant. */
export interface SessionBarsTrend {
  valuesPct: (number | null)[]
  label: string
  color: string
}

/** Une série de bâtons : une grandeur, une couleur, son repère et sa tendance. */
export interface SessionBarsSeriesSpec {
  name: string
  color: string
  /** Taux par soirée, en POURCENTS, aligné sur `labels`. `null` = pas de bâton. */
  valuesPct: (number | null)[]
  /** Par soirée : bâton en contour (échantillon faible). */
  hollow?: boolean[]
  /** Nom de pile — deux séries de même pile occupent la même colonne (verdicts à trous). */
  stack?: string
  usual?: SessionBarsUsual
  trend?: SessionBarsTrend
}

export interface SessionBarsTrendOpts {
  /** Les soirées, dans l'ordre du temps. */
  labels: string[]
  series: SessionBarsSeriesSpec[]
  yAxisLabel: string
  /** Second rang d'étiquettes sous l'axe des dates (un volume par soirée). */
  volumeAxis?: { label: string; values: string[] }
  /** Lignes ajoutées à l'infobulle d'une soirée (ses dénominateurs). */
  tooltipLines?: (index: number) => string[]
}

function round1(v: number): number {
  return parseFloat(v.toFixed(1))
}

/**
 * shortSessionLabel réduit un libellé de session à sa DATE.
 *
 * Le libellé complet (« 13/10/2025 22:27–22:46 (3) ») porte la plage horaire et le nombre
 * de matchs : posé sur quarante graduations d'axe, il se chevauche et devient illisible.
 * L'axe dit QUAND — la soirée —, pas le détail. Un libellé sans espace est rendu tel
 * quel : on ne coupe jamais à l'aveugle.
 */
export function shortSessionLabel(label: string): string {
  const espace = label.indexOf(' ')
  return espace > 0 ? label.slice(0, espace) : label
}

function barData(spec: SessionBarsSeriesSpec): unknown[] {
  return spec.valuesPct.map((v, i) => {
    if (v == null) return null
    const value = round1(v)
    if (!spec.hollow?.[i]) return value
    // Bâton CREUX : contour de la couleur de la série sur un fond transparent. Le bâton
    // reste à sa place et à sa hauteur — c'est sa FIABILITÉ qui est dite, pas sa valeur.
    return {
      value,
      itemStyle: { color: 'transparent', borderColor: spec.color, borderWidth: 1.5 },
    }
  })
}

function markLineOf(usual: SessionBarsUsual, axisColor: string): Record<string, unknown> {
  const couleur = usual.color ?? axisColor
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { type: 'dashed', color: couleur, width: 1 },
    label: { formatter: usual.label, color: couleur, fontSize: 10, position: 'insideEndTop' },
    data: [{ yAxis: round1(usual.valuePct) }],
  }
}

/**
 * buildSessionBarsTrendOption — l'option ECharts complète de la frise.
 *
 * ORDRE DE RENDU : tous les bâtons dans l'ordre des séries, puis toutes les tendances.
 * Une courbe doit passer AU-DESSUS des bâtons, y compris de ceux d'une série voisine.
 */
export function buildSessionBarsTrendOption(opts: SessionBarsTrendOpts): EChartsCoreOption {
  if (opts.labels.length === 0 || opts.series.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const barres = opts.series.map((s) => ({
    type: 'bar',
    name: s.name,
    ...(s.stack ? { stack: s.stack } : {}),
    barMaxWidth: 18,
    itemStyle: { borderRadius: [3, 3, 0, 0] },
    color: s.color,
    data: barData(s),
    ...(s.usual ? { markLine: markLineOf(s.usual, tc.axisLabel) } : {}),
  }))
  const tendances = opts.series
    .filter((s) => s.trend != null)
    .map((s) => {
      const trend = s.trend as SessionBarsTrend
      return {
        type: 'line',
        name: trend.label,
        data: trend.valuesPct.map((v) => (v == null ? null : round1(v))),
        smooth: false,
        lineStyle: { width: 2, color: trend.color },
        itemStyle: { color: trend.color },
        symbol: 'circle',
        symbolSize: 6,
      }
    })

  const xAxis: Record<string, unknown>[] = [{ ...axis, type: 'category', data: opts.labels }]
  if (opts.volumeAxis) {
    // Le rang des volumes : un axe de catégories SANS ligne ni graduation, posé sous le
    // premier. Il ne porte aucune série — seulement ses étiquettes.
    xAxis.push({
      type: 'category',
      data: opts.volumeAxis.values,
      position: 'bottom',
      offset: 22,
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
      name: opts.volumeAxis.label,
      nameLocation: 'end',
      nameGap: 8,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { color: tc.axisLabel, fontSize: 10 },
    })
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: opts.volumeAxis ? 64 : 44, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Le DÉNOMINATEUR rejoint chaque infobulle : un taux sans son volume se lit faux.
      formatter: (params: unknown) => {
        const rows = Array.isArray(params)
          ? (params as {
              axisValue: string
              dataIndex: number
              marker: string
              seriesName: string
              value: number | null
            }[])
          : []
        if (rows.length === 0) return ''
        const lignes = rows
          .filter((r) => r.value != null)
          .map((r) => `${r.marker}${r.seriesName} : ${r.value} %`)
        lignes.push(...(opts.tooltipLines?.(rows[0].dataIndex) ?? []))
        return [rows[0].axisValue, ...lignes].join('<br/>')
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: [...barres.map((b) => b.name), ...tendances.map((l) => l.name)],
    },
    xAxis,
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      name: opts.yAxisLabel,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value} %' },
    },
    series: [...barres, ...tendances],
  }
}
