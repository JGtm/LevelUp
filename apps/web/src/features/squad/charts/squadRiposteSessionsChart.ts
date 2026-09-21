/**
 * squadRiposteSessionsChart — la FRISE de la riposte, soirée par soirée (carte « Riposte »
 * de la section Coordination, D19 du 2026-09-21).
 *
 * GRAMMAIRE REPRISE DE `squadSessionTimelineChart.ts`, que l'utilisateur a désignée comme
 * référence : bâtons à 18 px, courbe pleine 2 px NON lissée par-dessus, légende nommant
 * chaque série, axes en 10 px gris.
 *
 * TROIS ÉCARTS ASSUMÉS À LA RÉFÉRENCE :
 *
 *   1. UN SEUL AXE Y. La référence en a deux (perf et MMR) ; ici tout est en points de
 *      pourcentage — taux de la soirée, tendance, habituel —, donc un seul suffit, et deux
 *      seraient une faute de lecture.
 *   2. LA COULEUR DU BÂTON PORTE UN VERDICT, jamais l'identité d'une soirée : au-dessus de
 *      l'habituel (`success`) ou en dessous (`warning`). Les deux verdicts sont DEUX SÉRIES
 *      EMPILÉES à trous — c'est ce qui leur donne une entrée de légende chacun, là qu'une
 *      colorisation par `itemStyle` n'aurait nommée nulle part.
 *   3. LE VOLUME DE CHAQUE SOIRÉE (morts mesurées) est un SECOND RANG D'ÉTIQUETTES sous
 *      l'axe des dates, pas un second graphe : c'est le dénominateur, il n'a pas d'échelle
 *      propre. La courbe que cette frise remplace n'en montrait aucun — une soirée à
 *      3 morts mesurées s'y lisait comme une soirée à 80.
 *
 * Le repère de l'habituel est une `markLine` TIRETÉE posée sur la première série : une
 * valeur de comparaison n'est pas une mesure de la période, elle ne peut pas être un bâton.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  seriesColor,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { FriseRiposte } from '../squadRiposte.logic'

export interface SquadRiposteSessionsOpts {
  aboveLabel: string
  belowLabel: string
  trendLabel: string
  /** Libellé du repère d'habituel, déjà formaté avec son taux. */
  usualLabel: string
  yAxisLabel: string
  /** Rang d'étiquettes sous l'axe : son nom (« morts mesurées »). */
  volumeAxisLabel: string
  /** Infobulle du volume d'une soirée, déjà pluralisée. */
  volumeTooltip: (n: number) => string
}

function round1(v: number): number {
  return parseFloat(v.toFixed(1))
}

/**
 * buildSquadRiposteSessionsOption — l'option ECharts complète de la frise.
 *
 * La série porte UN SEUL datapoint : la frise entière (`FriseRiposte`). C'est la forme
 * qu'attend `ChartCard`, dont le contrat est « une liste de séries de points ».
 */
export function buildSquadRiposteSessionsOption(
  series: ChartSeries<FriseRiposte>[],
  opts: SquadRiposteSessionsOpts,
): EChartsCoreOption {
  const frise = series[0]?.datapoints[0]
  if (!frise || frise.soirees.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const above = resolveToken('success')
  const below = resolveToken('warning')

  const labels = frise.soirees.map((s) => s.label)
  const volumes = frise.soirees.map((s) => String(s.morts))
  // Deux séries à TROUS, empilées : chaque soirée n'alimente que celle de son verdict.
  const dataAbove = frise.soirees.map((s) => (s.auDessus ? round1(s.tauxPct) : null))
  const dataBelow = frise.soirees.map((s) => (s.auDessus ? null : round1(s.tauxPct)))
  const dataTrend = frise.tendancePct.map(round1)
  const parLabel = new Map(frise.soirees.map((s) => [s.label, s.morts]))

  const barBase = {
    type: 'bar',
    stack: 'soiree',
    barMaxWidth: 18,
    itemStyle: { borderRadius: [3, 3, 0, 0] },
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 36, bottom: 64, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Le DÉNOMINATEUR rejoint chaque infobulle : un taux sans son volume se lit faux.
      formatter: (params: unknown) => {
        const rows = Array.isArray(params) ? (params as { axisValue: string; marker: string; seriesName: string; value: number | null }[]) : []
        if (rows.length === 0) return ''
        const titre = rows[0].axisValue
        const lignes = rows
          .filter((r) => r.value != null)
          .map((r) => `${r.marker}${r.seriesName} : ${r.value} %`)
        lignes.push(opts.volumeTooltip(parLabel.get(titre) ?? 0))
        return [titre, ...lignes].join('<br/>')
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: [opts.aboveLabel, opts.belowLabel, opts.trendLabel],
    },
    xAxis: [
      { ...axis, type: 'category', data: labels },
      {
        // Le rang des volumes : un axe de catégories SANS ligne ni graduation, posé sous
        // le premier. Il ne porte aucune série — seulement ses étiquettes.
        type: 'category',
        data: volumes,
        position: 'bottom',
        offset: 22,
        axisLine: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
        name: opts.volumeAxisLabel,
        nameLocation: 'end',
        nameGap: 8,
        nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        axisLabel: { color: tc.axisLabel, fontSize: 10 },
      },
    ],
    yAxis: {
      ...axis,
      type: 'value',
      min: 0,
      name: opts.yAxisLabel,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, formatter: '{value} %' },
    },
    series: [
      {
        ...barBase,
        name: opts.aboveLabel,
        data: dataAbove,
        color: above,
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { type: 'dashed', color: tc.axisLabel, width: 1 },
          label: { formatter: opts.usualLabel, color: tc.axisLabel, fontSize: 10, position: 'insideEndTop' },
          data: [{ yAxis: round1(frise.habituelPct) }],
        },
      },
      { ...barBase, name: opts.belowLabel, data: dataBelow, color: below },
      {
        name: opts.trendLabel,
        type: 'line',
        data: dataTrend,
        smooth: false,
        lineStyle: { width: 2, color: seriesColor(2) },
        itemStyle: { color: seriesColor(2) },
        symbol: 'circle',
        symbolSize: 6,
      },
    ],
  }
}
