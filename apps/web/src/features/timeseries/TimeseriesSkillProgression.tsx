/**
 * TimeseriesSkillProgression — graphe de progression CSR (classé) ou LUSR (non classé)
 * par match, avec ruptures de saison et signalement des phases de placement.
 *
 * Consomme `TimeseriesMatchRow[]` filtrés par la page parente (en ordre de match).
 * Une série par (skill_playlist_group, skill_rating_type).
 * Rupture de courbe quand skill_season_id change entre deux matchs notés consécutifs.
 * Matchs de placement (skill_measurement_remaining > 0) = symboles séparés sans ligne.
 *
 * Axe X : catégoriel `#N\nMap` (index de match) aligné sur les autres graphes
 * par-match de la page (cf. buildMatchCategories), avec regroupement des
 * étiquettes via tickInterval() quand le panel devient large.
 * Axe Y : cadré sur la magnitude de session (sous-paliers contenant les données,
 * via frameToData) avec bandes de sous-palier, pour rendre lisible le mouvement
 * par-match sur les sessions courtes.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  tickInterval,
  CHART_BG,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { frameToData, buildSkillTierMarkArea } from '@/lib/charts/skillTierBands'
import { gridForRatingTypes } from '@/lib/skillTiers'
import { timeseriesManifest } from '@/lib/i18n/generated/timeseries'
import type { ManifestLocale } from '@/lib/i18n/format'
import { LUSR_GROUP_TOKENS } from '@/features/career/lusr-chains'
import { buildMatchCategories } from './matchLabels'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildProgressionSeries, type ProgressionSeries } from './progressionSeries'

// ── Helpers ─────────────────────────────────────────────────────────────────

function seriesColor(group: string): string {
  return resolveToken(LUSR_GROUP_TOKENS[group] ?? 'chart-series-1')
}

function buildTitle(series: ProgressionSeries[], locale: ManifestLocale): string {
  const types = new Set(series.map(s => s.ratingType.toUpperCase()))
  if (types.size === 1) {
    if (types.has('CSR'))  return timeseriesManifest['timeseries.skill_progression.title_csr'][locale]
    if (types.has('LUSR')) return timeseriesManifest['timeseries.skill_progression.title_lusr'][locale]
  }
  return timeseriesManifest['timeseries.skill_progression.title_mixed'][locale]
}

// buildProgressionSeries + le type ProgressionSeries sont extraits dans
// ./progressionSeries (react-refresh : ce module n'exporte que des composants).

// ── Option ECharts ────────────────────────────────────────────────────────────

function buildOption(
  series: ProgressionSeries[],
  rows: TimeseriesMatchRow[],
  locale: ManifestLocale,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axisBase = getAxisBase(tc)
  const categories = buildMatchCategories(rows)
  const n = categories.length

  const allRatings = series.flatMap(s => s.values.filter((v): v is number => v != null))
  const dataMin = allRatings.length > 0 ? Math.min(...allRatings) : 0
  const dataMax = allRatings.length > 0 ? Math.max(...allRatings) : 0
  const grid = gridForRatingTypes(series.map(s => s.ratingType))
  const { min: yMin, max: yMax } = frameToData(dataMin, dataMax, grid)

  // Série fantôme pour les bandes de sous-palier (reste visible quand les séries réelles sont masquées).
  const ghostSeries = {
    type: 'line',
    name: '__skill_tiers__',
    data: [],
    silent: true,
    legendHoverLink: false,
    markArea: buildSkillTierMarkArea(locale, yMin, yMax, grid, tc),
  }

  const echartsSeriesList: object[] = [ghostSeries]
  const legendData: string[] = []

  for (const s of series) {
    const color = seriesColor(s.group)
    const isCSR = s.ratingType.toUpperCase() === 'CSR'
    const seasonBreakLabel = timeseriesManifest['timeseries.skill_progression.season_break'][locale]

    // Courbe principale (gaps aux matchs non notés + ruptures de saison via markLine).
    echartsSeriesList.push({
      type: 'line',
      name: s.label,
      data: s.values,
      connectNulls: false,
      itemStyle: { color },
      lineStyle: { color, width: 2, type: isCSR ? ('dashed' as const) : ('solid' as const) },
      symbol: 'circle',
      symbolSize: 5,
      showSymbol: true,
      smooth: false,
      markLine: s.seasonBreaks.length > 0 ? {
        silent: true,
        symbol: 'none',
        lineStyle: { type: 'dotted' as const, color: tc.axisLine, width: 1 },
        label: { formatter: seasonBreakLabel, color: tc.axisLabel, fontSize: 10 },
        data: s.seasonBreaks.map(idx => ({ xAxis: categories[idx] })),
      } : undefined,
    })
    legendData.push(s.label)

    // Série placement (symboles séparés, sans ligne).
    if (s.placementPoints.length > 0) {
      const placementLabel = `${timeseriesManifest['timeseries.skill_progression.placement_series'][locale]} (${s.ratingType.toUpperCase()})`
      echartsSeriesList.push({
        type: 'scatter',
        name: placementLabel,
        data: s.placementPoints.map(([idx, y]) => [categories[idx], y]),
        itemStyle: { color, opacity: 0.5 },
        symbol: 'diamond',
        symbolSize: 8,
      })
      legendData.push(placementLabel)
    }
  }

  return {
    backgroundColor: CHART_BG,
    grid: { left: 12, right: 20, top: 30, bottom: 40, containLabel: true },
    tooltip: { trigger: 'axis', ...getTooltipBase(tc) },
    legend: { ...getLegendBase(tc), data: legendData, type: 'scroll' },
    xAxis: {
      ...axisBase,
      type: 'category',
      data: categories,
      axisLabel: {
        ...axisBase.axisLabel,
        interval: tickInterval(n) - 1,
      },
    },
    yAxis: {
      type: 'value',
      name: timeseriesManifest['timeseries.skill_progression.axis_y'][locale],
      min: yMin,
      max: yMax,
      ...axisBase,
      nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
    },
    series: echartsSeriesList,
  }
}

// ── Composant exporté ──────────────────────────────────────────────────────

export interface TimeseriesSkillProgressionProps {
  rows: TimeseriesMatchRow[]
  locale: ManifestLocale
  height?: number
  emptyMessage?: string
}

export function TimeseriesSkillProgression({ rows, locale, height = 280, emptyMessage }: TimeseriesSkillProgressionProps) {
  const series = buildProgressionSeries(rows, locale)

  // Pas de `return null` quand series est vide : on garde le bloc titré et on
  // laisse ChartCard afficher son `emptyMessage` (buildTitle retombe sur le
  // titre générique « Classement » quand aucun type de rating n'est présent).
  const title = buildTitle(series, locale)

  // Adaptateur ChartCard : séries passées = uniquement pour le guard "empty".
  // buildOption reçoit les ProgressionSeries directement via closure.
  const chartSeriesForCard: ChartSeries<number>[] = series.map(s => ({
    key: s.key,
    meta: { label: s.label },
    datapoints: s.values.filter((v): v is number => v != null),
  }))

  return (
    <ChartCard<number>
      title={title}
      series={chartSeriesForCard}
      height={height}
      emptyMessage={emptyMessage}
      buildOption={() => buildOption(series, rows, locale)}
      reviewKey="timeseries.skill_progression"
    />
  )
}
