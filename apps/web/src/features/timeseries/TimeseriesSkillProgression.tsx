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
import { LUSR_GROUP_TOKENS, lusrChainLabel } from '@/features/career/lusr-chains'
import { buildMatchCategories } from './matchLabels'
import type { TimeseriesMatchRow } from '@/lib/api/types'

// ── Types internes ──────────────────────────────────────────────────────────

interface ProgressionSeries {
  key: string
  label: string
  group: string
  ratingType: string
  /** Valeurs de classement indexées par position de match (null hors matchs notés). */
  values: (number | null)[]
  /** Points de placement : [indexMatch, y]. */
  placementPoints: Array<[number, number]>
  /** Index de match où débute une nouvelle saison (rupture de courbe). */
  seasonBreaks: number[]
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
  const n = rows.length

  // Regrouper les matchs notés par clé (ratingType:playlistGroup), en conservant
  // leur index de position (= index de catégorie X, aligné sur buildMatchCategories).
  const byKey = new Map<string, Array<{ row: TimeseriesMatchRow; idx: number }>>()
  rows.forEach((r, idx) => {
    if (r.skill_rating_value == null) return
    const k = groupKey(r)
    if (!byKey.has(k)) byKey.set(k, [])
    byKey.get(k)!.push({ row: r, idx })
  })

  const result: ProgressionSeries[] = []

  for (const [, entries] of byKey) {
    const first = entries[0].row
    const ratingType = first.skill_rating_type ?? ''
    const group = first.skill_playlist_group ?? ''
    const label = `${lusrChainLabel(group, locale)} (${ratingType.toUpperCase()})`

    // Midpoint y pour les diamants de placement : milieu de la plage des valeurs réelles.
    // Pour CSR, la valeur stockée pendant les placements est 0.0 — afficher à 0 serait trompeur.
    const realValues = entries
      .filter(e => (e.row.skill_measurement_remaining ?? 0) === 0 && e.row.skill_rating_value != null)
      .map(e => e.row.skill_rating_value as number)
    const placementY = realValues.length > 0
      ? (Math.min(...realValues) + Math.max(...realValues)) / 2
      : null

    const values = new Array<number | null>(n).fill(null)
    const placementPoints: Array<[number, number]> = []
    const seasonBreaks: number[] = []
    let prevSeasonId: string | null | undefined

    for (const { row, idx } of entries) {
      const isPlacement = (row.skill_measurement_remaining ?? 0) > 0

      if (isPlacement) {
        // Placements : y = milieu de la plage des valeurs réelles (pas 0.0 du CSR en DB).
        if (placementY !== null) placementPoints.push([idx, placementY])
        continue // values[idx] reste null → coupure de ligne
      }

      // Rupture de saison : season_id différent du précédent match noté.
      if (prevSeasonId != null && row.skill_season_id && prevSeasonId !== row.skill_season_id) {
        seasonBreaks.push(idx)
      }

      values[idx] = row.skill_rating_value!
      prevSeasonId = row.skill_season_id ?? prevSeasonId
    }

    result.push({
      key: `skill.${ratingType}.${group}`,
      label,
      group,
      ratingType,
      values,
      placementPoints,
      seasonBreaks,
    })
  }

  return result
}

// Réexport pour tests unitaires de la transformation pure.
export { buildProgressionSeries }

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
}

export function TimeseriesSkillProgression({ rows, locale, height = 280 }: TimeseriesSkillProgressionProps) {
  const series = buildProgressionSeries(rows, locale)

  if (series.length === 0) return null

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
      buildOption={() => buildOption(series, rows, locale)}
    />
  )
}
