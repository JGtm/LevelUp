/**
 * TimeseriesFormCharts — wrappers locaux pour les charts du tab Forme.
 *
 * Couvre timeseries.12, .13, .14, .15, .16, .19 (chart .11 traité via un
 * composant dédié quand l'agrégation backend sera disponible).
 *
 * Tous les composants consomment `TimeseriesMatchRow[]` et calculent leurs
 * propres séries / smoothings côté client. Aucune couleur hex directe : les
 * tokens sémantiques sont résolus via `resolveToken()`.
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  getEChartsThemeColors,
  getAxisBase,
  getLegendBase,
  getTooltipBase,
  CHART_BG,
  escapeHtml,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { perfScale } from '@/lib/accessibility/scales'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

// ─── Helpers numériques ───────────────────────────────────────────────────────

/** Moyenne mobile centrée fenêtre `window` (smoothing). NaN ignorés. */
function rollingMean(values: (number | null | undefined)[], window: number): (number | null)[] {
  const out: (number | null)[] = new Array(values.length).fill(null)
  if (window <= 1) return values.map((v) => (v == null ? null : v))
  const half = Math.floor(window / 2)
  for (let i = 0; i < values.length; i++) {
    const lo = Math.max(0, i - half)
    const hi = Math.min(values.length - 1, i + half)
    let sum = 0
    let n = 0
    for (let k = lo; k <= hi; k++) {
      const v = values[k]
      if (v == null || !Number.isFinite(v)) continue
      sum += v
      n++
    }
    out[i] = n > 0 ? sum / n : null
  }
  return out
}

interface CommonRenderProps {
  height: number
  option: EChartsCoreOption | null
  title?: ReactNode
  emptyMessage?: string
}

function ChartRender({ option, height, title, emptyMessage }: CommonRenderProps) {
  return (
    <ChartFromOption title={title} option={option} height={height} emptyMessage={emptyMessage} />
  )
}

// ─── FDA per match (KPIs) — bars + lissage rolling 5 ────────────────────────

export interface TimeseriesKdaValueTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  fdaLabel: string
  smoothingLabel: string
}

