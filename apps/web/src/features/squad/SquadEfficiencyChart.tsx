/**
 * SquadEfficiencyChart — track unique avec switch joueur.
 *
 * Pattern aligné sur SquadIntensityHeatmapChart : boutons segmentés en haut,
 * un seul ECharts pour le joueur sélectionné. Joueur principal par défaut.
 */
import { useCallback, useMemo, useState } from 'react'
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

const TRACK_HEIGHT = 320

function hasEfficiencyData(pts: SquadPerformanceSeriesPoint[]): boolean {
  return pts.some((p) => p.rendement_offensif !== undefined || p.resistance_defensive !== undefined)
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

  const [selectedPlayer, setSelectedPlayer] = useState<string>(players[0] ?? '')

  // Si la sélection courante n'est plus dans la liste (changement de squad),
  // retomber sur le premier joueur disponible.
  const activePlayer = players.includes(selectedPlayer) ? selectedPlayer : (players[0] ?? '')

  const pts = useMemo(() => rowsByPlayer[activePlayer] ?? [], [rowsByPlayer, activePlayer])
  const color = colorByPlayer[activePlayer] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée

  // Échelle Y globale : max sur tous les joueurs → stable au switch.
  const globalYMax = useMemo(() => {
    let max = 0
    for (const p of Object.values(rowsByPlayer).flat()) {
      if (p.rendement_offensif !== undefined) max = Math.max(max, p.rendement_offensif)
      if (p.resistance_defensive !== undefined) max = Math.max(max, p.resistance_defensive)
    }
    return Math.ceil(max * 2) / 2 // arrondi au demi supérieur
  }, [rowsByPlayer])

  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => (pts.length > 0 ? [{ key: activePlayer, datapoints: pts }] : []),
    [pts, activePlayer],
  )

  const buildOption = useCallback(
    () =>
      buildSquadEfficiencyTrackOption(pts, {
        color,
        rendementLabel: labels.rendementLabel,
        resistanceLabel: labels.resistanceLabel,
        refLabel: labels.refLabel,
        showXAxis: true,
        yMax: globalYMax,
      }),
    [pts, color, labels, globalYMax],
  )

  if (players.length === 0) return null

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-x-5 gap-y-1 px-1 text-xs" style={{ color: TEXT_COLOR }}>
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
        <div className="ml-auto flex flex-wrap justify-end gap-1">
          {players.map((player) => (
            <button
              key={player}
              type="button"
              onClick={() => setSelectedPlayer(player)}
              className={[
                'rounded-md border px-2.5 py-1 text-xs transition-colors',
                player === activePlayer
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-input bg-background hover:bg-muted',
              ].join(' ')}
            >
              {player}
            </button>
          ))}
        </div>
      </div>
      <ChartCard
        series={series}
        buildOption={buildOption}
        height={TRACK_HEIGHT}
        emptyMessage={labels.noData}
      />
    </div>
  )
}
