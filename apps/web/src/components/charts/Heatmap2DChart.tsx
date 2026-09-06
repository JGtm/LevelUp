/**
 * Heatmap2DChart — wrapper ECharts pour `ChartSeries<ChartPointHeatmap>`.
 *
 * Consomme :
 *   - 1 série dont les datapoints sont `{ x, y, value, detail? }`.
 *   - Axes X et Y sont des chaînes (déjà résolues côté service en labels
 *     lisibles).
 *
 * Cas d'usage Squad V2 :
 *   - S3 Heatmap player × map (perf score)
 *   - S6 Intensity match × bucket
 *   - S5 Impact heatmap match × player (potentiel future)
 *
 * Le `value` est mappé via `visualMap` ECharts en gradient cold→hot ou
 * divergent low→high selon `paletteMode`.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { resolveToken } from '@/lib/accessibility'
import { useSettingsDraftStore, type ColorPalette } from '@/stores/settingsDraftStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { heatmapRampTokens } from './heatmapColors'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from './_utils'

export interface ChartPointHeatmap {
  x: string
  y: string
  /**
   * Valeur de la cellule, ou `null` pour une case VIDE — non peinte, hors échelle,
   * sans étiquette (ajout 2026-09-06, correction W1).
   *
   * POURQUOI UNE CASE VIDE PLUTÔT QU'UNE CASE ABSENTE. Les axes de ce wrapper sont
   * DÉDUITS de l'ordre d'apparition des points : omettre une case décale les
   * catégories, et une matrice carrée à diagonale omise sortait avec un axe X
   * décalé d'un cran par rapport à l'axe Y (roster [A,B,C,D] → colonnes B,C,D,A).
   * Une case impossible doit donc être ÉMISE, et dite vide.
   */
  value: number | null
  detail?: Record<string, unknown>
}

/**
 * Modes de rampe acceptes par ce wrapper — sous-ensemble de HeatmapRampMode.
 *
 * 'frequency' (ajout 2026-09-06, matrice d'echange de l'escouade) : rampe NEUTRE
 * mono-teinte, monotone en luminance dans toutes les palettes. C'est le mode d'une
 * intensite qui ne porte AUCUN jugement — un nombre de vengeances n'est ni chaud ni
 * froid, et la rampe cold->hot lui collerait un « bon / mauvais » que la donnee ne dit
 * pas. Elle interdit aussi de confondre l'echelle avec les couleurs par JOUEUR.
 */
export type HeatmapPaletteMode = 'sequential' | 'divergent' | 'frequency'

export interface Heatmap2DChartProps {
  title?: string
  series: ChartSeries<ChartPointHeatmap>[]
  loading?: boolean
  error?: Error | null
  emptyMessage?: string
  height?: number
  /** sequential (cold→hot) ou divergent (low→neutral→high). Default 'sequential'. */
  paletteMode?: HeatmapPaletteMode
  /** Min/max forcés du visualMap (default = auto-fit). */
  valueRange?: [number, number]
  /**
   * Contenu HTML du tooltip d'une cellule. Absent = le libellé historique de la
   * heatmap joueur × carte (taux de victoire + nombre de matchs), qui ne convient
   * qu'à ce cas d'usage — toute autre donnée DOIT passer sa propre fonction, sinon
   * la cellule s'annonce sous un nom qui n'est pas le sien.
   *
   * L'appelant est responsable de l'échappement de ce qu'il injecte.
   */
  formatTooltip?: (point: ChartPointHeatmap) => string
}

export function Heatmap2DChart({
  title,
  series,
  loading,
  error,
  emptyMessage,
  height,
  paletteMode = 'sequential',
  valueRange,
  formatTooltip,
}: Heatmap2DChartProps) {
  // Palette d'accessibilité active : pilote la rampe CVD-safe (rebuild via
  // useColorPaletteVersion dans ChartCard + ce sélecteur au changement de palette).
  const colorPalette = useSettingsDraftStore((s) => s.localUiPrefs.colorPalette)
  const buildOption = useCallback(
    (s: ChartSeries<ChartPointHeatmap>[]) =>
      buildHeatmap2DOption(s, { paletteMode, valueRange, colorPalette, formatTooltip }),
    [paletteMode, valueRange, colorPalette, formatTooltip],
  )

  return (
    <ChartCard
      title={title}
      series={series}
      loading={loading}
      error={error}
      emptyMessage={emptyMessage}
      height={height}
      buildOption={buildOption}
    />
  )
}

