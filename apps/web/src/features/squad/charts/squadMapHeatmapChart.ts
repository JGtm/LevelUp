/**
 * squadMapHeatmapChart — teammates.03 : heatmap perf joueur × carte.
 *
 * Spec : .ai/charts_specs/teammates/03_squad_heatmap.yaml
 *
 * visualMap discret (5 paliers) sur les seuils SCORE_THRESHOLDS (75/60/45/30).
 * yAxis = joueurs (moi en haut). xAxis = toutes les cartes jouées en escouade,
 * triées par fréquence décroissante.
 *
 * Accessibilité (CVD) : ce heatmap reste volontairement HORS du helper
 * heatmapRampTokens — ce dernier ne couvre que les rampes CONTINUES
 * (sequential/frequency/divergent). Ici l'échelle est ORDINALE discrète à 5
 * paliers, encodée par les tokens sémantiques perf-tier-* (palette-aware et
 * CVD-safe par construction : axe bleu→jaune→vermillon en palette daltonienne,
 * cf. palettes/okabe-ito) ET chaque palier porte un label texte (opts.pieceLabels)
 * qui désambiguïse même à teintes proches. Migration non pertinente.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SquadMapHeatmap, SquadMapHeatmapCell } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

export interface SquadMapHeatmapOpts {
  mapLabelOf: (mapUI: string) => string
  pieceLabels: { tier1: string; tier2: string; tier3: string; tier4: string; tier5: string }
  noScoreLabel: string
}

export function buildSquadMapHeatmapOption(
  series: ChartSeries<SquadMapHeatmap>[],
  opts: SquadMapHeatmapOpts,
): EChartsCoreOption {
  const heatmap = series[0]?.datapoints[0]
  const players = heatmap?.players ?? []
  const mapsTopn = heatmap?.maps_topn ?? []
  const cells = heatmap?.cells ?? []
  if (!heatmap || players.length === 0 || mapsTopn.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  // Étiquettes X « #N\nCarte » (format compact 2 lignes, comme les autres charts).
  // Nom complet conservé dans `mapNames` pour le tooltip.
  const mapNames = mapsTopn.map(opts.mapLabelOf)
  const xLabels = mapNames.map((name, i) => `#${i + 1}\n${truncateMap(name)}`)
  const yLabels = players

  // Map (player, map) → cell pour lookup O(1).
  const cellByKey = new Map<string, SquadMapHeatmapCell>()
  for (const c of cells) {
    cellByKey.set(`${c.player}|${c.map_ui}`, c)
  }

  const data: Array<[number, number, number | null]> = []
  for (let yi = 0; yi < players.length; yi += 1) {
    for (let xi = 0; xi < mapsTopn.length; xi += 1) {
      const c = cellByKey.get(`${players[yi]}|${mapsTopn[xi]}`)
      const v = c?.perf_avg !== undefined ? Number(c.perf_avg.toFixed(1)) : null
      data.push([xi, yi, v])
    }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 110, left: 8, right: 8, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: unknown) => {
        const point = p as { data?: [number, number, number | null] }
        const d = point?.data
        if (!d) return ''
        const [xi, yi, v] = d
        const cell = cellByKey.get(`${players[yi]}|${mapsTopn[xi]}`)
        const perf = v === null ? opts.noScoreLabel : v.toFixed(1)
        const n = cell?.match_count ?? 0
        return `${escapeHtml(players[yi] ?? '')} — ${escapeHtml(mapNames[xi] ?? '')}<br/>Perf: ${perf}<br/>N: ${n}`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      // margin : décolle les étiquettes (2 lignes « #N\nCarte ») du bas du graphe.
      axisLabel: { ...axis.axisLabel, rotate: -35, interval: 0, margin: 14 },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      inverse: true,
    },
    visualMap: {
      type: 'piecewise',
      pieces: [
        { lt: 30, color: resolveToken('perf-tier-5'), label: opts.pieceLabels.tier5 },
        { gte: 30, lt: 45, color: resolveToken('perf-tier-4'), label: opts.pieceLabels.tier4 },
        { gte: 45, lt: 60, color: resolveToken('perf-tier-3'), label: opts.pieceLabels.tier3 },
        { gte: 60, lt: 75, color: resolveToken('perf-tier-2'), label: opts.pieceLabels.tier2 },
        { gte: 75, color: resolveToken('perf-tier-1'), label: opts.pieceLabels.tier1 },
      ],
      orient: 'horizontal',
      left: 'center',
      bottom: 4,
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        type: 'heatmap',
        data,
        label: { show: false },
        emphasis: { itemStyle: { borderColor: tc.text, borderWidth: 1 } },
      },
    ],
  }
}
