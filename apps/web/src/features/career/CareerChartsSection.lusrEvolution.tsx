/**
 * CareerChartsSection — career.04 — évolution LUSR / CSR par playlist_group.
 *
 * Découpé depuis CareerChartsSection.tsx (audit #6 god-file split).
 * Inclut : ligne par chaîne (slayer/objectif/...) + markAreas par tier + dataZoom
 * activé seulement si l'historique dépasse 13 mois. Une seule échelle (LUSR ou CSR)
 * à la fois, bascule dans le titre quand les deux existent (cf. lusrEvolution.logic).
 */
import { useState } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { Segmented } from '@/components/ui/segmented'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { intlLocale as toIntlLocale } from '@/lib/formatters'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { frameToData, buildSkillTierMarkArea } from '@/lib/charts/skillTierBands'
import { gridForRatingTypes } from '@/lib/skillTiers'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { LUSR_GROUP_TOKENS, lusrChainLabel } from './lusr-chains'
import type { CareerLusrCheckpoint } from '@/lib/api/types'
import {
  type RatingScale,
  availableScales,
  chartableCheckpoints,
  checkpointsForScale,
  resolveScale,
} from './lusrEvolution.logic'

// Hauteur minimale du graphe et marges verticales de la zone de tracé : partagées
// par `grid` et l'estimation de hauteur utile qui décide de la densité des libellés
// de rang (en mode fluid la carte peut être plus haute : estimation basse, sûre).
const CHART_HEIGHT = 320
const GRID_TOP = 30
const GRID_BOTTOM = 60

const SCALE_LABEL_KEYS = {
  lusr: 'career.charts.lusr_rating_axis_y',
  csr: 'career.charts.csr_rating_axis_y',
} as const satisfies Record<RatingScale, keyof typeof careerManifest>

function lusrGroupColor(group: string): string {
  return resolveToken(LUSR_GROUP_TOKENS[group] ?? 'chart-series-1')
}

function buildLusrSeries(checkpoints: CareerLusrCheckpoint[], locale: ManifestLocale): ChartSeries<[string, number]>[] {
  const byKey = new Map<string, { group: string; ratingType: string; pts: Map<string, number> }>()

  for (const cp of checkpoints) {
    if (!cp.recorded_at) continue
    const group = cp.playlist_group ?? 'arena_slayer'
    const ratingType = cp.rating_type ?? 'LUSR'
    const seriesKey = `${ratingType}:${group}`
    const date = cp.recorded_at.slice(0, 10)

    if (!byKey.has(seriesKey)) {
      byKey.set(seriesKey, { group, ratingType, pts: new Map() })
    }
    byKey.get(seriesKey)!.pts.set(date, cp.rating_value)
  }

  return Array.from(byKey.entries()).map(([seriesKey, { group, ratingType, pts }]) => {
    const label = `${lusrChainLabel(group, locale)} (${ratingType})`
    return {
      key: `career.lusr.${seriesKey}`,
      meta: { label, groupKey: group, ratingType },
      datapoints: Array.from(pts.entries())
        .sort(([a], [b]) => a.localeCompare(b))
        .map(([date, val]) => [date, val] as [string, number]),
    }
  })
}