interface BuildOpts {
  paletteMode?: HeatmapPaletteMode
  valueRange?: [number, number]
  /** Palette d'accessibilité active — pilote la rampe CVD-safe (cf. heatmapColors). */
  colorPalette?: ColorPalette
  formatTooltip?: (point: ChartPointHeatmap) => string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildHeatmap2DOption(
  series: ChartSeries<ChartPointHeatmap>[],
  opts: BuildOpts = {},
): EChartsCoreOption {
  const { paletteMode = 'sequential', valueRange, colorPalette = 'default' } = opts
  const { formatTooltip } = opts
  if (series.length === 0) {
    return { backgroundColor: CHART_BG }
  }
  const main = series[0]
  const dps = main.datapoints

  // Axes X / Y déduits des datapoints (preserve l'ordre d'apparition).
  const xs: string[] = []
  const ys: string[] = []
  const xSet = new Set<string>()
  const ySet = new Set<string>()
  for (const d of dps) {
    if (!xSet.has(d.x)) {
      xSet.add(d.x)
      xs.push(d.x)
    }
    if (!ySet.has(d.y)) {
      ySet.add(d.y)
      ys.push(d.y)
    }
  }

  // ECharts heatmap data : [xIndex, yIndex, value, detail].
  //
  // Une case VIDE part en `'-'` : c'est la valeur « pas de donnée » d'ECharts — la
  // cellule n'est pas peinte, le visualMap ne la classe pas, et son étiquette est
  // vide. `null` ferait la même chose côté rendu mais casserait le `Math.min`
  // ci-dessous ; les valeurs vides sont donc écartées AVANT le calcul de l'échelle.
  const data = dps.map((d) => [xs.indexOf(d.x), ys.indexOf(d.y), d.value ?? '-', d.detail])

  const remplies = dps.filter((d): d is ChartPointHeatmap & { value: number } => d.value != null)
  const valeurs = remplies.map((d) => d.value)
  const minV = valueRange?.[0] ?? (valeurs.length > 0 ? Math.min(...valeurs) : 0)
  const maxV = valueRange?.[1] ?? (valeurs.length > 0 ? Math.max(...valeurs) : 0)

  // Rampe centralisée : en palette d'accessibilité, une heatmap séquentielle
  // bascule sur la rampe de fréquence (luminance monotone, CVD-safe).
  const colors = heatmapRampTokens(paletteMode, colorPalette).map(resolveToken)

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 80, left: 96, right: 24 },
    tooltip: {
      ...getTooltipBase(tc),
      position: 'top',
      formatter: (params: { data: [number, number, number | string, Record<string, unknown>?] }) => {
        const [xi, yi, brut, detail] = params.data
        // Case vide : rien à dire, pas même « 0 ».
        if (typeof brut !== 'number') return ''
        const v = brut
        if (formatTooltip) {
          return formatTooltip({ x: xs[xi], y: ys[yi], value: v, detail })
        }
        const count = detail?.count ?? 0
        return `${escapeHtml(ys[yi])} × ${escapeHtml(xs[xi])}<br/>Win Rate: <b>${(v * 100).toFixed(1)}%</b><br/>Matchs: <b>${count}</b>`
      },
    },
    xAxis: { ...axis, type: 'category', data: xs, splitArea: { show: true } },
    yAxis: { ...axis, type: 'category', data: ys, splitArea: { show: true } },
    visualMap: {
      min: minV,
      max: maxV,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: 8,
      inRange: { color: colors },
      textStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: [
      {
        name: main.key,
        type: 'heatmap',
        data,
        label: {
          show: true,
          formatter: (params: { data: [number, number, number | string, Record<string, unknown>?] }) => {
            const [, , brut, detail] = params.data
            // Case vide : aucune étiquette (un « 0 » se lirait comme une mesure).
            if (typeof brut !== 'number') return ''
            return String(detail?.count ?? 0)
          },
        },
        emphasis: {
          itemStyle: { shadowBlur: 8, shadowColor: 'rgba(0,0,0,0.5)' },
        },
      },
    ],
  }
}
