/**
 * SessionEngagementOptions — 3 représentations de l'engagement (résidu signé) côte à côte,
 * pour que l'utilisateur choisisse celle qu'il préfère :
 *   - Option 1 : barre divergente de la MOYENNE (signée, gauche/droite de 0).
 *   - Option 2 : histogramme de DISTRIBUTION des résidus par match.
 *   - Option 3 : courbe de progression par match + markLine de moyenne.
 *
 * `engagement_score` (match_series) et `avg_residual_brut` sont centrés sur 0
 * (+ = sur-engagement, − = sous-engagement).
 */
import { type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { HistogramChart, type ChartPointHistogram } from '@/components/charts/HistogramChart'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken, tokenCssVar } from '@/lib/accessibility'
import type { SessionCompareEntry } from '@/lib/api/types'

import { useSessionT } from './_shared'

const round2 = (n: number) => Math.round(n * 100) / 100

/** Découpe des valeurs en K bins {binStart, binEnd, count}. */
// eslint-disable-next-line react-refresh/only-export-components
export function binValues(values: number[], k = 7): ChartPointHistogram[] {
  if (values.length === 0) return []
  const min = Math.min(...values)
  const max = Math.max(...values)
  if (min === max) return [{ binStart: round2(min), binEnd: round2(max), count: values.length }]
  const width = (max - min) / k
  const out: ChartPointHistogram[] = Array.from({ length: k }, (_, i) => ({
    binStart: round2(min + i * width),
    binEnd: round2(min + (i + 1) * width),
    count: 0,
  }))
  for (const v of values) {
    let idx = Math.floor((v - min) / width)
    if (idx >= k) idx = k - 1
    if (idx < 0) idx = 0
    out[idx].count += 1
  }
  return out
}

/** Courbe de progression de l'engagement par match + markLine de moyenne. */
// eslint-disable-next-line react-refresh/only-export-components
export function buildEngagementLineOption(
  series: ChartSeries<{ x: number; y: number }>[],
  opts: { meanLabel: string },
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const mean = round2(points.reduce((acc, p) => acc + p.y, 0) / points.length)
  const color = resolveToken(mean >= 0 ? 'divergent-pos' : 'divergent-neg')
  return {
    backgroundColor: CHART_BG,
    grid: { top: 20, bottom: 28, left: 40, right: 60 },
    tooltip: { ...getTooltipBase(tc), trigger: 'axis' },
    xAxis: { ...axis, type: 'category', data: points.map((_, i) => `#${i + 1}`) },
    yAxis: { ...axis, type: 'value' },
    series: [
      {
        type: 'line',
        data: points.map((p) => p.y),
        smooth: true,
        showSymbol: true,
        symbolSize: 6,
        lineStyle: { color: resolveToken('chart-series-1'), width: 1.5 },
        itemStyle: { color: resolveToken('chart-series-1') },
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color, type: 'dashed', width: 2 },
          label: {
            show: true,
            position: 'end',
            formatter: `${opts.meanLabel} ${mean}`,
            color,
            fontWeight: 'bold',
            textBorderColor: CHART_BG,
            textBorderWidth: 3,
          },
          data: [{ yAxis: mean }],
        },
      },
    ],
  }
}

interface Props {
  entry: SessionCompareEntry | null
}

export function SessionEngagementOptions({ entry }: Props) {
  const t = useSessionT()
  const residuals = (entry?.match_series ?? [])
    .map((p) => p.engagement_score)
    .filter((v): v is number => v != null)
  const avg = entry?.avg_residual_brut ?? null

  if (residuals.length === 0 && avg == null) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t('session.compare.engagement_title')}</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice title={t('session.compare.engagement_empty')} description="" />
        </CardContent>
      </Card>
    )
  }

  const maxAbs = Math.max(1, ...residuals.map((v) => Math.abs(v)), Math.abs(avg ?? 0))
  const histoSeries: ChartSeries<ChartPointHistogram>[] = residuals.length
    ? [{ key: 'engagement', datapoints: binValues(residuals) }]
    : []
  const lineSeries: ChartSeries<{ x: number; y: number }>[] = residuals.length
    ? [{ key: 'engagement', datapoints: residuals.map((y, i) => ({ x: i, y: round2(y) })) }]
    : []

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{t('session.compare.engagement_title')}</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-6 xl:grid-cols-3">
        <OptionBlock label={`1 — ${t('session.detail.engagement_opt_diverging')}`}>
          <DivergingBar value={avg} maxAbs={maxAbs} />
        </OptionBlock>
        <OptionBlock label={`2 — ${t('session.detail.engagement_opt_histogram')}`}>
          <HistogramChart series={histoSeries} height={180} colorToken="chart-series-1" yAxisLabel={t('session.detail.stat_matches')} />
        </OptionBlock>
        <OptionBlock label={`3 — ${t('session.detail.engagement_opt_progression')}`}>
          <ChartCard
            series={lineSeries}
            height={180}
            buildOption={(s) => buildEngagementLineOption(s, { meanLabel: t('session.detail.chart_perf_mean') })}
          />
        </OptionBlock>
      </CardContent>
    </Card>
  )
}

function OptionBlock({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <p className="text-3xs font-semibold uppercase tracking-wide text-muted-foreground">{label}</p>
      {children}
    </div>
  )
}

/** Barre divergente centrée sur 0 : remplit à droite (+) ou à gauche (−), couleur par signe. */
function DivergingBar({ value, maxAbs }: { value: number | null; maxAbs: number }) {
  if (value == null) return <p className="text-sm text-muted-foreground">—</p>
  const positive = value >= 0
  const pct = maxAbs > 0 ? Math.min(50, (Math.abs(value) / maxAbs) * 50) : 0
  const color = tokenCssVar(positive ? 'divergent-pos' : 'divergent-neg')
  return (
    <div className="space-y-2 py-4">
      <div className="relative h-7 rounded bg-muted/60">
        <div className="absolute left-1/2 top-0 h-full w-px bg-border" aria-hidden="true" />
        <div
          className="absolute top-0 h-full rounded"
          style={{
            backgroundColor: color,
            width: `${pct}%`,
            ...(positive ? { left: '50%' } : { right: '50%' }),
          }}
        />
      </div>
      <p className="text-center text-lg font-bold" style={{ color }}>
        {value > 0 ? '+' : ''}
        {round2(value)}
      </p>
    </div>
  )
}
