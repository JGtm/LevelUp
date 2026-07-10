/**
 * BarStackedChart — wrapper ECharts pour `ChartSeries<ChartPointStacked>`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ category, components }`.
 *   - Les `components` sont les sous-clés empilées (ex. win/loss/tie/dnf).
 *
 * Cas d'usage Squad V2 :
 *   - S5 Impact ranking par rôle (vertical, components = nb par joueur)
 *   - S6 Cadence (vertical, components = kills par phase)
 *   - S7 HS+PK (vertical, components = headshots + power_weapons)
 *
 * Variants :
 *   - `orientation` : 'vertical' (default) ou 'horizontal' (lollipop-like)
 *   - `componentColors` : map sous-clé → SemanticToken pour palette stable
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  seriesColor,
} from './_utils'

export interface ChartPointStacked {
  category: string
  components: Record<string, number>
}

export interface BarStackedChartProps {
  title?: string
  series: ChartSeries<ChartPointStacked>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Vertical (default) ou horizontal (categories sur Y). */
  orientation?: 'vertical' | 'horizontal'
  /**
   * Map sous-clé component → SemanticToken pour coloration cohérente.
   * Ex: { win: 'outcome-win', loss: 'outcome-loss' }.
   */
  componentColors?: Record<string, SemanticToken>
  /** Tableau des sous-clés à inclure dans l'ordre voulu (default: collecte auto). */
  componentOrder?: string[]
  /**
   * Si true, le tooltip filtre les composants à 0 pour la catégorie survolée
   * (utile sur les bars empilées éparses où la majorité des sous-clés sont 0).
   * Default false.
   */
  tooltipHideZero?: boolean
  /**
   * Override hex couleur direct par sous-clé (priorité absolue : > componentColors > cycle).
   * Permet d'éviter la résolution tardive via tokens quand l'appelant a déjà
   * résolu les couleurs (sinon `resolveToken` peut retourner '' et ECharts
   * applique sa palette interne — premières couleurs en bleu).
   */
  componentHexColors?: Record<string, string>
}

export function BarStackedChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  orientation = 'vertical',
  componentColors,
  componentOrder,
  tooltipHideZero = false,
  componentHexColors,
}: BarStackedChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointStacked>[]) =>
      buildBarStackedOption(s, {
        orientation,
        componentColors,
        componentOrder,
        tooltipHideZero,
        componentHexColors,
      }),
    [orientation, componentColors, componentOrder, tooltipHideZero, componentHexColors],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
    />
  )
}

interface BuildOpts {
  orientation?: 'vertical' | 'horizontal'
  componentColors?: Record<string, SemanticToken>
  componentOrder?: string[]
  tooltipHideZero?: boolean
  componentHexColors?: Record<string, string>
}

interface TooltipParam {
  seriesName?: string
  value?: number | null
  color?: string
  marker?: string
  axisValueLabel?: string
  axisValue?: string | number
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildBarStackedOption(
  series: ChartSeries<ChartPointStacked>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const {
    orientation = 'vertical',
    componentColors,
    componentOrder,
    tooltipHideZero = false,
    componentHexColors,
  } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  // 1 série attendue (le wrapper agit sur la première).
  const main = series[0]
  const dps = main.datapoints

  const categories = dps.map((d) => d.category)

  // Collecter l'ordre des composants (preserve l'ordre componentOrder si fourni).
  const componentSet = new Set<string>()
  for (const dp of dps) {
    for (const k of Object.keys(dp.components)) componentSet.add(k)
  }
  const components = componentOrder
    ? componentOrder.filter((c) => componentSet.has(c))
    : Array.from(componentSet)

  // 1 ECharts series par component (toutes empilées sur le même stack).
  // Priorité de résolution couleur :
  //  1. componentHexColors[comp] (hex pré-résolu — option la plus sûre)
  //  2. componentColors[comp] résolu via resolveToken (avec fallback vers
  //     seriesColor si la CSS var n'est pas chargée — sinon ECharts utilise
  //     son palette interne qui commence par du bleu).
  //  3. seriesColor(idx) — palette chart-series cyclée.
  const echartsSeries = components.map((comp, idx) => {
    const explicitHex = componentHexColors?.[comp]
    let color: string
    if (explicitHex && explicitHex.length > 0) {
      color = explicitHex
    } else if (componentColors?.[comp]) {
      color = resolveToken(componentColors[comp]) || seriesColor(idx)
    } else {
      color = seriesColor(idx)
    }
    return {
      name: comp,
      type: 'bar' as const,
      stack: 'total',
      barMaxWidth: 24,
      itemStyle: { color, borderRadius: 2 },
      data: dps.map((d) => d.components[comp] ?? 0),
    }
  })

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const valueAxis = { ...axis, type: 'value' as const }
  const categoryAxis = {
    ...axis,
    type: 'category' as const,
    data: categories,
  }

  const tooltipBase = {
    ...getTooltipBase(tc),
    trigger: 'axis' as const,
    axisPointer: { type: 'shadow' as const },
  }
  const tooltip = tooltipHideZero
    ? {
        ...tooltipBase,
        formatter: (raw: unknown) => {
          const params = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
          if (params.length === 0) return ''
          const header = params[0]?.axisValueLabel ?? String(params[0]?.axisValue ?? '')
          const lines = params
            .filter((p) => typeof p.value === 'number' && p.value !== 0)
            .map(
              (p) =>
                `${p.marker ?? ''}${escapeHtml(p.seriesName ?? '')}: <strong>${p.value}</strong>`,
            )
          if (lines.length === 0) return ''
          return `<div style="margin-bottom:4px;font-weight:600">${escapeHtml(header)}</div>${lines.join('<br/>')}`
        },
      }
    : tooltipBase

  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 40, left: 8, right: 16, containLabel: true },
    tooltip,
    legend: { ...getLegendBase(tc), data: components },
    xAxis: orientation === 'horizontal' ? valueAxis : categoryAxis,
    yAxis: orientation === 'horizontal' ? categoryAxis : valueAxis,
    series: echartsSeries,
  }
}
