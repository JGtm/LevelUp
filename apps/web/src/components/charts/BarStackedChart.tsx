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

import { groupSeparatorMarkLine, groupTitleGraphic, type CategoryGroup } from './barStackedGroups'
import { barLabel } from './barLabel'
import { ChartCard, type ChartSeries } from './ChartCard'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
  seriesColor,
  type EChartsThemeColors,
} from './_utils'
import { tipNumberAuto } from './tipNumber'

import type { components } from '@/lib/api/generated'
export type ChartPointStacked = components['schemas']['ChartPointStacked']

export interface BarStackedChartProps {
  title?: string
  series: ChartSeries<ChartPointStacked>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Propage a ChartCard : le graphe est rendu nu, sans bordure ni fond (voir ChartCard.frameless). */
  frameless?: boolean
  /** Propage a ChartCard : le graphe remplit la cellule que sa voisine etire (voir ChartCard.fluid). */
  fluid?: boolean
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
   * Légende ECharts (au-dessus du graphe). `false` la retire : l'appelant en pose une en DOM
   * sous le graphe (`ChartLegend`), et deux légendes pour les mêmes séries se contrediraient
   * dès qu'un réglage d'accessibilité change les encres. Défaut `true` — le rendu de tous les
   * appelants antérieurs.
   */
  showLegend?: boolean
  /**
   * GROUPES DE COLONNES (vertical seulement) : un trait vertical discret à chaque frontière et
   * le titre de chaque groupe centré en haut du graphe. Voir `barStackedGroups.ts` pour les
   * deux mécanismes et leurs limites. Somme des `span` = nombre de catégories.
   */
  categoryGroups?: readonly CategoryGroup[]
  /**
   * ÉTIQUETTES DE VALEUR. `segments` écrit le compte DANS chaque segment (les étiquettes qui
   * ne tiennent pas sont retirées par le moteur, jamais rognées) ; `totals` écrit le total de
   * chaque colonne à son SOMMET — une valeur par catégorie, dans l'ordre des catégories.
   */
  valueLabels?: { segments?: boolean; totals?: readonly number[] }
  /**
   * Seconde ligne sous l'étiquette d'une catégorie (« + N sans nom »). Rendue en gris, elle
   * reste hors de la catégorie elle-même : l'infobulle garde le nom nu.
   */
  categoryNote?: (category: string) => string | undefined
  /**
   * Opacité par sous-clé — la façon canvas de distinguer deux joueurs d'un MÊME camp sans
   * introduire une couleur qui ne serait pas celle du camp. Défaut : opaque.
   */
  componentOpacity?: Record<string, number>
}

export function BarStackedChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  frameless,
  fluid,
  orientation = 'vertical',
  componentColors,
  componentOrder,
  tooltipHideZero = false,
  componentHexColors,
  showLegend,
  categoryGroups,
  valueLabels,
  categoryNote,
  componentOpacity,
}: BarStackedChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointStacked>[]) =>
      buildBarStackedOption(s, {
        orientation,
        componentColors,
        componentOrder,
        tooltipHideZero,
        componentHexColors,
        showLegend,
        categoryGroups,
        valueLabels,
        categoryNote,
        componentOpacity,
      }),
    [
      orientation,
      componentColors,
      componentOrder,
      tooltipHideZero,
      componentHexColors,
      showLegend,
      categoryGroups,
      valueLabels,
      categoryNote,
      componentOpacity,
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
      fluid={fluid}
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
  showLegend?: boolean
  categoryGroups?: readonly CategoryGroup[]
  valueLabels?: { segments?: boolean; totals?: readonly number[] }
  categoryNote?: (category: string) => string | undefined
  componentOpacity?: Record<string, number>
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
 * Les series empilees : une par sous-cle, plus la serie muette des totaux quand l'appelant en
 * demande. EXTRAIT DE `buildBarStackedOption` LE 2026-09-21 (lot I), meme raison que les axes.
 */
function buildStackedSeries(
  dps: ChartPointStacked[],
  components: string[],
  tc: EChartsThemeColors,
  opts: BuildOpts,
) {
  const { componentColors, componentHexColors, componentOpacity, valueLabels, categoryGroups } = opts
  // 1 ECharts series par component (toutes empilées sur le même stack).
  // Priorité de résolution couleur :
  //  1. componentHexColors[comp] (hex pré-résolu — option la plus sûre)
  //  2. componentColors[comp] résolu via resolveToken (avec fallback vers
  //     seriesColor si la CSS var n'est pas chargée — sinon ECharts utilise
  //     son palette interne qui commence par du bleu).
  //  3. seriesColor(idx) — palette chart-series cyclée.
  const separator = categoryGroups ? groupSeparatorMarkLine(categoryGroups, tc.axisLine) : undefined
  const totalsSeries = buildTotalsSeries(valueLabels?.totals, tc.text)

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
      itemStyle: { color, borderRadius: 2, opacity: componentOpacity?.[comp] ?? 1 },
      // Le compte DANS le segment, et retire par le moteur quand il n'y tient pas
      // (`hideOverlap`) : une etiquette rognee ment, une etiquette absente se lit dans
      // l'infobulle. Les zeros ne s'ecrivent jamais.
      ...(valueLabels?.segments
        ? {
            label: barLabel({ fontSize: 10, formatter: (p: { value?: number | null }) => (p.value ? tipNumberAuto(p.value) : '') }),
            labelLayout: { hideOverlap: true },
          }
        : {}),
      // Le trait des frontieres de groupe se pose sur UNE serie (la premiere) : une
      // `markLine` par serie empilee dessinerait le meme trait N fois.
      ...(idx === 0 && separator ? { markLine: separator } : {}),
      data: dps.map((d) => d.components[comp] ?? 0),
    }
  })
  if (totalsSeries) echartsSeries.push(totalsSeries as unknown as (typeof echartsSeries)[number])
  return echartsSeries
}

