/**
 * RelationsMomentsHeatmap — grille « Rythme des rencontres » (relation × créneau).
 *
 * Générique : relation × bucket (heure 0..23 OU jour de semaine selon le toggle du
 * parent). La couleur reflète le `count` (matchs communs) via la rampe NEUTRE de
 * fréquence — un nombre de rencontres n'est ni bon ni mauvais.
 *
 * RENDU CANONIQUE (2026-09-20, lot 3 des ajustements pré-v7.5) : ce fichier n'écrit plus
 * son option ECharts. Il ne fait que RANGER ses cellules en points, et
 * `components/charts/Heatmap2DChart` les rend — mêmes cases aérées, mêmes titres d'axes et
 * même infobulle que « Activité par jour et heure », qui est le rendu de référence. Une
 * grille de plus qui écrivait sa propre option, c'était une grille de plus qui divergeait
 * au premier réglage.
 *
 * La barre de dégradé est partie avec la migration : chaque case porte son nombre, et
 * l'infobulle nomme la relation, le créneau et le compte.
 */
import { useMemo } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'

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
 * « AllyPlayer · 14h » suivi d'un compteur sans contexte. Le gamertag est échappé
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

/**
 * buildBucketPoints — les cellules rangées en points de la grille canonique.
 *
 * L'axe Y garde les MAX_HEATMAP_ROWS relations les plus actives (total décroissant) ; les
 * autres ne sont pas rendues. Toutes les cases du produit (relation × bucket) sont émises,
 * une sans rencontre valant `null` : les axes du wrapper sont déduits de l'ordre
 * d'apparition des points, en omettre une décalerait les catégories.
 *
 * Exporté pour tester le cap sans monter le React tree.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildBucketPoints(
  cells: HeatmapBucketCell[],
  bucketLabels: string[],
): ChartPointHeatmap[] {
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

  const lookup = new Map<string, number>()
  for (const c of cells) {
    if (!rowSet.has(c.xuid)) continue
    lookup.set(`${c.xuid}-${c.bucket}`, c.count)
  }

  const points: ChartPointHeatmap[] = []
  for (let b = 0; b < bucketLabels.length; b++) {
    for (const xuid of rowOrder) {
      const count = lookup.get(`${xuid}-${b}`)
      points.push({
        x: bucketLabels[b],
        y: xuidToGamertag.get(xuid) ?? '',
        value: count != null && count > 0 ? count : null,
        detail: { count: count ?? 0 },
      })
    }
  }
  return points
}

export function RelationsMomentsHeatmap({
  cells,
  bucketLabels,
  title,
  emptyMessage,
  tooltipText,
  height,
}: Props) {
  const points = useMemo(() => buildBucketPoints(cells, bucketLabels), [cells, bucketLabels])
  const series: ChartSeries<ChartPointHeatmap>[] =
    points.length > 0 ? [{ key: 'heatmap', datapoints: points }] : []

  // Clé stable des libellés : le parent peut recréer l'objet à chaque rendu sans
  // provoquer de rebuild inutile de l'option ECharts.
  const { playerLabel, bucketTypeLabel, matchesLabel, emptyCell } = tooltipText
  const formatTooltip = useMemo(
    () => (point: ChartPointHeatmap) =>
      formatHeatmapTooltip(point.y, point.x, point.value, {
        playerLabel,
        bucketTypeLabel,
        matchesLabel,
        emptyCell,
      }),
    [playerLabel, bucketTypeLabel, matchesLabel, emptyCell],
  )

  // La rampe de fréquence part d'un bleu très sombre : l'encre du nombre écrit dans la
  // case vient du thème, jamais du défaut gris d'ECharts, qui y disparaît.
  const cellLabelColor = getEChartsThemeColors().text

  return (
    <Heatmap2DChart
      title={title}
      series={series}
      emptyMessage={emptyMessage}
      height={height ?? 320}
      paletteMode="frequency"
      // La réglette est masquée, mais son ORIENTATION décide aussi des marges du tracé :
      // celles de la verticale logent les titres d'axes, que l'horizontale écraserait.
      visualMapOrient="vertical"
      showVisualMap={false}
      cellLabelColor={cellLabelColor}
      formatTooltip={formatTooltip}
      yAxisInverse
      axisNames={{ x: bucketTypeLabel, y: playerLabel }}
      emptyCells="hidden"
    />
  )
}
