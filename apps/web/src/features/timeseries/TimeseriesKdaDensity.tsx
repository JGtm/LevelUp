/**
 * TimeseriesKdaDensity — chart timeseries.03
 *
 * Densité KDE (smoothed) du FDA + rug plot des valeurs individuelles.
 *  - Série 1 : courbe lissée + zone (area) — densité reconstruite depuis kda_buckets
 *    (Go : buildKDABuckets sur m.KDA, colonne BDD synced ADR 0006).
 *  - Série 2 : ticks verticaux (rug) — un par match à sa valeur r.kda.
 */
import { useMemo, type ReactNode } from 'react'
import type { ChartOption } from '@/components/charts/chartOption'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { axisTooltipFormatter } from '@/components/charts/axisTooltip'
import { tipNumber } from '@/components/charts/tipNumber'
import { resolveToken } from '@/lib/accessibility'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { DistributionBucket, TimeseriesMatchRow } from '@/lib/api/types'
import { ChartFromOption } from './ChartFromOption'
import { buildKdaDensityModel } from './kdaDensity.logic'

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


  const option = useMemo<ChartOption | null>(() => {
    if (buckets.length === 0) return null
    const tc = getEChartsThemeColors()
    const accent = resolveToken('chart-series-1')

    // Le modèle (densité, tapis, moyenne et médiane, repères) vit dans kdaDensity.logic.
    const model = buildKdaDensityModel(buckets, rows)
    if (model == null) return null
    const { densityPoints, xMax, meanRounded, medianRounded, meanX, medianX, rugScatter, rugYTop, rugYBot } = model
    const meanColor = resolveToken('outcome-win')
    const medianColor = resolveToken('outcome-loss')

    return {
      backgroundColor: CHART_BG,
      grid: { top: 16, right: 16, bottom: 36, left: 48 },
      tooltip: {
        ...getTooltipBase(tc),
        trigger: 'axis',
        // L'abscisse est continue (ECharts la rendrait brute, « 1.2340567 ») : la bulle écrit la
        // valeur survolée à deux décimales, et la densité en pourcentage des matchs.
        formatter: axisTooltipFormatter({
          value: (r) => `${tipNumber(r.value, 1)} %`,
          title: (p) => `${labels.xAxis} ${tipNumber(Number(p.axisValue), 2)}`,
        }),
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
          formatter: (v: number) => (v >= 0 ? `${tipNumber(v, 0)} %` : ''),
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
                    label: { formatter: `${labels.mean} ${tipNumber(meanRounded, 2)}`, color: meanColor },
                  }]
                : []),
              ...(medianX != null && medianRounded != null
                ? [{
                    xAxis: medianX,
                    lineStyle: { color: medianColor, width: 1.5, type: 'dashed' as const },
                    label: { formatter: `${labels.median} ${tipNumber(medianRounded, 2)}`, color: medianColor },
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
