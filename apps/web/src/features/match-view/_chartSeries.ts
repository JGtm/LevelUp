/**
 * Helpers ChartSeries pour MatchViewPage : KD timeline, Tug-of-war, Cadence.
 *
 * P8.4 (revue 2026-04-29) : extraits de MatchViewPage.tsx (~40L).
 * 2026-05-02 : ajout helpers tugOfWarStackedSeries + cadenceSeriesWithGamertags
 *              pour match_view.10 et match_view.11.
 */
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { MatchScoreboardRow, MatchViewCadence } from '@/lib/api/types'

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
  bin_end?: number
  team_kills?: number
  enemy_kills?: number
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

const TEAM_LABEL = 'Mon équipe'
const ENEMY_LABEL = 'Adverses'

/** Composant tugOfWarStackedSeries — bars divergentes par bin
 *  (Mon équipe positive en haut, Adverses négatives en bas).
 *
 * Catégories formatées en `m:ss`.
 */
export function tugOfWarStackedSeries(
  bins: TugOfWarBin[],
): ChartSeries<ChartPointStacked>[] {
  if (bins.length === 0) return []
  return [
    {
      key: 'match_view.combat.tug_of_war.divergent',
      datapoints: bins.map((b) => ({
        category: formatBinSeconds(b.bin_start),
        components: {
          [TEAM_LABEL]: b.team_kills ?? 0,
          [ENEMY_LABEL]: -(b.enemy_kills ?? 0),
        },
      })),
    },
  ]
}

export const TUG_OF_WAR_LABELS = { team: TEAM_LABEL, enemy: ENEMY_LABEL }

/** Convertit cadence (xuid → kills) en cadence (gamertag → kills) lisible. */
export function cadenceSeriesWithGamertags(
  cadence: MatchViewCadence | null | undefined,
  scoreboard: MatchScoreboardRow[],
): ChartSeries<ChartPointStacked>[] {
  if (!cadence || cadence.datapoints.length === 0) return []
  const xuidToGT = new Map<string, string>()
  for (const r of scoreboard) {
    xuidToGT.set(r.xuid, r.gamertag || r.xuid)
  }
  return [
    {
      key: cadence.key,
      labelKey: cadence.label_key,
      datapoints: cadence.datapoints.map((dp) => {
        const components: Record<string, number> = {}
        for (const [xuid, kills] of Object.entries(dp.components)) {
          if (kills === 0) continue
          components[xuidToGT.get(xuid) ?? xuid] = kills
        }
        return { category: dp.category, components }
      }),
      meta: cadence.meta,
    },
  ]
}

/** Format seconds → "m:ss" (ex 75 → "1:15"). */
export function formatBinSeconds(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.max(0, Math.floor(seconds % 60))
  return `${m}:${s.toString().padStart(2, '0')}`
}
