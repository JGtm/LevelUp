/**
 * squadEngagementGapChart — « Écart d'engagement cumulé » par joueur (onglet
 * Dynamique).
 *
 * Pour chaque joueur : cumul de la contribution `(pace_observed − team_expected)
 * × (duration_seconds / 60)` (en ÉVÉNEMENTS) par match, dans l'ordre de la
 * session. 1 série `line` par joueur, couleur via `colorByPlayer`, markLine 0
 * (rythme observé = attendu), PAS d'aire (multi-séries).
 *
 * Report D5 (délégué à `cumulativeSigned` / `engagementGapEvents`) : un match
 * sans résidu ou sans durée ne fait pas avancer le cumul.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getLegendBase,
  getTooltipBase,
} from '@/components/charts/_utils'
import { truncateMap } from '@/lib/charts/matchLabels'
import { cumulativeSigned } from '@/lib/charts/cumulativeSeries'
import { engagementGapEvents } from '@/lib/charts/engagementGap'
import type { SquadEngagementSessionAPI } from '@/lib/api/types'

export interface SquadEngagementGapOpts {
  /** gamertag → couleur hex résolue (getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
}

/**
 * Cumul de l'écart d'engagement d'UN joueur sur la session, délégué au helper
 * générique `cumulativeSigned`. residual[i] = pace_observed[i] − team_expected[i]
 * ; contribution[i] = residual × durations_seconds[i]/60 (événements).
 */
export function cumulativeEngagementGapSeries(
  paceObserved: number[],
  teamExpected: number[],
  durationsSeconds: number[],
): Array<number | null> {
  const n = paceObserved.length
  const contribs = new Array<number | null>(n)
  for (let i = 0; i < n; i++) {
    const residual = paceObserved[i] - (teamExpected[i] ?? 0)
    contribs[i] = engagementGapEvents(residual, durationsSeconds[i])
  }
  return cumulativeSigned(contribs).map((p) => p.cumulative)
}

export function buildSquadEngagementGapOption(
  session: SquadEngagementSessionAPI,
  opts: SquadEngagementGapOpts,
): EChartsCoreOption {
  const n = session.labels.length
  if (n === 0 || session.players.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const mapNames = session.map_names ?? []
  const xLabels = Array.from({ length: n }, (_, i) => {
    const m = mapNames[i]
    return m ? `#${i + 1}\n${truncateMap(m)}` : `#${i + 1}`
  })
  const fmtSigned = (v: number) => (v >= 0 ? `+${Math.round(v)}` : `${Math.round(v)}`)

  const players = session.players
  const series = players.map((player, idx) => {
    const color = opts.colorByPlayer[player.gamertag] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const data = cumulativeEngagementGapSeries(
      player.pace_observed,
      session.team_expected,
      session.durations_seconds,
    )
    return {
      name: player.gamertag,
      type: 'line' as const,
      data,
      lineStyle: { color, width: 2 },
      itemStyle: { color },
      symbol: 'circle' as const,
      symbolSize: 4,
      connectNulls: true,
      // markLine 0 (rythme observé = attendu) rendue une seule fois, sur le premier joueur.
      ...(idx === 0
        ? {
            markLine: {
              silent: true,
              symbol: 'none',
              lineStyle: { color: tc.axisLabel, type: 'dashed' as const, width: 1 },
              label: { show: false },
              data: [{ yAxis: 0 }],
            },
          }
        : {}),
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 8, right: 24, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'line' },
      valueFormatter: (v: unknown) => (typeof v === 'number' ? fmtSigned(v) : '-'),
    },
    legend: { ...getLegendBase(tc), data: players.map((p) => p.gamertag) },
    xAxis: {
      ...axis,
      type: 'category',
      data: xLabels,
      axisLabel: { ...axis.axisLabel, interval: n > 30 ? Math.floor(n / 12) : 0 },
    },
    yAxis: {
      ...axis,
      type: 'value',
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtSigned(v) },
    },
    series,
  }
}
