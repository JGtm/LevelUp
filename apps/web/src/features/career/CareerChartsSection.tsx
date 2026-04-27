/**
 * CareerChartsSection — section charts de la page Carrière (chunk P2.B).
 *
 * Refactoré Phase 2 méta-plan : migration `react-plotly.js` → wrappers
 * ECharts (S10). Utilise les données brutes du DTO (`xp_history`,
 * `lusr.checkpoints`) au lieu des `PlotlyFigurePayload` server-side.
 *
 * Les 2 gauges (rank + hero progress) sont rendues en cartes simplifiées
 * (pourcentage + détail) en attendant un futur wrapper `<Gauge>` ECharts
 * dédié (différé Phase 3).
 *
 * i18n : labels résolus via `careerManifest` + `formatMessage` côté caller
 * (le composant prend les labels en prop pour rester découplé du store).
 */
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'

import type {
  CareerHistoryPoint,
  CareerLusrCheckpoint,
  HeroProgress,
  CareerSummary,
} from '@/lib/api/types'

export interface CareerChartsSectionLabels {
  rankProgressTitle: string
  heroProgressTitle: string
  xpHistoryTitle: string
  xpHistoryAxisY: string
  lusrRatingTitle: string
  lusrRatingAxisY: string
  placeholderUnavailable: string
  placeholderDescription: string
}

export interface CareerChartsSectionProps {
  xpHistory: CareerHistoryPoint[]
  lusrCheckpoints: CareerLusrCheckpoint[]
  summary: CareerSummary | null
  heroProgress: HeroProgress | null
  labels: CareerChartsSectionLabels
}

export function CareerChartsSection({
  xpHistory,
  lusrCheckpoints,
  summary,
  heroProgress,
  labels,
}: CareerChartsSectionProps) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2" data-testid="career-charts-section">
      <RankProgressCard summary={summary} labels={labels} />
      <HeroProgressCard heroProgress={heroProgress} labels={labels} />

      {xpHistory.length > 1 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{labels.xpHistoryTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <TimeseriesLineChart
              series={xpHistorySeries(xpHistory)}
              height={220}
              xAxisType="time"
              outcomeMarkers={false}
            />
          </CardContent>
        </Card>
      ) : (
        <UnavailableCard title={labels.xpHistoryTitle} labels={labels} />
      )}

      {lusrCheckpoints.length > 1 ? (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">{labels.lusrRatingTitle}</CardTitle>
          </CardHeader>
          <CardContent>
            <TimeseriesLineChart
              series={lusrSeries(lusrCheckpoints)}
              height={220}
              xAxisType="time"
              outcomeMarkers={false}
            />
          </CardContent>
        </Card>
      ) : (
        <UnavailableCard title={labels.lusrRatingTitle} labels={labels} />
      )}
    </div>
  )
}

function xpHistorySeries(points: CareerHistoryPoint[]): ChartSeries<ChartPoint2D>[] {
  return [
    {
      key: 'career.xp_history.cumulative',
      meta: { gamertag: 'XP cumulé' },
      datapoints: points.map((p) => ({
        x: p.recorded_at,
        y: p.xp_total,
      })),
    },
  ]
}

function lusrSeries(points: CareerLusrCheckpoint[]): ChartSeries<ChartPoint2D>[] {
  // 1 série par playlist_group (au plus 1-2 dans la pratique).
  const byGroup = new Map<string, ChartPoint2D[]>()
  for (const cp of points) {
    if (!byGroup.has(cp.playlist_group)) {
      byGroup.set(cp.playlist_group, [])
    }
    byGroup.get(cp.playlist_group)!.push({
      x: cp.recorded_at,
      y: cp.rating_value,
    })
  }
  return Array.from(byGroup.entries()).map(([group, pts]) => ({
    key: `career.lusr_rating.${group}`,
    meta: { gamertag: group },
    datapoints: pts,
  }))
}

interface RankProgressProps {
  summary: CareerSummary | null
  labels: CareerChartsSectionLabels
}

function RankProgressCard({ summary, labels }: RankProgressProps) {
  if (!summary) {
    return <UnavailableCard title={labels.rankProgressTitle} labels={labels} />
  }
  const pct = Math.min(100, Math.round(summary.progress_pct))
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{labels.rankProgressTitle}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-center justify-center gap-2 py-6">
        <div className="text-3xl font-bold" data-testid="career-rank-progress-pct">
          {pct}%
        </div>
        <div className="text-sm text-muted-foreground">
          {summary.rank_label} (#{summary.rank_number})
        </div>
      </CardContent>
    </Card>
  )
}

interface HeroProgressProps {
  heroProgress: HeroProgress | null
  labels: CareerChartsSectionLabels
}

function HeroProgressCard({ heroProgress, labels }: HeroProgressProps) {
  if (!heroProgress || heroProgress.percentage == null) {
    return <UnavailableCard title={labels.heroProgressTitle} labels={labels} />
  }
  const pct = Math.min(100, Math.round(heroProgress.percentage * 100))
  // xp_remaining = XP qui reste pour atteindre Hero. xp_total_required est la cible.
  const acquired = heroProgress.xp_total_required - heroProgress.xp_remaining
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{labels.heroProgressTitle}</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col items-center justify-center gap-2 py-6">
        <div className="text-3xl font-bold" data-testid="career-hero-progress-pct">
          {pct}%
        </div>
        <div className="text-sm text-muted-foreground">
          {acquired.toLocaleString()} / {heroProgress.xp_total_required.toLocaleString()} XP
        </div>
      </CardContent>
    </Card>
  )
}

function UnavailableCard({
  title,
  labels,
}: {
  title: string
  labels: CareerChartsSectionLabels
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <EmptyStateNotice
          title={labels.placeholderUnavailable}
          description={labels.placeholderDescription}
        />
      </CardContent>
    </Card>
  )
}
