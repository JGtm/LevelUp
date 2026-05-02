/**
 * squadSynergyRadarChart — teammates.06 : radar 6 axes (1 trace par joueur).
 *
 * Spec : .ai/charts_specs/teammates/06_synergy_radar.yaml
 *
 * - 1 série par joueur (main + coéquipiers).
 * - Lignes seules (PAS d'aire centrale) — multi-profils superposés
 *   deviennent illisibles avec `areaStyle`.
 * - Couleurs des joueurs cohérentes avec la pill / combobox de la page.
 *
 * Le builder consomme directement `SquadSynergyRadarSeries[]` (pas de
 * passage par `ChartSeries<T>` — payload trop spécifique).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { CHART_BG, legendBase, tooltipBase } from '@/components/charts/_utils'
import type { SquadSynergyRadarSeries } from '@/lib/api/types'

export interface SquadSynergyRadarOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  /** axis.key → label affiché (i18n). */
  axisLabels: Record<string, string>
}

export function buildSquadSynergyRadarOption(
  series: SquadSynergyRadarSeries[],
  opts: SquadSynergyRadarOpts,
): EChartsCoreOption {
  if (series.length === 0) return { backgroundColor: CHART_BG }

  // Tous les profils ont la même structure d'axes ; on prend le premier.
  const axes = series[0].axes.map((a) => ({
    name: opts.axisLabels[a.axis] ?? a.axis,
    max: 100,
  }))

  const data = series.map((s) => {
    const color = opts.colorByPlayer[s.player] ?? '#888'
    return {
      name: s.player,
      value: s.axes.map((a) => a.value),
      itemStyle: { color },
      lineStyle: { color, width: 2 },
      // Pas d'areaStyle → ligne seule (spec : show_fill=false).
      symbol: 'circle',
      symbolSize: 4,
    }
  })

  return {
    backgroundColor: CHART_BG,
    tooltip: {
      ...tooltipBase,
      formatter: (params: { name: string; value: number[] }) => {
        const lines = axes.map((a, i) => `${a.name}: <b>${params.value[i].toFixed(0)}</b>`)
        return `<b>${params.name}</b><br/>${lines.join('<br/>')}`
      },
    },
    legend: { ...legendBase, data: data.map((d) => d.name) },
    radar: {
      indicator: axes,
      shape: 'polygon',
      splitNumber: 4,
      axisName: { color: 'rgba(255,255,255,0.65)', fontSize: 10 },
      splitArea: { areaStyle: { color: ['rgba(255,255,255,0.02)', 'rgba(255,255,255,0.05)'] } },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
    },
    series: [{ type: 'radar', data }],
  }
}
