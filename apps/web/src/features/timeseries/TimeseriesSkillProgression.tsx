/**
 * TimeseriesSkillProgression — graphe de progression CSR (classé) ou LUSR (non classé)
 * par match, avec ruptures de saison et signalement des phases de placement.
 *
 * Consomme `TimeseriesMatchRow[]` filtrés par la page parente.
 * Une série par (skill_playlist_group, skill_rating_type).
 * Rupture de courbe quand skill_season_id change entre deux matchs consécutifs.
 * Matchs de placement (skill_measurement_remaining > 0) = symboles séparés sans ligne.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { LUSR_TIERS } from '@/lib/skillTiers'
import { timeseriesManifest } from '@/lib/i18n/generated/timeseries'
import type { ManifestLocale } from '@/lib/i18n/format'
import { LUSR_GROUP_TOKENS, lusrChainLabel } from '@/features/career/lusr-chains'
import type { TimeseriesMatchRow } from '@/lib/api/types'

// ── Types internes ──────────────────────────────────────────────────────────

type DataPoint = [string, number | null]

interface ProgressionSeries {
  key: string
  label: string
  group: string
  ratingType: string
  points: DataPoint[]         // progression principale (nulls aux ruptures de saison)
  placementPoints: DataPoint[] // matchs de placement
  seasonBreaks: string[]       // dates des ruptures (ISO)
}

// ── Helpers ─────────────────────────────────────────────────────────────────

function groupKey(row: TimeseriesMatchRow): string {
  return `${row.skill_rating_type ?? ''}:${row.skill_playlist_group ?? ''}`
}

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

// ── Construction des séries ──────────────────────────────────────────────────

function buildProgressionSeries(rows: TimeseriesMatchRow[], locale: ManifestLocale): ProgressionSeries[] {
  // 1. Filtrer les rows avec une valeur skill, trier chronologiquement.
  const withSkill = rows
    .filter(r => r.skill_rating_value != null)
    .sort((a, b) => a.start_time.localeCompare(b.start_time))

  // 2. Regrouper par clé (ratingType:playlistGroup).
  const byKey = new Map<string, TimeseriesMatchRow[]>()
  for (const r of withSkill) {
    const k = groupKey(r)
    if (!byKey.has(k)) byKey.set(k, [])
    byKey.get(k)!.push(r)
  }

  const result: ProgressionSeries[] = []

  for (const [, groupRows] of byKey) {
    const first = groupRows[0]
    const ratingType = first.skill_rating_type ?? ''
    const group = first.skill_playlist_group ?? ''
    const label = `${lusrChainLabel(group, locale)} (${ratingType.toUpperCase()})`

    // Midpoint y pour les diamants de placement : milieu de la plage des valeurs réelles.
    // Pour CSR, la valeur stockée pendant les placements est 0.0 — afficher à 0 serait trompeur.
    const realValues = groupRows
      .filter(r => (r.skill_measurement_remaining ?? 0) === 0 && r.skill_rating_value != null)
      .map(r => r.skill_rating_value as number)
    const placementY = realValues.length > 0
      ? (Math.min(...realValues) + Math.max(...realValues)) / 2
      : null

    const points: DataPoint[] = []
    const placementPoints: DataPoint[] = []
    const seasonBreaks: string[] = []

    for (let i = 0; i < groupRows.length; i++) {
      const row = groupRows[i]
      const date = row.start_time
      const isPlacement = (row.skill_measurement_remaining ?? 0) > 0

      if (isPlacement) {
        // Placements : y = milieu de la plage des valeurs réelles (pas 0.0 du CSR en DB).
        if (placementY !== null) placementPoints.push([date, placementY])
        points.push([date, null])
        continue
      }

      // Rupture de saison : season_id différent du match précédent non-placement.
      if (i > 0) {
        const prev = groupRows.slice(0, i).findLast(r => (r.skill_measurement_remaining ?? 0) === 0)
        if (prev && prev.skill_season_id && row.skill_season_id
            && prev.skill_season_id !== row.skill_season_id) {
          points.push([date, null])  // gap visuel
          seasonBreaks.push(date)
        }
      }

      points.push([date, row.skill_rating_value!])
    }

    result.push({
      key: `skill.${ratingType}.${group}`,
      label,
      group,
      ratingType,
      points,
      placementPoints,
      seasonBreaks,
    })
  }

  return result
}

// ── Option ECharts ────────────────────────────────────────────────────────────

function buildTierMarkArea(locale: ManifestLocale) {
  return {
    silent: true,
    label: { show: true, position: 'insideTopLeft' as const, fontSize: 10, opacity: 0.6 },
    data: LUSR_TIERS.map(tier => [
      {
        yAxis: tier.min,
        name: locale === 'fr' ? tier.fr : tier.en,
        itemStyle: { color: resolveToken(tier.token) + '30' },
        label: { color: resolveToken(tier.token) },
      },
      { yAxis: tier.max },
    ]),
  }
}

function buildOption(series: ProgressionSeries[], locale: ManifestLocale): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const axisBase = getAxisBase(tc)
  const intlLocale = locale === 'fr' ? 'fr-FR' : 'en-US'

  const allRatings = series.flatMap(s => s.points.flatMap(p => p[1] != null ? [p[1] as number] : []))
  const dataMin = allRatings.length > 0 ? Math.min(...allRatings) : 0
  const tierMin = LUSR_TIERS.findLast(t => t.min <= dataMin)?.min ?? 0

  // Série fantôme pour les bandes de tier (reste visible quand les séries réelles sont masquées).
  const ghostSeries = {
    type: 'line',
    name: '__skill_tiers__',
    data: [],
    silent: true,
    legendHoverLink: false,
    markArea: buildTierMarkArea(locale),
  }

  const echartsSeriesList: object[] = [ghostSeries]
  const legendData: string[] = []

  for (const s of series) {
    const color = seriesColor(s.group)
    const isCSR = s.ratingType.toUpperCase() === 'CSR'
    const seasonBreakLabel = timeseriesManifest['timeseries.skill_progression.season_break'][locale]

    // Courbe principale (avec gaps aux ruptures de saison).
    echartsSeriesList.push({
      type: 'line',
      name: s.label,
      data: s.points,
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
        data: s.seasonBreaks.map(date => ({ xAxis: date })),
      } : undefined,
    })
    legendData.push(s.label)

    // Série placement (symboles séparés, sans ligne).
    if (s.placementPoints.length > 0) {
      const placementLabel = `${timeseriesManifest['timeseries.skill_progression.placement_series'][locale]} (${s.ratingType.toUpperCase()})`
      echartsSeriesList.push({
        type: 'scatter',
        name: placementLabel,
        data: s.placementPoints,
        itemStyle: { color, opacity: 0.5 },
        symbol: 'diamond',
        symbolSize: 8,
      })
      legendData.push(placementLabel)
    }
  }

  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 20, top: 30, bottom: 60, containLabel: false },
    tooltip: { trigger: 'axis', ...getTooltipBase(tc) },
    legend: { ...getLegendBase(tc), bottom: 5, data: legendData },
    xAxis: {
      type: 'time',
      ...axisBase,
      axisLabel: {
        ...axisBase.axisLabel,
        formatter: (value: number) =>
          new Intl.DateTimeFormat(intlLocale, { month: 'short', day: 'numeric' }).format(new Date(value)),
      },
    },
    yAxis: {
      type: 'value',
      name: timeseriesManifest['timeseries.skill_progression.axis_y'][locale],
      min: tierMin,
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
}

export function TimeseriesSkillProgression({ rows, locale, height = 280 }: TimeseriesSkillProgressionProps) {
  const series = buildProgressionSeries(rows, locale)

  if (series.length === 0) return null

  const title = buildTitle(series, locale)

  // Adaptateur ChartCard : séries passées = uniquement pour le guard "empty".
  // buildOption reçoit les ProgressionSeries directement via closure.
  const chartSeriesForCard: ChartSeries<DataPoint>[] = series.flatMap(s => [
    {
      key: s.key,
      meta: { label: s.label },
      datapoints: s.points.filter((p): p is [string, number] => p[1] != null),
    },
  ])

  return (
    <ChartCard<DataPoint>
      title={title}
      series={chartSeriesForCard}
      height={height}
      buildOption={(_) => buildOption(series, locale)}
    />
  )
}
