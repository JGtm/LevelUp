/**
 * ExplorerActivityHeatmapChart — « Carte de chaleur d'activité commune » : la grille
 * heure × jour des matchs croisés avec un joueur cible (Explorer, mode Joueur).
 *
 * Variante intensité de « Activité par jour et heure » : la couleur reflète le `count`
 * (nombre de matchs croisés) via la rampe NEUTRE de fréquence (mono-teinte, luminance
 * monotone, CVD-safe), pas le taux de victoire — la question posée est « quand se
 * croise-t-on le plus ? ». L'infobulle rappelle le taux de victoire à titre informatif.
 *
 * RENDU CANONIQUE (2026-09-20, lot 3 des ajustements pré-v7.5) : ce fichier n'écrit plus
 * son option ECharts, il range ses cellules en points et `Heatmap2DChart` les rend. Mêmes
 * cases aérées, mêmes titres d'axes, et plus de barre de dégradé — chaque case porte son
 * nombre et l'infobulle dit le reste.
 */
import { useCallback, useMemo } from 'react'

import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { dowLabels, HOUR_LABELS, calendarChartText } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { HeatmapCell } from '@/lib/api/types'

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

/**
 * Les 168 points (24 h × 7 j), Lundi → Dimanche : l'axe Y inversé du wrapper place Lundi
 * en haut, l'ordre des DONNÉES ne porte donc pas de décision d'affichage. Une heure sans
 * match croisé vaut `null` (case sans mesure), jamais 0.
 */
function buildPoints(cells: HeatmapCell[], dowLabelsList: readonly string[]): ChartPointHeatmap[] {
  const lookup = new Map<string, { count: number; win_rate: number }>()
  for (const c of cells) {
    if (c.count > 0) {
      lookup.set(`${c.dow}-${c.hour}`, { count: c.count, win_rate: c.win_rate ?? 0 })
    }
  }

  const points: ChartPointHeatmap[] = []
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      const cell = lookup.get(`${d}-${h}`)
      points.push({
        x: HOUR_LABELS[h],
        y: dowLabelsList[d],
        value: cell ? cell.count : null,
        detail: { count: cell ? cell.count : 0, winRate: cell ? cell.win_rate : 0 },
      })
    }
  }
  return points
}

export function ExplorerActivityHeatmapChart({ cells, title, height }: Props) {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const txt = calendarChartText(locale)
  const dowLabelsList = dowLabels(locale)

  const series: ChartSeries<ChartPointHeatmap>[] = useMemo(
    () => (cells.length > 0 ? [{ key: 'heatmap', datapoints: buildPoints(cells, dowLabelsList) }] : []),
    [cells, dowLabelsList],
  )

  const formatTooltip = useCallback(
    (point: ChartPointHeatmap) => {
      const count = (point.detail?.count as number | undefined) ?? 0
      const winRate = (point.detail?.winRate as number | undefined) ?? 0
      const wrStr = `${(winRate * 100).toFixed(1)}%`
      return `${escapeHtml(point.y)} ${escapeHtml(point.x)}<br/>${txt.commonMatches} : ${count}<br/>${txt.winRate} : ${wrStr}`
    },
    [txt],
  )

  // La rampe de fréquence part d'un bleu très sombre : l'encre du nombre écrit dans la
  // case vient du thème, jamais du défaut gris d'ECharts, qui y disparaît.
  const cellLabelColor = getEChartsThemeColors().text

  return (
    <Heatmap2DChart
      title={title}
      series={series}
      height={height ?? 300}
      paletteMode="frequency"
      cellLabelColor={cellLabelColor}
      formatTooltip={formatTooltip}
      yAxisInverse
      axisNames={{ x: txt.hourAxis, y: txt.dayAxis }}
      // La réglette est masquée, mais son ORIENTATION décide aussi des marges du tracé :
      // celles de la verticale logent les titres d'axes, que l'horizontale écraserait.
      visualMapOrient="vertical"
      showVisualMap={false}
      emptyCells="hidden"
    />
  )
}
