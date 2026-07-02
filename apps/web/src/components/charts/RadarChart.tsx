/**
 * RadarChart — wrapper ECharts pour le radar 6 axes.
 *
 * Ne consomme PAS `ChartSeries<T>` standard car le payload backend est
 * spécifique (cf. service.RadarChartSeries Go). Chaque trace = 1 joueur,
 * 6 axes alignés avec valeurs 0..100 normalisées.
 *
 * Cas d'usage Squad V2 :
 *   - S8 Radar Participation 6 axes (Combat/Survival/Support/Score/Objective/Impact)
 *
 * Les axes sont passés par le caller (le service amont fournit la liste
 * dans l'ordre canonique). Les valeurs raw sont disponibles dans
 * `meta.raw_by_axis` pour le tooltip.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard } from './ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors, getLegendBase, getTooltipBase, seriesColor } from './_utils'

/** 1 axe radar : libellé + valeur 0..100 + raw debug. */
export interface RadarAxis {
  axis: string
  value: number
  raw: number
}

/** 1 série radar (1 joueur). Mirror service.RadarChartSeries Go. */
export interface RadarSeriesPayload {
  key: string
  labelKey?: string
  axes: RadarAxis[]
  meta?: { gamertag?: string; mode_family?: string; raw_by_axis?: Record<string, number> }
}

export interface RadarChartProps {
  title?: string
  series: RadarSeriesPayload[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** Si fournie, override le label de série (par défaut meta.gamertag ou key). */
  seriesNameResolver?: (s: RadarSeriesPayload) => string
  /** Map axis key → label affiché (i18n du caller). Si absent, utilise axis brut. */
  axisLabels?: Record<string, string>
  /**
   * Si true, le tooltip affiche la valeur BRUTE (`meta.raw_by_axis`, 1 décimale)
   * au lieu du pourcentage normalisé 0..100. Utile quand chaque axe a un cap de
   * référence (radar de frags par match) : l'utilisateur veut lire "2.4 frags"
   * et non "48 %". Défaut false → comportement historique (radar participation).
   */
  rawInTooltip?: boolean
  /**
   * Si true, affiche la VALEUR de chaque axe directement sur le radar (sous le libellé
   * d'axe), en plus du tooltip — pour les radars de session (FDA / Frags) où lire au
   * survol n'est pas pratique. Utilise `meta.raw_by_axis` (radar mono-série).
   */
  showValues?: boolean
}

export function RadarChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  seriesNameResolver,
  axisLabels,
  rawInTooltip,
  showValues,
}: RadarChartProps) {
  // Le ChartCard est typé sur ChartSeries<T> mais on passe RadarSeriesPayload.
  // Ce wrapper est volontairement non-strict (cast au call site) pour
  // garder le ChartCard minimal — le radar a une structure trop spécifique.
  const buildOption = useCallback(
    (s: RadarSeriesPayload[]) =>
      buildRadarOption(s, { seriesNameResolver, axisLabels, rawInTooltip, showValues }),
    [seriesNameResolver, axisLabels, rawInTooltip, showValues],
  )

  return (
    <ChartCard
      title={title}
      series={series as unknown as { key: string; datapoints: unknown[] }[]}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={() => buildOption(series)}
    />
  )
}

interface BuildOpts {
  seriesNameResolver?: (s: RadarSeriesPayload) => string
  axisLabels?: Record<string, string>
  rawInTooltip?: boolean
  showValues?: boolean
}

/** Valeur d'axe affichée sur le radar : entier tel quel, sinon 1 décimale. */
const fmtRadarValue = (v: number) => (Number.isInteger(v) ? String(v) : v.toFixed(1))

// eslint-disable-next-line react-refresh/only-export-components
export function buildRadarOption(
  series: RadarSeriesPayload[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { seriesNameResolver, axisLabels, rawInTooltip, showValues } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const tc = getEChartsThemeColors()

  // Axes : du premier joueur (tous alignés côté backend). On conserve la clé
  // d'axe (`key`) en plus du libellé pour retrouver la valeur brute par axe.
  const axisKeys = series[0].axes.map((a) => a.axis)
  const axes = series[0].axes.map((a) => ({
    key: a.axis,
    name: axisLabels?.[a.axis] ?? a.axis,
    max: 100,
  }))

  const data = series.map((s, idx) => {
    const name =
      seriesNameResolver?.(s) ?? (s.meta?.gamertag ?? s.key)
    return {
      name,
      value: s.axes.map((a) => a.value),
      itemStyle: { color: seriesColor(idx) },
      lineStyle: { color: seriesColor(idx), width: 2 },
      areaStyle: { color: seriesColor(idx), opacity: 0.15 },
    }
  })

  // Lookup nom de série → valeurs brutes par clé d'axe (pour rawInTooltip).
  const rawByName = new Map(
    series.map((s, idx) => [data[idx].name, s.meta?.raw_by_axis]),
  )

  return {
    backgroundColor: CHART_BG,
    tooltip: {
      ...getTooltipBase(tc),
      formatter: (params: { name: string; value: number[] }) => {
        const raw = rawInTooltip ? rawByName.get(params.name) : undefined
        const lines = axes.map((a, i) => {
          if (raw) {
            const v = raw[axisKeys[i]]
            return `${escapeHtml(a.name)}: <b>${v != null ? v.toFixed(1) : '—'}</b>`
          }
          return `${escapeHtml(a.name)}: <b>${params.value[i].toFixed(0)}</b>`
        })
        return `<b>${escapeHtml(params.name)}</b><br/>${lines.join('<br/>')}`
      },
    },
    legend: {
      ...getLegendBase(tc),
      data: data.map((d) => d.name),
    },
    radar: {
      indicator: axes,
      shape: 'polygon',
      splitNumber: 4,
      axisName: { color: tc.axisLabel, fontSize: 10 },
      splitArea: { areaStyle: { color: [tc.splitAreaA, tc.splitAreaB] } },
      splitLine: { lineStyle: { color: tc.splitLine } },
      axisLine: { lineStyle: { color: tc.axisLine } },
    },
    series: [
      {
        type: 'radar',
        data,
        // Valeurs affichées AUX POINTES (vertices), en clair (≈ blanc sur thème sombre) avec
        // un léger contour pour rester lisible sur l'aire colorée. Valeur BRUTE par axe via
        // meta.raw_by_axis (radar mono-série de session) ; dimensionIndex = index de l'axe.
        ...(showValues
          ? {
              label: {
                show: true,
                color: tc.text,
                fontSize: 10,
                fontWeight: 'bold' as const,
                textBorderColor: tc.tooltipBg,
                textBorderWidth: 2,
                formatter: (p: { name?: string; dimensionIndex?: number }) => {
                  const di = p.dimensionIndex
                  if (di == null) return ''
                  const raw = p.name != null ? rawByName.get(p.name) : undefined
                  const v = raw?.[axisKeys[di]]
                  return v != null ? fmtRadarValue(v) : ''
                },
              },
            }
          : {}),
      },
    ],
  }
}
