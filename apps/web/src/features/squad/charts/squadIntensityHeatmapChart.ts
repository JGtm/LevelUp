/**
 * squadIntensityHeatmapChart — teammates.15 : heatmap d'intensité kills par phase.
 *
 * Spec : .ai/charts_specs/teammates/15_squad_intensity_heatmap.yaml
 *
 *   - Y axis : matchs (label = "Carte — date").
 *   - X axis : 10 phases (0-10%, ..., 90-100%).
 *   - Z value : densité de kills normalisée 0..1 par match.
 *   - visualMap continu sur la rampe NEUTRE de fréquence (mono-teinte, luminance
 *     monotone → CVD-safe partout) via le helper central heatmapRampTokens.
 *   - Toggle (cf. wrapper) : "all" ou un joueur spécifique.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { heatmapRampTokens } from '@/components/charts/heatmapColors'
import { resolveToken } from '@/lib/accessibility'
import type { SquadIntensityMatchRow } from '@/lib/api/types'
import { truncateMap } from '@/lib/charts/matchLabels'

const PHASE_LABELS = [
  '0-10%', '10-20%', '20-30%', '30-40%', '40-50%',
  '50-60%', '60-70%', '70-80%', '80-90%', '90-100%',
]

export interface SquadIntensityOpts {
  /** label tooltip pour la valeur (i18n caller, ex: "kills"). */
  zLabel: string
}

export function buildSquadIntensityHeatmapOption(
  rows: SquadIntensityMatchRow[],
  opts: SquadIntensityOpts,
): EChartsCoreOption {
  if (rows.length === 0) return { backgroundColor: CHART_BG }

  // Y axis : « #N Carte ». N = numéro du match dans l'ordre passé par le caller
  // (oldest-first → #1 en haut, le plus récent en bas grâce à yAxis.inverse).
  // La carte est extraite du label "Carte — date" ; label complet gardé pour le tooltip.
  const fullLabels = rows.map((r) => r.label)
  const yLabels = rows.map((r, i) => `#${i + 1} ${truncateMap(r.label.split(' — ')[0] || r.label)}`)
  const data: Array<[number, number, number]> = []
  for (let yi = 0; yi < rows.length; yi += 1) {
    const phases = rows[yi].phases ?? []
    for (let xi = 0; xi < PHASE_LABELS.length; xi += 1) {
      const v = phases[xi] ?? 0
      data.push([xi, yi, Number(v.toFixed(2))])
    }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 60, left: 8, right: 60, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'item',
      formatter: (p: unknown) => {
        const point = p as { data?: [number, number, number] }
        const d = point?.data
        if (!d) return ''
        const [xi, yi, v] = d
        return `${escapeHtml(fullLabels[yi] ?? '')}<br/>${PHASE_LABELS[xi]}<br/>${opts.zLabel}: <b>${(v * 100).toFixed(0)}%</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: PHASE_LABELS,
      axisLabel: { ...axis.axisLabel, rotate: -25, interval: 0 },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      inverse: true,
    },
    visualMap: {
      type: 'continuous',
      min: 0,
      max: 1,
      calculable: false,
      orient: 'vertical',
      right: 4,
      top: 'middle',
      // Rampe NEUTRE de fréquence (mono-teinte, luminance monotone, CVD-safe) :
      // l'intensité de kills est une mesure de volume, pas une perf bien/mal →
      // pas de rampe cold→hot. Rampe centralisée (cf. components/charts/heatmapColors).
      inRange: { color: heatmapRampTokens('frequency').map(resolveToken) },
      textStyle: { color: tc.axisLabel, fontSize: 10 },
      formatter: (v: number) => `${(v * 100).toFixed(0)}%`,
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

export const SQUAD_INTENSITY_PHASE_LABELS = PHASE_LABELS
