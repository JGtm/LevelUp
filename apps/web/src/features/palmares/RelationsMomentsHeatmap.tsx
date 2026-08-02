/**
 * RelationsMomentsHeatmap — heatmap agrégé « Quand tu les croises » (Phase 3a).
 *
 * Générique : relation × bucket (heure 0..23 OU jour de semaine selon le
 * toggle du parent, #8). La couleur reflète le `count` (matchs communs) via la
 * rampe NEUTRE de fréquence heatmap-freq-low → heatmap-freq-high (mono-teinte,
 * sans connotation bien/mal — c'est de l'intensité de rencontre). Axe X =
 * buckets (labels fournis), axe Y = top-N relations. Strings via palmares.toml
 * (FR/EN).
 */
import { useCallback, useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'

/** Cellule générique : bucket = index de tranche horaire OU de jour de semaine. */
export interface HeatmapBucketCell {
  xuid: string
  gamertag: string
  bucket: number
  count: number
}

/**
 * Libellés du tooltip (une cellule = un joueur × un créneau). Fournis par le
 * parent depuis le manifeste palmares.toml — parité FR/EN garantie par le
 * générateur de manifestes, qui refuse une clé sans ses deux langues.
 * `bucketTypeLabel` suit le toggle : « Créneau » en mode heure, « Jour » en
 * mode jour de semaine.
 */
export interface HeatmapTooltipText {
  playerLabel: string
  bucketTypeLabel: string
  matchesLabel: string
  emptyCell: string
}

interface Props {
  cells: HeatmapBucketCell[]
  bucketLabels: string[] // index = bucket (tranche horaire ou jour)
  title?: string
  legendLabel: string
  emptyMessage: string
  tooltipText: HeatmapTooltipText
  height?: number
}

// Cap d'affichage : au plus MAX_HEATMAP_ROWS relations (les plus actives) sur
// l'axe Y — au-delà, la heatmap devient illisible (choix produit 2026-07-18).
const MAX_HEATMAP_ROWS = 12

/**
 * formatHeatmapTooltip — contenu du tooltip d'une cellule, en trois lignes
 * étiquetées (joueur / créneau ou jour / matchs communs) au lieu de l'ancien
 * « Joueur · 14h » suivi d'un compteur sans contexte. Le gamertag est échappé
 * (donnée joueur) ; les libellés viennent du manifeste i18n.
 * Exporté pour test unitaire sans monter ECharts.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function formatHeatmapTooltip(
  gamertag: string,
  bucketLabel: string,
  count: number | null,
  t: HeatmapTooltipText,
): string {
  const who = escapeHtml(gamertag)
  const when = escapeHtml(bucketLabel)
  const head = `<strong>${escapeHtml(t.playerLabel)} : ${who}</strong>`
  const slot = `${escapeHtml(t.bucketTypeLabel)} : ${when}`
  if (count == null || count <= 0) {
    return `${head}<br>${slot}<br>${escapeHtml(t.emptyCell)}`
  }
  return `${head}<br>${slot}<br>${escapeHtml(t.matchesLabel)} : ${count}`
}

// Exporté pour tester le cap sans monter le React tree (garde-rail des 12 lignes).
// eslint-disable-next-line react-refresh/only-export-components
export function buildOption(
  cells: HeatmapBucketCell[],
  bucketLabels: string[],
  legendLabel: string,
  tooltipText: HeatmapTooltipText,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()

  // Ordre Y : relations triées par total décroissant (les plus actives en haut),
  // tronquées aux MAX_HEATMAP_ROWS premières.
  const totals = new Map<string, number>()
  for (const c of cells) {
    totals.set(c.xuid, (totals.get(c.xuid) ?? 0) + c.count)
  }
  const rowOrder = [...totals.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, MAX_HEATMAP_ROWS)
    .map(([xuid]) => xuid)
  const rowSet = new Set(rowOrder)
  const xuidToGamertag = new Map(cells.map((c) => [c.xuid, c.gamertag]))
  const rowLabels = rowOrder.map((x) => xuidToGamertag.get(x) ?? '')
  const rowIndex = new Map(rowOrder.map((x, i) => [x, i]))

  // lookup + max sur les seules lignes retenues (l'échelle de couleur suit l'affiché).
  const lookup = new Map<string, number>()
  let maxCount = 0
  for (const c of cells) {
    if (!rowSet.has(c.xuid)) continue
    lookup.set(`${c.xuid}-${c.bucket}`, c.count)
    if (c.count > maxCount) maxCount = c.count
  }

  const data: { value: [number, number, number | null] }[] = []
  for (let b = 0; b < bucketLabels.length; b++) {
    for (const xuid of rowOrder) {
      const count = lookup.get(`${xuid}-${b}`)
      data.push({ value: [b, rowIndex.get(xuid) ?? 0, count != null && count > 0 ? count : null] })
    }
  }

  const hasData = maxCount > 0

  return {
    backgroundColor: CHART_BG,
    grid: { left: 110, right: 20, top: 30, bottom: 40, containLabel: false },
    tooltip: {
      trigger: 'item',
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text },
      formatter: (params: { data: { value: [number, number, number | null] } }) => {
        const [b, row, count] = params.data.value
        return formatHeatmapTooltip(rowLabels[row] ?? '', bucketLabels[b] ?? '', count, tooltipText)
      },
    },
    legend: false as unknown as undefined,
    xAxis: {
      type: 'category',
      data: bucketLabels,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
    },
    yAxis: {
      type: 'category',
      inverse: true,
      data: rowLabels,
      splitLine: { show: true, lineStyle: { color: tc.splitLine } },
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
    },
    visualMap: hasData
      ? {
          min: 0,
          max: maxCount,
          calculable: false,
          show: false,
          orient: 'vertical',
          right: 30,
          top: 'center',
          itemWidth: 12,
          itemHeight: 140,
          // Rampe NEUTRE de fréquence (mono-teinte, luminance monotone, CVD-safe) :
          // ce heatmap mesure l'intensité de rencontre, pas une perf → pas de
          // rouge « mauvais ». Rampe centralisée (cf. components/charts/heatmapColors).
          inRange: { color: heatmapRampTokens('frequency').map(resolveToken) },
          formatter: (val: number) => `${Math.round(val)}`,
          text: [legendLabel, ''],
          textStyle: { color: tc.axisLabel, fontSize: 10 },
        }
      : undefined,
    series: [
      {
        type: 'heatmap',
        data,
        label: {
          show: true,
          fontSize: 11,
          color: tc.text,
          formatter: (params: { data: { value: [number, number, number | null] } }) => {
            const count = params.data.value[2]
            return count != null && count > 0 ? String(count) : ''
          },
        },
        emphasis: { itemStyle: { shadowBlur: 8 } },
      },
    ],
  }
}

type Pt = { xuid: string; bucket: number }

export function RelationsMomentsHeatmap({
  cells,
  bucketLabels,
  title,
  legendLabel,
  emptyMessage,
  tooltipText,
  height,
}: Props) {
  const series: ChartSeries<Pt>[] =
    cells.length > 0
      ? [{ key: 'heatmap', datapoints: cells.map((c) => ({ xuid: c.xuid, bucket: c.bucket })) }]
      : []

  const cellsKey = useMemo(() => JSON.stringify(cells), [cells])
  const bucketsKey = useMemo(() => bucketLabels.join('|'), [bucketLabels])
  // Clé stable des libellés : le parent peut recréer l'objet à chaque rendu sans
  // provoquer de rebuild inutile de l'option ECharts.
  const tooltipKey = useMemo(
    () =>
      [tooltipText.playerLabel, tooltipText.bucketTypeLabel, tooltipText.matchesLabel, tooltipText.emptyCell].join('|'),
    [tooltipText.playerLabel, tooltipText.bucketTypeLabel, tooltipText.matchesLabel, tooltipText.emptyCell],
  )
  const build = useCallback(
    () => buildOption(cells, bucketLabels, legendLabel, tooltipText),
    // cells/bucketLabels/tooltipText capturés via clés stables.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [cellsKey, bucketsKey, legendLabel, tooltipKey],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={build as (s: ChartSeries<Pt>[]) => EChartsCoreOption}
      emptyMessage={emptyMessage}
      height={height ?? 320}
    />
  )
}
