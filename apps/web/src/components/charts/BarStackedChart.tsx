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

/** Distance (px) entre les étiquettes d'un axe et son titre, quand il en a un. */
const AXIS_NAME_GAP = 28

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
  /** Propage a ChartCard : le graphe est rendu nu, sans bordure ni fond (voir ChartCard.frameless). */
  frameless?: boolean
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
  /**
   * Note additionnelle affichée dans l'infobulle, à droite de la valeur d'un segment.
   * Reçoit la catégorie survolée et la sous-clé, retourne le texte (déjà localisé) ou
   * `undefined` pour ne rien ajouter.
   *
   * Sans cette prop l'infobulle est INCHANGÉE pour tous les appelants existants. Avec
   * elle, le formateur personnalisé s'active même si `tooltipHideZero` est faux.
   */
  tooltipComponentNote?: (category: string, component: string) => string | undefined
  /**
   * Titre de l'axe des catégories, posé au milieu sous les étiquettes. Nomme le RÔLE des
   * barres quand la catégorie seule ne le dit pas : sur les assistances d'escouade, la barre
   * est un gamertag et son rôle (« Larbin ») n'est écrit nulle part ailleurs.
   */
  categoryAxisName?: string
  /**
   * Titre de l'axe des valeurs (ce que la hauteur des segments compte), même position et
   * même repli que `categoryAxisName`.
   */
  valueAxisName?: string
  /**
   * Rôles nommés dans l'infobulle : `category` préfixe l'en-tête (la barre survolée),
   * `component` préfixe chaque segment (« Larbin · X » / « Patron · Y : 3 »). Active le
   * formateur personnalisé, comme `tooltipComponentNote`.
   */
  tooltipRoles?: { category: string; component: string }
}

export function BarStackedChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  frameless,
  orientation = 'vertical',
  componentColors,
  componentOrder,
  tooltipHideZero = false,
  componentHexColors,
  tooltipComponentNote,
  categoryAxisName,
  valueAxisName,
  tooltipRoles,
}: BarStackedChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointStacked>[]) =>
      buildBarStackedOption(s, {
        orientation,
        componentColors,
        componentOrder,
        tooltipHideZero,
        componentHexColors,
        tooltipComponentNote,
        categoryAxisName,
        valueAxisName,
        tooltipRoles,
      }),
    [
      orientation,
      componentColors,
      componentOrder,
      tooltipHideZero,
      componentHexColors,
      tooltipComponentNote,
      categoryAxisName,
      valueAxisName,
      tooltipRoles,
    ],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      frameless={frameless}
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
  tooltipComponentNote?: (category: string, component: string) => string | undefined
  categoryAxisName?: string
  valueAxisName?: string
  tooltipRoles?: { category: string; component: string }
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
    tooltipComponentNote,
    categoryAxisName,
    valueAxisName,
    tooltipRoles,
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
  // Titre d'axe AU MILIEU, à distance des graduations (même choix que
  // Heatmap2DChart.axisNameOpts) : la seule position qui ne chevauche ni la première ni la
  // dernière étiquette. Sans titre, l'axe reste exactement celui d'avant l'ajout de l'option.
  const axisName = (name: string | undefined) =>
    name
      ? {
          name,
          nameLocation: 'middle' as const,
          nameGap: AXIS_NAME_GAP,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        }
      : {}
  const valueAxis = { ...axis, type: 'value' as const, ...axisName(valueAxisName) }
  const categoryAxis = {
    ...axis,
    type: 'category' as const,
    data: categories,
    ...axisName(categoryAxisName),
  }
  const xAxis = orientation === 'horizontal' ? valueAxis : categoryAxis
  const yAxis = orientation === 'horizontal' ? categoryAxis : valueAxis

  const tooltipBase = {
    ...getTooltipBase(tc),
    trigger: 'axis' as const,
    axisPointer: { type: 'shadow' as const },
  }
  // Formateur personnalisé dès que l'appelant demande le masquage des zéros, une note
  // par segment OU des rôles nommés. Sans aucun des trois on laisse le formateur natif
  // d'ECharts — c'est le comportement de tous les appelants antérieurs.
  const tooltip =
    tooltipHideZero || tooltipComponentNote || tooltipRoles
      ? {
          ...tooltipBase,
          formatter: (raw: unknown) => {
            const params = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
            if (params.length === 0) return ''
            const header = params[0]?.axisValueLabel ?? String(params[0]?.axisValue ?? '')
            const lines = params
              .filter((p) => !tooltipHideZero || (typeof p.value === 'number' && p.value !== 0))
              .map((p) => {
                const name = p.seriesName ?? ''
                const note = tooltipComponentNote?.(header, name)
                const suffix = note ? ` <span style="opacity:0.75">${escapeHtml(note)}</span>` : ''
                // Le rôle précède le nom (« Patron · X ») : la note et le compte gardent leur
                // place, et `tooltipComponentNote` reçoit toujours le nom NU.
                const shown = tooltipRoles ? `${tooltipRoles.component} · ${name}` : name
                return `${p.marker ?? ''}${escapeHtml(shown)}: <strong>${p.value}</strong>${suffix}`
              })
            if (lines.length === 0) return ''
            const title = tooltipRoles ? `${tooltipRoles.category} · ${header}` : header
            return `<div style="margin-bottom:4px;font-weight:600">${escapeHtml(title)}</div>${lines.join('<br/>')}`
          },
        }
      : tooltipBase

  return {
    backgroundColor: CHART_BG,
    // `containLabel` réserve la place des étiquettes, PAS celle d'un titre d'axe : un axe
    // nommé agrandit sa marge (bas pour X, gauche pour Y), sinon le titre sort du cadre.
    grid: {
      top: 20,
      bottom: 'name' in xAxis ? 40 + AXIS_NAME_GAP : 40,
      left: 'name' in yAxis ? 8 + AXIS_NAME_GAP : 8,
      right: 16,
      containLabel: true,
    },
    tooltip,
    legend: { ...getLegendBase(tc), data: components },
    xAxis,
    yAxis,
    series: echartsSeries,
  }
}
