/**
 * heatmapChart — Win rate par carte.
 *
 * Produit un ChartSeries<ChartPointHeatmap>[], consommé par Heatmap2DChart.
 * Remplace l'ancien builder Plotly heatmap 1D.
 *
 * Multi-titres : winAxisLabel et noms de cartes (via mapLabelOf) passés
 * en argument par le caller. Fallback sur l'ID brut si la carte n'est pas
 * dans le mapping du titre courant.
 */
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { MapBreakdownRow } from '@/lib/api/types'

export interface HeatmapSeriesArgs {
  rows: MapBreakdownRow[]
  winAxisLabel: string
  mapLabelOf: (mapId: string) => string
}

export function buildHeatmapSeries({
  rows,
  winAxisLabel,
  mapLabelOf,
}: HeatmapSeriesArgs): ChartSeries<ChartPointHeatmap>[] {
  if (rows.length === 0) return []

  const sorted = [...rows].sort((a, b) => b.win_rate - a.win_rate)
  const datapoints: ChartPointHeatmap[] = sorted.map((r) => ({
    x: mapLabelOf(r.map_ui),
    y: winAxisLabel,
    value: r.win_rate * 100,
    detail: { match_count: r.match_count },
  }))

  return [{ key: 'heatmap-map', datapoints }]
}
