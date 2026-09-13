/**
 * HistogramChart — wrapper ECharts pour un histogramme depuis
 * `ChartSeries<ChartPointHistogram>[]`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ binStart, binEnd, count }`.
 *   - Les buckets sont rendus comme des barres adjacentes (catégories).
 *
 * Cas d'usage Phase 3 :
 *   - Distributions Timeseries : K/D, kills/match, précision, score/min,
 *     win rate glissant.
 *
 * Le ChartCard parent gère les états loading/error/empty.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken, type SemanticToken } from '@/lib/accessibility'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase, seriesColor } from './_utils'

export interface ChartPointHistogram {
  binStart: number
  binEnd: number
  count: number
}

export interface HistogramChartProps {
  title?: string
  series: ChartSeries<ChartPointHistogram>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Token couleur pour la barre (default chart-series-1). */
  colorToken?: SemanticToken
  /** Libellé de l'axe X (ex. "K/D", "Kills / match"). */
  xAxisLabel?: string
  /** Libellé de l'axe Y (default = nb de matchs en FR). */
  yAxisLabel?: string
  /**
   * Format des bornes de bucket. Default : "binStart–binEnd" arrondi à 2
   * décimales si non-entier.
   */
  formatBin?: (point: ChartPointHistogram) => string
  /**
   * Barres HORS PÉRIMÈTRE : montrées, mais exclues de ce que le graphe compte (ajout
   * 2026-09-06 pour la distribution du délai d'échange de l'escouade ; HACHURE rétablie
   * le 2026-09-13, décision utilisateur et maquette 4c520da6).
   *
   * DEUX INDICES, ET LES DEUX SE VOIENT. La barre reçoit un `decal` ECharts — la hachure
   * diagonale du dépôt (même motif que `Heatmap2DChart.emptyCellItemStyle`), pas un liseré
   * tireté de la couleur du remplissage : l'essai du 2026-09-06 avait été retiré parce
   * qu'il était INVISIBLE, ce qui est un défaut de réalisation, pas de l'idée. La couleur
   * de série reste, atténuée — une seconde teinte inventerait une seconde signification,
   * et aucun jeton du dépôt n'est achromatique dans les quatre palettes.
   *
   * Le TROISIÈME indice n'est pas graphique : c'est le mot, porté par l'étiquette d'axe et
   * le pied de carte de l'appelant.
   *
   * POURQUOI PAS DEUX SÉRIES : ce wrapper ne peint que `series[0]` (une seconde série
   * serait ignorée EN SILENCE), et deux séries sur les mêmes catégories décaleraient
   * les barres.
   */
  binHatched?: (point: ChartPointHistogram, index: number) => boolean
  /**
   * Valeur écrite AU-DESSUS de chaque barre (maquette 4c520da6). Absent = aucune
   * étiquette : le comportement de tous les appelants antérieurs.
   */
  showValues?: boolean
  /**
   * Repère vertical tireté posé sur la FRONTIÈRE qui précède l'intervalle `binIndex` — la
   * borne d'une fenêtre. Porte `label` et prend le jeton `warning`.
   *
   * Un seuil qui découpe la population DOIT se voir : sans lui, le lecteur ne sait pas où
   * la fenêtre tombe et additionne des barres qui n'entrent dans aucun taux.
   */
  windowMark?: { binIndex: number; label: string }
  /**
   * SEUILS TRACÉS EN POINTILLÉ sur l'axe des catégories (ajout 2026-09-13, distribution
   * des distances à l'équipier de l'onglet Tactique).
   *
   * `at` est une position en INDICE DE CATÉGORIE, fractionnaire : un seuil de 18 m sur
   * des intervalles de 10 m tombe à 1,8 — entre la deuxième et la troisième barre, là où
   * il est vraiment. L'arrondir à une frontière de barre déplacerait la règle du jeu.
   *
   * PLUSIEURS SEUILS SONT LE CAS NORMAL : un filtre qui mélange Arène (18 m) et BTB
   * (24 m) mélange deux règles du jeu, et leur moyenne n'est la règle d'aucun match.
   */
  thresholds?: readonly { at: number; label: string }[]
}

