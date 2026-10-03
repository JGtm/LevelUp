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

/** Distance (px) entre les étiquettes d'un axe et son titre, quand il en a un. */
const AXIS_NAME_GAP = 28

/**
 * LE TITRE D'UN AXE SE MESURE DEPUIS LA LIGNE D'AXE, PAS DEPUIS SES ÉTIQUETTES — et
 * `grid.containLabel` ne réserve JAMAIS la place d'un titre (il ne connaît que les
 * étiquettes). Sous un axe horizontal, la bande d'étiquettes fait la hauteur d'une ligne
 * (~12 px + marge) : un titre à 28 px la dépasse, il s'affiche. À GAUCHE d'un axe de
 * CATÉGORIES vertical (barres horizontales), la même bande fait la LARGEUR du plus long
 * libellé — un gamertag, 60 à 90 px : le titre posé à 28 px tombait DANS les étiquettes,
 * superposé à elles, donc illisible, pendant que la marge gauche réservée pour lui restait
 * vide (constat utilisateur du 2026-09-22 sur « Appui », dont le titre d'axe « Larbin »
 * n'apparaissait pas alors que l'option ECharts le portait bien).
 *
 * ECharts ne sait pas mesurer ce texte au moment où l'on construit l'option : on ESTIME la
 * bande à partir du plus long libellé (police 10 px de `getAxisBase`) et l'on pose le titre
 * `AXIS_NAME_TO_LABEL_GAP` px plus à gauche. La largeur moyenne d'un caractère est prise AU
 * MILIEU de la fourchette réelle (~5,5 px tout en bas de casse, ~6,2 px en capitales) : la
 * sous-estimation est absorbée par la respiration, la sur-estimation par `grid.left`, et le
 * titre reste entre les deux dans les deux sens (±17 px de tolérance pour un libellé de
 * 15 caractères — la longueur maximale d'un gamertag Xbox).
 */
const AXIS_LABEL_CHAR_PX = 5.8
/** Marge par défaut d'`axisLabel` dans ECharts, entre la ligne d'axe et son étiquette. */
const AXIS_LABEL_MARGIN_PX = 8
/** Plafond de la bande estimée : au-delà, ECharts tronque rarement mais la marge suffit. */
const AXIS_LABEL_BAND_MAX_PX = 160
/** Respiration entre le bord gauche des étiquettes et le titre de l'axe. */
const AXIS_NAME_TO_LABEL_GAP = 14

/**
 * Largeur estimée de la bande d'étiquettes d'un axe de catégories VERTICAL (marge comprise).
 * Exportée pour le test : c'est elle qui fixe l'écart du titre d'axe.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function categoryLabelBandPx(categories: readonly string[]): number {
  const plusLong = categories.reduce((m, c) => Math.max(m, c.length), 0)
  return Math.min(AXIS_LABEL_BAND_MAX_PX, plusLong * AXIS_LABEL_CHAR_PX) + AXIS_LABEL_MARGIN_PX
}

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
  orientation = 'vertical',
  componentColors,
  componentOrder,
  tooltipHideZero = false,
  componentHexColors,
  tooltipComponentNote,
  categoryAxisName,
  valueAxisName,
  tooltipRoles,
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
        tooltipComponentNote,
        categoryAxisName,
        valueAxisName,
        tooltipRoles,
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
      tooltipComponentNote,
      categoryAxisName,
      valueAxisName,
      tooltipRoles,
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
            label: {
              show: true,
              position: 'inside' as const,
              color: tc.text,
              fontSize: 10,
              formatter: (p: { value?: number | null }) => (p.value ? String(p.value) : ''),
            },
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
  opts: Pick<BuildOpts, 'orientation' | 'categoryAxisName' | 'valueAxisName' | 'categoryNote'>,
) {
  const { orientation = 'vertical', categoryAxisName, valueAxisName, categoryNote } = opts
  const axis = getAxisBase(tc)
  // Titre d'axe AU MILIEU, à distance des graduations (même choix que
  // Heatmap2DChart.axisNameOpts) : la seule position qui ne chevauche ni la première ni la
  // dernière étiquette. Sans titre, l'axe reste exactement celui d'avant l'ajout de l'option.
  const axisName = (name: string | undefined, gap = AXIS_NAME_GAP) =>
    name
      ? {
          name,
          nameLocation: 'middle' as const,
          nameGap: gap,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        }
      : {}
  // L'axe des CATÉGORIES est vertical quand les barres sont horizontales : son titre doit
  // alors franchir toute la bande des libellés (voir AXIS_LABEL_CHAR_PX).
  const categoryNameGap =
    orientation === 'horizontal'
      ? categoryLabelBandPx(categories) + AXIS_NAME_TO_LABEL_GAP
      : AXIS_NAME_GAP
  const valueAxis = { ...axis, type: 'value' as const, ...axisName(valueAxisName) }
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
    ...axisName(categoryAxisName, categoryNameGap),
  }
  return orientation === 'horizontal'
    ? { xAxis: valueAxis, yAxis: categoryAxis }
    : { xAxis: categoryAxis, yAxis: valueAxis }
}

/**
 * L'infobulle d'une barre empilee. Formateur personnalise des que l'appelant demande le
 * masquage des zeros, une note par segment OU des roles nommes ; sans aucun des trois on laisse
 * le formateur natif d'ECharts — c'est le comportement de tous les appelants anterieurs.
 *
 * EXTRAIT DE `buildBarStackedOption` LE 2026-09-21 (lot I) : la fonction passait le plafond de
 * taille du depot en accueillant les groupes de colonnes.
 */
function buildStackedTooltip(
  tc: EChartsThemeColors,
  opts: Pick<BuildOpts, 'tooltipHideZero' | 'tooltipComponentNote' | 'tooltipRoles'>,
) {
  const { tooltipHideZero = false, tooltipComponentNote, tooltipRoles } = opts
  const tooltipBase = {
    ...getTooltipBase(tc),
    trigger: 'axis' as const,
    axisPointer: { type: 'shadow' as const },
  }
  // Formateur personnalisé dès que l'appelant demande le masquage des zéros, une note
  // par segment OU des rôles nommés. Sans aucun des trois on laisse le formateur natif
  // d'ECharts — c'est le comportement de tous les appelants antérieurs.
  return tooltipHideZero || tooltipComponentNote || tooltipRoles
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
    tooltipComponentNote,
    categoryAxisName,
    valueAxisName,
    tooltipRoles,
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

  const { xAxis, yAxis } = buildStackedAxes(tc, categories, {
    orientation,
    categoryAxisName,
    valueAxisName,
    categoryNote,
  })

  const tooltip = buildStackedTooltip(tc, {
    tooltipHideZero,
    tooltipComponentNote,
    tooltipRoles,
  })

  const groupTitles = categoryGroups ? groupTitleGraphic(categoryGroups, tc.axisLabel) : undefined

  return {
    backgroundColor: CHART_BG,
    // `containLabel` réserve la place des étiquettes, PAS celle d'un titre d'axe : un axe
    // nommé agrandit sa marge (bas pour X, gauche pour Y), sinon le titre sort du cadre.
    // Cette marge loge le TITRE SEUL — les étiquettes sont réservées en plus par
    // `containLabel`, et c'est `nameGap` qui fait franchir leur bande au titre.
    grid: {
      top: 20,
      bottom: 'name' in xAxis ? 40 + AXIS_NAME_GAP : 40,
      left: 'name' in yAxis ? 8 + AXIS_NAME_GAP : 8,
      right: 16,
      containLabel: true,
    },
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