export function TimeseriesKdaValueTrend({
  rows,
  title,
  emptyMessage,
  height = 360,
  fdaLabel,
  smoothingLabel,
}: TimeseriesKdaValueTrendProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    // FDA < 1 → barre rouge (plus de morts que de frags+assists), sinon vert.
    const colNegative = resolveToken('outcome-loss')
    const colPositive = resolveToken('outcome-win')
    const smoothColor = resolveToken('chart-series-1')
    const categories = buildMatchCategories(rows)
    const bars = rows.map((r) => {
      if (r.kda == null || !Number.isFinite(r.kda)) return { value: null }
      const v = Math.round(r.kda * 100) / 100
      return {
        value: v,
        itemStyle: { color: v < 1 ? colNegative : colPositive, opacity: 0.85 },
      }
    })
    const rawValues = rows.map((r) =>
      r.kda != null && Number.isFinite(r.kda) ? r.kda : null,
    )
    const smooth = rollingMean(rawValues, 5)
    const smoothValues = smooth.map((v) => (v == null ? null : Math.round(v * 100) / 100))
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value' },
      series: [
        {
          type: 'bar',
          name: fdaLabel,
          data: bars,
          barMaxWidth: 14,
          markLine: {
            silent: true,
            symbol: 'none',
            // Seuil FDA = 1 : autant de frags+assists que de morts. Barre rouge gras.
            lineStyle: { color: colNegative, width: 2, type: 'solid' },
            data: [{ yAxis: 1 }],
          },
        },
        {
          type: 'line',
          name: smoothingLabel,
          data: smoothValues,
          showSymbol: false,
          smooth: true,
          connectNulls: true,
          lineStyle: { color: smoothColor, width: 2 },
          z: 5,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, fdaLabel, smoothingLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.12 — Performance bars colorées par palier + smoothing ─────────

export interface TimeseriesPerformanceTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  smoothingLabel: string
}

export function TimeseriesPerformanceTrend({
  rows,
  title,
  emptyMessage,
  height = 320,
  smoothingLabel,
}: TimeseriesPerformanceTrendProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const categories = buildMatchCategories(rows)
    const scores = rows.map((r) => r.perf_score ?? null)
    const bars = rows.map((r) => {
      const v = r.perf_score
      if (v == null) return { value: null }
      return {
        value: v,
        itemStyle: { color: resolveToken(perfScale(v)), opacity: 0.85 },
      }
    })
    const smooth = rollingMean(scores, 5)
    const smoothValues = smooth.map((v) => (v == null ? null : Math.round(v * 10) / 10))

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value', min: 0, max: 100 },
      series: [
        {
          type: 'bar',
          name: 'Score',
          data: bars,
          barMaxWidth: 8,
        },
        {
          type: 'line',
          name: smoothingLabel,
          data: smoothValues,
          showSymbol: false,
          smooth: true,
          connectNulls: true,
          lineStyle: { color: resolveToken('chart-series-1'), width: 2 },
          z: 5,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, smoothingLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.13 — Assists timeseries — bars + smoothing rolling 10 ────────

export interface TimeseriesAssistsTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  assistsLabel: string
  smoothingLabel: string
}

export function TimeseriesAssistsTrend({
  rows,
  title,
  emptyMessage,
  height = 320,
  assistsLabel,
  smoothingLabel,
}: TimeseriesAssistsTrendProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const categories = buildMatchCategories(rows)
    const assists = rows.map((r) => r.assists)
    const accent = resolveToken('chart-series-3')
    const smoothColor = resolveToken('chart-series-1')
    const smooth = rollingMean(assists, 10)
    const smoothValues = smooth.map((v) => (v == null ? null : Math.round(v * 100) / 100))
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value', minInterval: 1 },
      series: [
        {
          type: 'bar',
          name: assistsLabel,
          data: assists,
          itemStyle: { color: accent, opacity: 0.8 },
          barMaxWidth: 8,
        },
        {
          type: 'line',
          name: smoothingLabel,
          data: smoothValues,
          showSymbol: false,
          smooth: true,
          connectNulls: true,
          lineStyle: { color: smoothColor, width: 2 },
          z: 5,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, assistsLabel, smoothingLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.14 — Stats par minute (groupées par match) ───────────────────

export interface TimeseriesPerMinuteTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  killsLabel: string
  deathsLabel: string
  assistsLabel: string
  perMinuteSuffix: string
}

export function TimeseriesPerMinuteTrend({
  rows,
  title,
  emptyMessage,
  height = 360,
  killsLabel,
  deathsLabel,
  assistsLabel,
  perMinuteSuffix,
}: TimeseriesPerMinuteTrendProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colKills = resolveToken('chart-series-1')
    const colDeaths = resolveToken('outcome-loss')
    const colAssists = resolveToken('chart-series-3')

    // Étiquettes X au format `#N\nMap` (style SquadPerformanceCharts).
    const categories = buildMatchCategories(rows)

    const perMin = (v: number, sec: number | null | undefined) => {
      if (sec == null || sec <= 0) return null
      return v / (sec / 60)
    }
    const round2 = (v: number | null) => (v == null ? null : Math.round(v * 100) / 100)
    const fmt = (v: unknown) => {
      const n = typeof v === 'number' ? v : 0
      return Math.abs(n).toFixed(2)
    }

    const kills = rows.map((r) => round2(perMin(r.kills, r.time_played_seconds)))
    const deaths = rows.map((r) => {
      const v = perMin(r.deaths, r.time_played_seconds)
      return v == null ? null : -Math.round(v * 100) / 100 // négatif (sous l'axe)
    })
    const assists = rows.map((r) => round2(perMin(r.assists, r.time_played_seconds)))

    return {
      backgroundColor: CHART_BG,
      grid: { top: 24, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        axisPointer: { type: 'shadow' },
        formatter: (params: unknown) => {
          const arr = Array.isArray(params) ? params : []
          if (arr.length === 0) return ''
          const cat = escapeHtml((arr[0] as { name?: string }).name?.replace(/\n/g, ' ') ?? '')
          const lines = arr.map((p) => {
            const point = p as { seriesName: string; value: number; color: string }
            return `<span style="display:inline-block;width:8px;height:8px;border-radius:50%;background:${point.color};margin-right:6px"></span>${escapeHtml(point.seriesName ?? '')}: ${fmt(point.value)}${perMinuteSuffix}`
          })
          return `<strong>${cat}</strong><br/>${lines.join('<br/>')}`
        },
      },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLine: { lineStyle: { color: tc.text, width: 1.5 } },
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          interval: 0,
          fontSize: 9,
        },
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          formatter: (v: number) => Math.abs(v).toFixed(1),
        },
      },
      series: [
        {
          type: 'bar',
          name: killsLabel,
          data: kills,
          itemStyle: { color: colKills, opacity: 0.85 },
          barMaxWidth: 14,
        },
        {
          type: 'bar',
          name: deathsLabel,
          data: deaths,
          itemStyle: { color: colDeaths, opacity: 0.85 },
          barMaxWidth: 14,
        },
        {
          type: 'bar',
          name: assistsLabel,
          data: assists,
          itemStyle: { color: colAssists, opacity: 0.85 },
          barMaxWidth: 14,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, killsLabel, deathsLabel, assistsLabel, perMinuteSuffix, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.15 — Average life timeseries ─────────────────────────────────

export interface TimeseriesAvgLifeTrendProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  lifeLabel: string
}

export function TimeseriesAvgLifeTrend({
  rows,
  title,
  emptyMessage,
  height = 240,
  lifeLabel,
}: TimeseriesAvgLifeTrendProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const accent = resolveToken('chart-series-3')
    const categories = buildMatchCategories(rows)
    const values: (number | null)[] = rows.map((r) => {
      if (r.time_played_seconds == null || r.time_played_seconds <= 0) return null
      const life = r.time_played_seconds / (r.deaths + 1)
      return Math.round(life * 10) / 10
    })
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 48, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value' },
      series: [
        {
          type: 'line',
          name: lifeLabel,
          data: values,
          showSymbol: false,
          smooth: true,
          connectNulls: true,
          areaStyle: { color: accent, opacity: 0.18 },
          lineStyle: { color: accent, width: 2 },
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, lifeLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.16 — Spree + Headshots + Perfect kills (grouped bars) ───────

export interface TimeseriesSpreeHeadshotsProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  spreeLabel: string
  headshotsLabel: string
  perfectLabel: string
}

export function TimeseriesSpreeHeadshots({
  rows,
  title,
  emptyMessage,
  height = 320,
  spreeLabel,
  headshotsLabel,
  perfectLabel,
}: TimeseriesSpreeHeadshotsProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const categories = buildMatchCategories(rows)
    const spree = rows.map((r) => r.max_killing_spree ?? 0)
    const head = rows.map((r) => r.headshot_kills ?? 0)
    const perfect = rows.map((r) => r.perfect_kills ?? 0)
    // Couleurs visuellement distinctes : cycle large dans la palette
    // chart-series (1, 5, 7) — bleu / violet / orange.
    const colSpree = resolveToken('chart-series-1')
    const colHead = resolveToken('chart-series-5')
    const colPerf = resolveToken('chart-series-7')
    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 64, left: 48, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: { ...getAxisBase(tc), type: 'value', minInterval: 1 },
      series: [
        {
          type: 'bar',
          name: spreeLabel,
          data: spree,
          itemStyle: { color: colSpree, opacity: 0.85 },
          barMaxWidth: 8,
        },
        {
          type: 'bar',
          name: headshotsLabel,
          data: head,
          itemStyle: { color: colHead, opacity: 0.85 },
          barMaxWidth: 8,
        },
        {
          type: 'bar',
          name: perfectLabel,
          data: perfect,
          itemStyle: { color: colPerf, opacity: 0.85 },
          barMaxWidth: 8,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, spreeLabel, headshotsLabel, perfectLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── Skill rank + Performance line (Forme) ───────────────────────────────────

export interface TimeseriesSkillRankPerformanceProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  ratingLabel: string
  perfLabel: string
}

export function TimeseriesSkillRankPerformance({
  rows,
  title,
  emptyMessage,
  height = 320,
  ratingLabel,
  perfLabel,
}: TimeseriesSkillRankPerformanceProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const colRating = resolveToken('chart-series-2')
    const colPerf = resolveToken('outcome-win')

    const categories = buildMatchCategories(rows)
    const ratings: (number | null)[] = rows.map((r) =>
      r.skill_rating_value != null && Number.isFinite(r.skill_rating_value)
        ? Math.round(r.skill_rating_value)
        : null,
    )
    const perfs: (number | null)[] = rows.map((r) =>
      r.perf_score != null && Number.isFinite(r.perf_score)
        ? Math.round(r.perf_score * 10) / 10
        : null,
    )

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 56, bottom: 64, left: 56, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: [
        {
          ...getAxisBase(tc),
          type: 'value',
          name: ratingLabel,
          // scale:true → l'axe ne démarre PAS à 0 et fit la plage réelle des
          // valeurs : indispensable pour visualiser les variations fines de
          // skill rank match-à-match (~10-50 pts sur une plage 1000-2000).
          scale: true,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        },
        {
          ...getAxisBase(tc),
          type: 'value',
          name: perfLabel,
          min: 0,
          max: 100,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          type: 'bar',
          name: ratingLabel,
          yAxisIndex: 0,
          data: ratings,
          itemStyle: { color: colRating, opacity: 0.8 },
          barMaxWidth: 14,
        },
        {
          type: 'line',
          name: perfLabel,
          yAxisIndex: 1,
          data: perfs,
          showSymbol: false,
          // smooth:false → segments rectilignes (angles) entre les points,
          // sans interpolation curvée.
          smooth: false,
          connectNulls: true,
          lineStyle: { color: colPerf, width: 2 },
          z: 5,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, ratingLabel, perfLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}

// ─── timeseries.19 — Rank score (personal_score + rank Y2 inversé) ──────────

export interface TimeseriesRankScoreProps {
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  scoreLabel: string
  rankLabel: string
}

export function TimeseriesRankScore({
  rows,
  title,
  emptyMessage,
  height = 320,
  scoreLabel,
  rankLabel,
}: TimeseriesRankScoreProps) {
  const themeVersion = useThemeVersion()

  const option = useMemo<EChartsCoreOption | null>(() => {
    if (rows.length === 0) return null
    const tc = getEChartsThemeColors()
    const categories = buildMatchCategories(rows)
    const scores = rows.map((r) => r.personal_score ?? 0)
    const ranks: (number | null)[] = rows.map((r) =>
      r.rank != null && r.rank > 0 ? r.rank : null,
    )
    const colScore = resolveToken('chart-series-5')
    const colRank = resolveToken('outcome-win')

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 56, bottom: 64, left: 56, containLabel: true },
      tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
      legend: { ...getLegendBase(tc), bottom: 0 },
      xAxis: {
        ...getAxisBase(tc),
        type: 'category',
        data: categories,
        axisLabel: { ...getAxisBase(tc).axisLabel, interval: 0, fontSize: 9 },
      },
      yAxis: [
        {
          ...getAxisBase(tc),
          type: 'value',
          name: scoreLabel,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
        },
        {
          ...getAxisBase(tc),
          type: 'value',
          name: rankLabel,
          inverse: true,
          minInterval: 1,
          nameTextStyle: { color: tc.axisLabel, fontSize: 10 },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          type: 'bar',
          name: scoreLabel,
          yAxisIndex: 0,
          data: scores,
          itemStyle: { color: colScore, opacity: 0.75 },
          barMaxWidth: 10,
        },
        {
          type: 'line',
          name: rankLabel,
          yAxisIndex: 1,
          data: ranks,
          showSymbol: false,
          smooth: false,
          connectNulls: true,
          lineStyle: { color: colRank, width: 2 },
          z: 5,
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, scoreLabel, rankLabel, themeVersion])
  return <ChartRender option={option} height={height} title={title} emptyMessage={emptyMessage} />
}