export function HistogramChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  colorToken,
  xAxisLabel,
  yAxisLabel,
  formatBin,
  binHatched,
  showValues,
  windowMark,
  thresholds,
}: HistogramChartProps) {
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHistogram>[]) =>
      buildHistogramOption(s, {
        colorToken,
        xAxisLabel,
        yAxisLabel,
        formatBin,
        binHatched,
        showValues,
        windowMark,
        thresholds,
      }),
    [colorToken, xAxisLabel, yAxisLabel, formatBin, binHatched, showValues, windowMark, thresholds],
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
  colorToken?: SemanticToken
  xAxisLabel?: string
  yAxisLabel?: string
  formatBin?: (point: ChartPointHistogram) => string
  binHatched?: (point: ChartPointHistogram, index: number) => boolean
  showValues?: boolean
  windowMark?: { binIndex: number; label: string }
  thresholds?: readonly { at: number; label: string }[]
}

/**
 * Opacité d'une barre hors périmètre. Assez basse pour se distinguer d'un coup d'œil d'une
 * barre pleine, assez haute pour rester lisible sur les deux thèmes. La HACHURE est le
 * second indice : ensemble ils se voient, séparément non (mesuré le 2026-09-06).
 */
const ATTENUATION_OPACITE = 0.35

/**
 * Motif de hachure INVISIBLE, pose au niveau de la SERIE.
 *
 * `aria.decal.show` est requis pour qu'ECharts applique les `decal` poses a la main — mais
 * il hachure AUSSI, automatiquement, toute serie qui n'en declare pas. Sans ce motif
 * neutre, activer la hachure des barres hors fenetre hachurait les cinq autres du meme
 * coup (mesure sur capture le 2026-09-13). Un motif transparent ne peint rien et occupe la
 * place que la hachure automatique aurait prise ; les barres hors fenetre, elles, posent
 * leur propre motif PAR BARRE, qui prend le dessus.
 */
const DECAL_NEUTRE = { color: 'transparent' }

/**
 * Hachure diagonale d'une barre hors périmètre — même `decal` que la case vide de
 * `Heatmap2DChart` : un seul motif de hachure dans toute l'application.
 */
function hachureBarre(color: string) {
  return {
    symbol: 'rect',
    symbolSize: 0.8,
    dashArrayX: [1, 0],
    dashArrayY: [4, 4],
    rotation: -Math.PI / 4,
    color,
  }
}

function defaultFormatBin(point: ChartPointHistogram): string {
  const fmt = (n: number): string => (Number.isInteger(n) ? String(n) : n.toFixed(2))
  return `${fmt(point.binStart)}–${fmt(point.binEnd)}`
}