/**
 * Les deux axes d'une barre empilee : celui des categories (avec sa note de seconde ligne, si
 * l'appelant en pose une) et celui des valeurs, ranges selon l'orientation.
 *
 * EXTRAIT DE `buildBarStackedOption` LE 2026-09-21 (lot I), meme raison que l'infobulle.
 */
function buildStackedAxes(
  tc: EChartsThemeColors,
  categories: string[],
  opts: Pick<BuildOpts, 'orientation' | 'categoryNote'>,
) {
  const { orientation = 'vertical', categoryNote } = opts
  const axis = getAxisBase(tc)
  const valueAxis = { ...axis, type: 'value' as const }
  const categoryAxis = {
    ...axis,
    type: 'category' as const,
    data: categories,
    ...(categoryNote
      ? {
          axisLabel: {
            ...axis.axisLabel,
            interval: 0,
            // La note est une SECONDE LIGNE, en gris : elle dit ce que la colonne ne montre
            // pas (« + N sans nom ») sans entrer dans la categorie, donc sans polluer
            // l'infobulle ni l'identite de la barre.
            formatter: (value: string) => {
              const note = categoryNote(value)
              return note ? [value, `{note|${note}}`].join('\n') : value
            },
            rich: { note: { color: tc.axisLabel, fontSize: 9, padding: [2, 0, 0, 0] } },
          },
        }
      : {}),
  }
  return orientation === 'horizontal'
    ? { xAxis: valueAxis, yAxis: categoryAxis }
    : { xAxis: categoryAxis, yAxis: valueAxis }
}

/**
 * L'infobulle d'une barre empilee. Formateur personnalise des que l'appelant demande le
 * masquage des zeros ; sinon on laisse le formateur natif d'ECharts — c'est le comportement de
 * tous les appelants anterieurs.
 *
 * EXTRAIT DE `buildBarStackedOption` LE 2026-09-21 (lot I) : la fonction passait le plafond de
 * taille du depot en accueillant les groupes de colonnes.
 */
function buildStackedTooltip(tc: EChartsThemeColors, opts: Pick<BuildOpts, 'tooltipHideZero'>) {
  const { tooltipHideZero = false } = opts
  const tooltipBase = {
    ...getTooltipBase(tc),
    trigger: 'axis' as const,
    axisPointer: { type: 'shadow' as const },
  }
  return tooltipHideZero
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
                `${p.marker ?? ''}${escapeHtml(p.seriesName ?? '')}: <strong>${tipNumberAuto(p.value as number)}</strong>`,
            )
          if (lines.length === 0) return ''
          return `<div style="margin-bottom:4px;font-weight:600">${escapeHtml(header)}</div>${lines.join('<br/>')}`
        },
      }
    : tooltipBase
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
    componentOrder,
    tooltipHideZero = false,
    showLegend = true,
    categoryGroups,
    categoryNote,
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

  const tc = getEChartsThemeColors()
  const echartsSeries = buildStackedSeries(dps, components, tc, opts)

  const { xAxis, yAxis } = buildStackedAxes(tc, categories, { orientation, categoryNote })

  const tooltip = buildStackedTooltip(tc, { tooltipHideZero })

  const groupTitles = categoryGroups ? groupTitleGraphic(categoryGroups, tc.axisLabel) : undefined

  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 40, left: 8, right: 16, containLabel: true },
    tooltip,
    legend: showLegend ? { ...getLegendBase(tc), data: components } : { show: false },
    ...(groupTitles ? { graphic: groupTitles } : {}),
    xAxis,
    yAxis,
    series: echartsSeries,
  }
}

/**
 * La serie des TOTAUX : une barre de hauteur nulle empilee au sommet, dont la seule raison
 * d'etre est son etiquette. C'est le seul moyen, sur une pile ECharts, d'ecrire la somme de la
 * colonne a son sommet — aucune serie ne la connait, et un `graphic` ne saurait pas ou la poser.
 * Muette au survol (valeur 0, filtree par `tooltipHideZero`).
 */
function buildTotalsSeries(totals: readonly number[] | undefined, color: string) {
  if (!totals || totals.length === 0) return undefined
  return {
    name: TOTALS_SERIES_NAME,
    type: 'bar' as const,
    stack: 'total',
    silent: true,
    itemStyle: { color: 'transparent', opacity: 0 },
    label: {
      show: true,
      position: 'top' as const,
      color,
      fontSize: 10,
      fontWeight: 600 as const,
      formatter: (p: { dataIndex: number }) => String(totals[p.dataIndex] ?? ''),
    },
    data: totals.map(() => 0),
  }
}

/** Nom reserve de la serie des totaux — jamais une sous-cle de donnees. */
export const TOTALS_SERIES_NAME = '__totals__'
