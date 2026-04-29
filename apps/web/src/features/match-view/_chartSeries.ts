/**
 * Helpers ChartSeries pour MatchViewPage : KD timeline + Tug-of-war.
 *
 * P8.4 (revue 2026-04-29) : extraits de MatchViewPage.tsx (~40L).
 */
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'

export interface KDTimelinePoint {
  time_seconds: number
  kills: number
  deaths: number
}

export function kdTimelineSeries(
  points: KDTimelinePoint[],
  labelOf: (key: string, fallback: string) => string,
): ChartSeries<ChartPoint2D>[] {
  return [
    {
      key: 'match_view.combat.kd_timeline.kills',
      meta: { gamertag: labelOf('kills', 'Kills') },
      datapoints: points.map((p) => ({ x: p.time_seconds, y: p.kills })),
    },
    {
      key: 'match_view.combat.kd_timeline.deaths',
      meta: { gamertag: labelOf('deaths', 'Deaths') },
      datapoints: points.map((p) => ({ x: p.time_seconds, y: p.deaths })),
    },
  ]
}

export interface TugOfWarBin {
  bin_start: number
  net_kills: number
}

export function tugOfWarSeries(bins: TugOfWarBin[]): ChartSeries<ChartPoint2D>[] {
  return [
    {
      key: 'match_view.combat.tug_of_war.net_kills',
      meta: { gamertag: 'Kills nets' },
      datapoints: bins.map((b) => ({ x: b.bin_start, y: b.net_kills })),
    },
  ]
}
