/**
 * TimeseriesKdaDensity — chart timeseries.03
 *
 * Densité KDE (smoothed) du FDA + rug plot des valeurs individuelles.
 *  - Série 1 : courbe lissée + zone (area) — densité reconstruite depuis kda_buckets
 *    (Go : buildKDABuckets sur m.KDA, colonne BDD synced ADR 0006).
 *  - Série 2 : ticks verticaux (rug) — un par match à sa valeur r.kda.
 */
import { useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { DistributionBucket, TimeseriesMatchRow } from '@/lib/api/types'
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesKdaDensityProps {
  buckets: DistributionBucket[]
  rows: TimeseriesMatchRow[]
  height?: number
  title?: ReactNode
  emptyMessage?: string
  labels: {
    density: string
    rug: string
    xAxis: string
    mean: string
    median: string
  }
}

export function TimeseriesKdaDensity({
  buckets,
  rows,
  height = 360,
  title,
  emptyMessage,
  labels,
}: TimeseriesKdaDensityProps) {
  const themeVersion = useThemeVersion()


  const option = useMemo<EChartsCoreOption | null>(() => {
    if (buckets.length === 0) return null
    const tc = getEChartsThemeColors()
    const accent = resolveToken('chart-series-1')

    // Total des matchs pour normaliser en pourcentage.
    const total = buckets.reduce((s, b) => s + b.count, 0)
    if (total === 0) return null

    // Densité : (midpoint, %) → série lisse avec area. Restreint à la plage
    // réelle des données pour ne pas afficher des buckets vides au-delà.
    const densityPoints = buckets.map((b) => [
      (b.bucket_lower + b.bucket_upper) / 2,
      Math.round((b.count / total) * 1000) / 10, // % avec 1 décimale
    ])

    const xMax = buckets.reduce(
      (m, b) => (b.bucket_upper > m ? b.bucket_upper : m),
      0,
    )

    // Rug : valeurs FDA individuelles (r.kda — colonne BDD, sync ADR 0006).
    // On collecte aussi les valeurs full-range (sans clamp xMax) pour le calcul
    // mean/median, qui doit refléter toutes les données — y compris la queue.
    const rugValues: number[] = []
    const allKds: number[] = []
    for (const r of rows) {
      if (r.kda == null || !Number.isFinite(r.kda)) continue
      const kda = r.kda
      allKds.push(kda)
      if (kda <= xMax) rugValues.push(Math.round(kda * 1000) / 1000)
    }

    // Mean et median sur l'ensemble des matchs (population, pas sample).
    let mean: number | null = null
    let median: number | null = null
    if (allKds.length > 0) {
      mean = allKds.reduce((s, v) => s + v, 0) / allKds.length
      const sorted = [...allKds].sort((a, b) => a - b)
      const mid = Math.floor(sorted.length / 2)
      median = sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid]
    }
    const meanRounded = mean != null ? Math.round(mean * 100) / 100 : null
    const medianRounded = median != null ? Math.round(median * 100) / 100 : null

    const yMaxPct = densityPoints.reduce((m, p) => (p[1] > m ? p[1] : m), 0)
    const rugYTop = -(yMaxPct * 0.05)
    const rugYBot = -(yMaxPct * 0.14)
    const rugScatter = rugValues.map((v) => [v, rugYBot])

    const meanColor = resolveToken('outcome-win')
    const medianColor = resolveToken('outcome-loss')
    // Position du label : clamp dans la plage X visible pour rester lisible.
    const meanX = meanRounded != null ? Math.min(meanRounded, xMax) : null
    const medianX = medianRounded != null ? Math.min(medianRounded, xMax) : null

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 36, left: 48 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        valueFormatter: (val: unknown) => {
          if (typeof val === 'number') return `${val.toFixed(1)} %`
          return String(val)
        },
      },
      xAxis: {
        ...getAxisBase(tc),
        type: 'value',
        name: labels.xAxis,
        nameLocation: 'middle',
        nameGap: 24,
        nameTextStyle: { color: tc.axisLabel, fontSize: 11 },
        min: 0,
        max: xMax,
      },
      yAxis: {
        ...getAxisBase(tc),
        type: 'value',
        min: rugYBot,
        axisLabel: {
          ...getAxisBase(tc).axisLabel,
          formatter: (v: number) => (v >= 0 ? `${v.toFixed(0)} %` : ''),
        },
      },
      series: [
        {
          name: labels.density,
          type: 'line',
          data: densityPoints,
          smooth: true,
          showSymbol: false,
          lineStyle: { color: accent, width: 2 },
          areaStyle: { color: accent, opacity: 0.2 },
          markLine: {
            silent: true,
            symbol: 'none',
            lineStyle: { width: 1.5, type: 'dashed' },
            label: {
              color: tc.text,
              fontSize: 10,
              fontWeight: 600,
              backgroundColor: tc.tooltipBg,
              padding: [2, 4],
              borderRadius: 2,
              position: 'insideEndTop',
            },
            data: [
              ...(meanX != null && meanRounded != null
                ? [{
                    xAxis: meanX,
                    lineStyle: { color: meanColor, width: 1.5, type: 'dashed' as const },
                    label: { formatter: `${labels.mean} ${meanRounded.toFixed(2)}`, color: meanColor },
                  }]
                : []),
              ...(medianX != null && medianRounded != null
                ? [{
                    xAxis: medianX,
                    lineStyle: { color: medianColor, width: 1.5, type: 'dashed' as const },
                    label: { formatter: `${labels.median} ${medianRounded.toFixed(2)}`, color: medianColor },
                  }]
                : []),
            ],
          },
          z: 2,
        },
        {
          name: labels.rug,
          type: 'custom',
          renderItem: (_params: unknown, api: unknown) => {
            const a = api as { value: (i: number) => number; coord: (v: number[]) => number[] }
            const x = a.value(0)
            const top = a.coord([x, rugYTop])
            const bot = a.coord([x, rugYBot])
            return {
              type: 'line',
              shape: { x1: top[0], y1: top[1], x2: bot[0], y2: bot[1] },
              style: { stroke: accent, lineWidth: 1, opacity: 0.45 },
            }
          },
          data: rugScatter,
          z: 1,
          tooltip: { show: false },
        },
      ],
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buckets, rows, labels, themeVersion])

  return (
    <ChartFromOption
      title={title}
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      reviewKey="timeseries.fda_distribution"
    />
  )
}
