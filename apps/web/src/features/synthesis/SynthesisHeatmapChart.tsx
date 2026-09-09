/**
 * SynthesisHeatmapChart — synthesis.03.
 * Heatmap 2D heure × jour (X=heure, Y=jour) colorée par win_rate via la rampe
 * DIVERGENTE centralisée (perdant → neutre 50 % → gagnant) : le win_rate est un
 * indicateur signé autour de 0,5, et la rampe divergente est CVD-safe par
 * construction — neutre gris en palette daltonienne (cf. heatmapColors).
 * Toutes les 168 cellules sont émises — `value: null` pour les cases vides.
 *
 * Passe par le wrapper canonique `Heatmap2DChart` (lot C2, décision D1 du plan
 * vague C formes — « il existe déjà un composant canonique, on l'étend, on
 * n'en crée pas un second »). `valueRange={[0, 1]}` FIGE l'échelle : le
 * neutre à 50 % doit rester au CENTRE de la rampe quelle que soit la plage
 * réelle des taux de victoire mesurés sur la période — laisser le wrapper
 * auto-ajuster min/max (comportement par défaut des 4 autres consommateurs)
 * décentrerait le neutre.
 *
 * Lundi en haut, Dimanche en bas : le wrapper déduit l'ordre de l'axe Y de la
 * PREMIÈRE APPARITION de chaque jour dans les datapoints (pas d'option
 * `inverse`, contrairement à l'ancienne implémentation ECharts à la main) —
 * les points sont donc émis Dimanche → Lundi pour que Lundi occupe le dernier
 * index (le haut d'un axe catégoriel non inversé).
 */
import { useCallback, useMemo } from 'react'
import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { escapeHtml } from '@/components/charts/_utils'
import { dowLabels, HOUR_LABELS, calendarChartText } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { HeatmapCell } from '@/lib/api/types'

interface Props {
  cells: HeatmapCell[]
  title?: string
  height?: number
}

/** Construit les 168 points (24 h × 7 j), Dimanche → Lundi (cf. doc de tête —
 *  ordre d'apparition qui place Lundi en haut de l'axe Y non inversé). */
function buildPoints(cells: HeatmapCell[], dowLabelsList: readonly string[]): ChartPointHeatmap[] {
  const lookup = new Map<string, { win_rate: number; count: number }>()
  for (const c of cells) {
    if (c.count > 0) {
      lookup.set(`${c.dow}-${c.hour}`, { win_rate: c.win_rate ?? 0, count: c.count })
    }
  }

  const points: ChartPointHeatmap[] = []
  for (let d = 6; d >= 0; d--) {
    for (let h = 0; h < 24; h++) {
      const cell = lookup.get(`${d}-${h}`)
      points.push({
        x: HOUR_LABELS[h],
        y: dowLabelsList[d],
        value: cell ? cell.win_rate : null,
        detail: { count: cell ? cell.count : 0 },
      })
    }
  }
  return points
}

export function SynthesisHeatmapChart({ cells, title, height }: Props) {
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
      const wrStr = `${((point.value ?? 0) * 100).toFixed(1)}%`
      return `${escapeHtml(point.y)} ${escapeHtml(point.x)}<br/>${txt.winRate} : ${wrStr}<br/>${txt.matches} : ${count}`
    },
    [txt],
  )

  return (
    <Heatmap2DChart
      title={title}
      series={series}
      height={height ?? 300}
      paletteMode="divergent"
      valueRange={[0, 1]}
      formatTooltip={formatTooltip}
    />
  )
}
