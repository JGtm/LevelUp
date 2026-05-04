/**
 * SquadEfficiencyChart — une track ECharts par joueur.
 *
 * Affiche le rendement offensif (trait plein) et la résistance défensive
 * (trait pointillé) normalisés par leur P80, empilés verticalement.
 * Seul le dernier track affiche l'axe X (numéro de match).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { TEXT_COLOR } from '@/components/charts/_utils'
import { buildSquadEfficiencyTrackOption } from './charts/squadEfficiencyChart'

interface EfficiencyLabels {
  rendementLabel: string
  resistanceLabel: string
  refLabel: string
  noData: string
}

interface SquadEfficiencyChartProps {
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  playerOrder: string[]
  /** gamertag → couleur hex résolue depuis semantic tokens. */
  colorByPlayer: Record<string, string>
  labels: EfficiencyLabels
}

const TRACK_HEIGHT = 200

function hasEfficiencyData(pts: SquadPerformanceSeriesPoint[]): boolean {
  return pts.some((p) => p.rendement_offensif !== undefined || p.resistance_defensive !== undefined)
}

interface TrackProps {
  player: string
  pts: SquadPerformanceSeriesPoint[]
  color: string
  labels: EfficiencyLabels
  showXAxis: boolean
}

function EfficiencyTrack({ player, pts, color, labels, showXAxis }: TrackProps) {
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => (pts.length > 0 ? [{ key: player, datapoints: pts }] : []),
    [pts, player],
  )
  const buildOption = useCallback(
    () =>
      buildSquadEfficiencyTrackOption(pts, {
        color,
        rendementLabel: labels.rendementLabel,
        resistanceLabel: labels.resistanceLabel,
        refLabel: labels.refLabel,
        showXAxis,
      }),
    [pts, color, labels, showXAxis],
  )

  return (
    <ChartCard
      title={player}
      series={series}
      buildOption={buildOption}
      height={TRACK_HEIGHT}
      emptyMessage={labels.noData}
    />
  )
}

export function SquadEfficiencyChart({
  rowsByPlayer,
  playerOrder,
  colorByPlayer,
  labels,
}: SquadEfficiencyChartProps) {
  const players = useMemo(
    () => playerOrder.filter((p) => rowsByPlayer[p] && hasEfficiencyData(rowsByPlayer[p])),
    [playerOrder, rowsByPlayer],
  )

  if (players.length === 0) return null

  const lastIdx = players.length - 1

  return (
    <div className="space-y-1">
      <div
        className="flex flex-wrap items-center gap-x-5 gap-y-1 px-1 pb-1 text-xs"
        style={{ color: TEXT_COLOR }}
      >
        <span className="flex items-center gap-1.5">
          <svg aria-hidden="true" width="20" height="4">
            <line x1="0" y1="2" x2="20" y2="2" stroke="currentColor" strokeWidth="2" />
          </svg>
          {labels.rendementLabel}
        </span>
        <span className="flex items-center gap-1.5">
          <svg aria-hidden="true" width="20" height="4">
            <line
              x1="0"
              y1="2"
              x2="20"
              y2="2"
              stroke="currentColor"
              strokeWidth="2"
              strokeDasharray="4 2"
              strokeOpacity="0.55"
            />
          </svg>
          {labels.resistanceLabel}
        </span>
        <span className="flex items-center gap-1.5">
          <svg aria-hidden="true" width="20" height="4">
            <line
              x1="0"
              y1="2"
              x2="20"
              y2="2"
              stroke="currentColor"
              strokeWidth="1"
              strokeDasharray="4 2"
            />
          </svg>
          {labels.refLabel}
        </span>
      </div>
      {players.map((player, i) => (
        <EfficiencyTrack
          key={player}
          player={player}
          pts={rowsByPlayer[player] ?? []}
          color={colorByPlayer[player] ?? '#888'} // color-allow: gris structurel pour joueur sans couleur attribuée
          labels={labels}
          showXAxis={i === lastIdx}
        />
      ))}
    </div>
  )
}