function buildLusrEvolutionOption(
  series: ChartSeries<[string, number]>[],
  scale: RatingScale,
  locale: ManifestLocale,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axisBase = getAxisBase(tc)
  const intlLocale = toIntlLocale(locale)

  const allRatings = series.flatMap(s => s.datapoints.map(p => p[1]))
  const dataMin = allRatings.length > 0 ? Math.min(...allRatings) : 0
  const dataMax = allRatings.length > 0 ? Math.max(...allRatings) : 0
  // Axe Y cadré sur la magnitude (sous-paliers contenant les données + marge),
  // grille selon le type de rating. Cf. graphe « Classement ».
  const grid = gridForRatingTypes(series.map(s => (s.meta as { ratingType?: string } | undefined)?.ratingType))
  const { min: yMin, max: yMax } = frameToData(dataMin, dataMax, grid)

  // Fenêtre par défaut = 12 derniers mois depuis le point le plus récent.
  // Si l'historique est plus court, on affiche tout (dataZoom laisse dérouler vers le passé).
  const allDates = series.flatMap(s => s.datapoints.map(p => p[0]))
  const lastDate = allDates.length > 0 ? allDates.reduce((a, b) => (a > b ? a : b)) : null
  const firstDate = allDates.length > 0 ? allDates.reduce((a, b) => (a < b ? a : b)) : null
  const defaultWindowStart = lastDate
    ? new Date(new Date(lastDate).getTime() - 365 * 86_400_000).toISOString().slice(0, 10)
    : null
  // N'active le zoom que si les données dépassent 13 mois d'historique.
  const needsZoom = firstDate !== null && defaultWindowStart !== null && firstDate < defaultWindowStart

  // Légende explicite = les noms des séries réelles seulement.
  // La série fantôme n'est pas incluse dans legend.data → absente de la légende.
  const legendData = series.map(
    s => (s.meta as { label: string } | undefined)?.label ?? s.key,
  )

  // Les bandes de sous-palier sont attachées à une série fantôme (pas de données, pas de légende).
  // Sans ghost, les markArea disparaissent quand la série porteuse est masquée.
  const ghostSeries = {
    type: 'line',
    name: '__lusr_tiers__',
    data: [],
    silent: true,
    legendHoverLink: false,
    markArea: buildSkillTierMarkArea({
      locale, yMin, yMax, grid, theme: tc, plotHeightPx: CHART_HEIGHT - GRID_TOP - GRID_BOTTOM,
    }),
  }

  const echartsSeriesList = [
    ghostSeries,
    ...series.map((s) => {
      const meta = s.meta as { label: string; groupKey: string; ratingType: string } | undefined
      const label = meta?.label ?? s.key
      // LUSR = ligne pleine, CSR = ligne pointillée pour distinguer visuellement.
      const isCSR = meta?.ratingType === 'CSR'
      const color = lusrGroupColor(meta?.groupKey ?? '')
      return {
        type: 'line',
        name: label,
        data: s.datapoints,
        itemStyle: { color },
        lineStyle: { color, width: 2, type: isCSR ? ('dashed' as const) : ('solid' as const) },
        symbol: 'circle',
        symbolSize: 5,
        showSymbol: true,
        smooth: false,
      }
    }),
  ]

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 20, top: GRID_TOP, bottom: GRID_BOTTOM, containLabel: false },
    tooltip: { trigger: 'axis', ...getTooltipBase(tc) },
    legend: { ...getLegendBase(tc), bottom: 5, data: legendData },
    xAxis: {
      type: 'time',
      ...axisBase,
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (value: number) =>
          new Intl.DateTimeFormat(intlLocale, { month: 'short', year: '2-digit' }).format(new Date(value)),
      },
    },
    yAxis: {
      type: 'value',
      name: careerManifest[SCALE_LABEL_KEYS[scale]][locale],
      min: yMin,
      max: yMax,
      ...axisBase,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    ...(needsZoom ? {
      dataZoom: [{
        type: 'inside',
        startValue: defaultWindowStart,
        endValue: lastDate,
        filterMode: 'none',
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
      }],
    } : {}),
    series: echartsSeriesList,
  }
}

// ── Composant exporté ──────────────────────────────────────────────────────

export interface CareerLusrEvolutionChartProps {
  lusrCheckpoints: CareerLusrCheckpoint[]
  locale: ManifestLocale
  /**
   * Étire la card pour remplir sa cellule (CSS Grid stretch) — `height` (320) devient le
   * minimum garanti. Opt-in quand le chart partage une ligne avec « Classements » (souvent
   * plus haut) : évite le vide sous le graphe (V72-28). Défaut false (pleine largeur).
   */
  fluid?: boolean
}

export function CareerLusrEvolutionChart({ lusrCheckpoints, locale, fluid }: CareerLusrEvolutionChartProps) {
  const [selectedScale, setSelectedScale] = useState<RatingScale | null>(null)
  const chartable = chartableCheckpoints(lusrCheckpoints)
  const scales = availableScales(chartable)
  const scale = resolveScale(selectedScale, scales)
  const titleText = careerManifest['career.charts.lusr_evolution_title'][locale]
  const title = scales.length > 1 && scale ? (
    <div className="flex items-center justify-between gap-2">
      <span>{titleText}</span>
      <Segmented
        options={scales.map(s => ({ value: s, label: careerManifest[SCALE_LABEL_KEYS[s]][locale] }))}
        value={scale}
        onChange={setSelectedScale}
        ariaLabel={careerManifest['career.charts.lusr_evolution_scale'][locale]}
      />
    </div>
  ) : titleText
  return (
    <ChartCard<[string, number]>
      title={title}
      series={buildLusrSeries(checkpointsForScale(chartable, scale), locale)}
      height={CHART_HEIGHT}
      fluid={fluid}
      buildOption={(series) => buildLusrEvolutionOption(series, scale ?? 'lusr', locale)}
      emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
    />
  )
}