/**
 * Pure builder — exporté pour tester l'option ECharts sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildHistogramOption(
  series: ChartSeries<ChartPointHistogram>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { colorToken, xAxisLabel, yAxisLabel: yLabelOpt, formatBin = defaultFormatBin } = opts
  const { binHatched, showValues, windowMark, thresholds } = opts
  const yAxisLabel = yLabelOpt ?? 'Matchs'
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints
  if (dps.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const categories = dps.map((d) => formatBin(d))
  const color = colorToken ? resolveToken(colorToken) : seriesColor(0)
  // Une barre ne porte un style propre QUE si l'appelant la déclare hors périmètre : sans
  // `binHatched`, chaque valeur reste un nombre nu et ECharts applique la couleur de
  // série — le comportement historique, bit pour bit.
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  const counts = dps.map((d, i) =>
    binHatched?.(d, i)
      ? {
          value: d.count,
          itemStyle: {
            color,
            opacity: ATTENUATION_OPACITE,
            decal: hachureBarre(tc.axisLabel),
          },
        }
      : d.count,
  )

  // Repère de fenêtre : markLine verticale tiretée posée sur la FRONTIÈRE entre deux
  // catégories (`binIndex - 0.5`), jamais au centre d'une barre — une fenêtre est une
  // borne, pas un intervalle.
  const warningColor = resolveToken('warning')
  const markLine =
    windowMark && windowMark.binIndex > 0 && windowMark.binIndex < dps.length
      ? {
          silent: true,
          symbol: 'none' as const,
          lineStyle: { type: 'dashed' as const, color: warningColor, width: 1.5 },
          label: {
            show: true,
            formatter: windowMark.label,
            // `rotate: 0` : sans lui l'etiquette suit l'inclinaison de la ligne, donc
            // s'ecrit VERTICALEMENT sur un repere vertical (mesure sur capture).
            position: 'end' as const,
            rotate: 0,
            distance: 6,
            color: warningColor,
            fontSize: 10,
            fontWeight: 600,
          },
          data: [{ xAxis: windowMark.binIndex - 0.5 }],
        }
      : undefined

  return {
    backgroundColor: CHART_BG,
    // Requis pour qu'ECharts applique les `itemStyle.decal` posés à la main ci-dessus
    // (même flag que `Heatmap2DChart` et `MatchSummaryCharts.ARIA_DECAL`).
    aria: { decal: { show: true } },
    grid: { top: 28, bottom: 56, left: 48, right: 12 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: categories,
      name: xAxisLabel,
      nameLocation: 'middle',
      nameGap: 36,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
      axisLabel: { ...axis.axisLabel, rotate: -30 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      name: yAxisLabel,
      nameLocation: 'middle',
      nameGap: 32,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'bar',
        data: counts,
        barCategoryGap: '10%',
        // `decal: 'none'` EXPLICITE : `aria.decal.show` demande a ECharts de HACHURER
        // TOUTE serie qui ne declare pas son motif. Sans ce `none`, activer la hachure des
        // barres hors fenetre hachurait les cinq autres du meme coup (mesure sur capture le
        // 2026-09-13). Le motif reste pose PAR BARRE, dans `counts`.
        itemStyle: { color, borderRadius: 2, decal: DECAL_NEUTRE },
        label: showValues
          ? { show: true, position: 'top', color: tc.text, fontSize: 10, fontWeight: 500 }
          : { show: false },
        // Deux origines de reperes sur la meme serie (fusion des lots D1 et F, 2026-09-13) :
        // la borne de fenetre (`windowMark`) et les seuils fractionnaires (`thresholds`).
        // ECharts n'accepte qu'un `markLine` par serie : leurs donnees sont reunies.
        ...(markLine || (thresholds && thresholds.length > 0)
          ? { markLine: reunirMarkLines(markLine, thresholds ? markLineSeuils(thresholds, tc) : undefined) }
          : {}),
      },
    ],
  }
}

/** Les seuils en pointillé — jeton `warning`, étiquette en haut de la ligne. */
/**
 * Reunit deux markLine (fenetre + seuils) en un seul : le style du premier present sert de
 * socle, les donnees des deux sont concatenees. Chaque entree de seuil porte deja son
 * `name` ; la borne de fenetre porte son texte par `label.formatter`, conserve.
 */
function reunirMarkLines(
  a: Record<string, unknown> | undefined,
  b: Record<string, unknown> | undefined,
): Record<string, unknown> {
  if (!a) return b ?? {}
  if (!b) return a
  const da = Array.isArray(a.data) ? (a.data as unknown[]) : []
  const db = Array.isArray(b.data) ? (b.data as unknown[]) : []
  const labelA = (a.label ?? {}) as Record<string, unknown>
  // Le formatter de la fenetre est une chaine fixe : il ecraserait le nom de chaque seuil.
  // Un formatter par entree rend le nom quand il existe, sinon le texte de la fenetre.
  const texteFenetre = typeof labelA.formatter === 'string' ? labelA.formatter : ''
  return {
    ...a,
    label: { ...labelA, formatter: (pt: { name?: string }) => pt.name ?? texteFenetre },
    data: [...da, ...db],
  }
}

function markLineSeuils(
  thresholds: readonly { at: number; label: string }[],
  tc: ReturnType<typeof getEChartsThemeColors>,
): Record<string, unknown> {
  return {
    silent: true,
    symbol: 'none',
    lineStyle: { color: resolveToken('warning'), type: 'dashed', width: 1.5 },
    label: {
      show: true,
      position: 'insideEndTop',
      color: resolveToken('warning'),
      fontSize: 10,
      backgroundColor: tc.tooltipBg,
      padding: [1, 3],
      formatter: (p: { name?: string }) => p.name ?? '',
    },
    data: thresholds.map((t) => ({ name: t.label, xAxis: t.at })),
  }
}
