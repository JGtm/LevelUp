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
import {
  CHART_BG,
  escapeHtml,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadSynergyRadarSeries } from '@/lib/api/types'

export interface SquadSynergyRadarOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  /** axis.key → label affiché (i18n). */
  axisLabels: Record<string, string>
  /**
   * Préfixe i18n de la valeur BRUTE au survol (« brut » / « raw »). Le DTO porte
   * déjà `raw` à côté de `value` : sans lui, le radar n'affiche qu'un score
   * normalisé 0-100 impossible à relier à une grandeur de jeu.
   */
  rawLabel: string
}

/**
 * Formatage de la valeur brute : les 6 axes n'ont pas le même ordre de grandeur
 * (Combat ~ centaines de points, Survie ~ 1,6, Score ~ 195/min). On adapte la
 * précision plutôt que d'imposer un arrondi qui écraserait les axes ratio.
 */
function formatRaw(v: number): string {
  const a = Math.abs(v)
  if (a >= 100) return v.toFixed(0)
  if (a >= 10) return v.toFixed(1)
  return v.toFixed(2)
}

export function buildSquadSynergyRadarOption(
  series: SquadSynergyRadarSeries[],
  opts: SquadSynergyRadarOpts,
): EChartsCoreOption {
  if (series.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()

  // Tous les profils ont la même structure d'axes ; on prend le premier.
  const axes = (series[0].axes ?? []).map((a) => ({
    name: opts.axisLabels[a.axis] ?? a.axis,
    max: 100,
  }))

  // Valeurs BRUTES par joueur, dans l'ordre des axes : le tooltip les affiche à
  // côté du normalisé (B3(3)). Le DTO les porte déjà (SquadSynergyRadarAxis.raw).
  const rawByPlayer = new Map<string, number[]>(
    series.map((s) => [s.player, (s.axes ?? []).map((a) => a.raw)]),
  )

  const data = series.map((s) => {
    const color = opts.colorByPlayer[s.player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    return {
      name: s.player,
      value: (s.axes ?? []).map((a) => a.value),
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
      ...getTooltipBase(tc),
      formatter: (params: { name: string; value: number[] }) => {
        const raws = rawByPlayer.get(params.name) ?? []
        const lines = axes.map((a, i) => {
          const norm = `${escapeHtml(a.name ?? '')}: <b>${params.value[i].toFixed(0)}</b>`
          const raw = raws[i]
          // Valeur brute absente (profil partiel) → on n'affiche que le normalisé
          // plutôt qu'un « (brut NaN) ». `Number.isFinite` couvre aussi undefined.
          return !Number.isFinite(raw)
            ? norm
            : `${norm} <span style="opacity:.7">(${escapeHtml(opts.rawLabel)} ${formatRaw(raw)})</span>`
        })
        return `<b>${escapeHtml(params.name ?? '')}</b><br/>${lines.join('<br/>')}`
      },
    },
    legend: { ...getLegendBase(tc), data: data.map((d) => d.name) },
    radar: {
      indicator: axes,
      shape: 'polygon',
      splitNumber: 4,
      axisName: { color: tc.axisLabel, fontSize: 10 },
      splitArea: { areaStyle: { color: [tc.splitAreaA, tc.splitAreaB] } },
      splitLine: { lineStyle: { color: tc.splitLine } },
      axisLine: { lineStyle: { color: tc.axisLine } },
    },
    series: [{ type: 'radar', data }],
  }
}
