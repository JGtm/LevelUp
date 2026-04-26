/**
 * heatmapChart — Win rate par carte (heatmap horizontale).
 *
 * Multi-titres : libellés UI (titre, axes, hovertemplate) ET noms de cartes
 * passent par le caller. Les noms de cartes sont résolus via `mapLabelOf`
 * qui consomme le mapping `assets.map` du titre courant — fallback sur l'ID
 * brut si la carte n'est pas mappée (titre minimaliste, ou carte récente
 * pas encore au TOML).
 */
import type { MapBreakdownRow, PlotlyFigurePayload } from '@/lib/api/types'
import { buildOrdinalColorscale } from '@/lib/accessibility'

interface HeatmapChartArgs {
  rows: MapBreakdownRow[]
  title: string
  winAxis: string
  matchesLabel: string
  /**
   * Résout l'ID brut d'une carte (cf. MapBreakdownRow.map_ui) vers son
   * libellé localisé via assets.map du titre courant. Si la carte n'est
   * pas mappée, doit retourner l'ID inchangé.
   */
  mapLabelOf: (mapId: string) => string
}

export function buildHeatmapChart({
  rows,
  title,
  winAxis,
  matchesLabel,
  mapLabelOf,
}: HeatmapChartArgs): PlotlyFigurePayload | null {
  if (rows.length === 0) return null

  // Tokens dans l'ordre croissant : 0 % = tier-5 (mauvais), 100 % = tier-1 (excellent)
  const colorscale = buildOrdinalColorscale([
    'perf-tier-5', 'perf-tier-4', 'perf-tier-3', 'perf-tier-2', 'perf-tier-1',
  ])

  const sorted = [...rows].sort((a, b) => b.win_rate - a.win_rate)
  const maps = sorted.map((r) => mapLabelOf(r.map_ui))
  const winrates = sorted.map((r) => r.win_rate)
  const counts = sorted.map((r) => r.match_count)
  const customdata = counts.map((c, i) => [c, winrates[i]])

  return {
    data: [
      {
        type: 'heatmap',
        z: [winrates],
        x: maps,
        y: [winAxis],
        colorscale: colorscale as unknown as string,
        zmin: 0,
        zmax: 100,
        customdata: [customdata],
        hovertemplate:
          `<b>%{x}</b><br>${winAxis}: %{z:.1f}%<br>${matchesLabel}: %{customdata[0]}<extra></extra>`,
        showscale: true,
        colorbar: { title: winAxis, thickness: 14 },
      },
    ],
    layout: {
      height: 160,
      margin: { l: 80, r: 20, t: 30, b: 80 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
      title: { text: title, font: { size: 13 } },
      xaxis: { tickangle: -35 },
    },
  }
}
