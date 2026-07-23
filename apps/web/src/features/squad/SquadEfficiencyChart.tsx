/**
 * SquadEfficiencyChart — deux graphes multi-joueurs :
 *   - « Rendement — dégâts par frag » (métrique damagePerKill) ;
 *   - « Résistance — dégâts par mort » (métrique damagePerDeath).
 *
 * Chaque carte trace UNE courbe par joueur (couleur = joueur), togglable via la
 * légende native ECharts, avec le repère « 1 vie » du titre. Titres sans résistance
 * (ex. Halo 5, pas de damage_taken) : seule la carte Rendement est rendue.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { useEffectiveHpToKill, substituteHpToken } from '@/lib/damage/effectiveHp'
import { damagePerDeath } from '@/lib/charts/oneLifeDamageGradient'
import { buildSquadEfficiencyMultiOption } from './charts/squadEfficiencyChart'

interface EfficiencyLabels {
  rendementCardTitle: string
  resistanceCardTitle: string
  refLabel: string
  noData: string
}

interface SquadEfficiencyChartProps {
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  playerOrder: string[]
  /** gamertag → couleur hex résolue depuis semantic tokens. */
  colorByPlayer: Record<string, string>
  labels: EfficiencyLabels
  /** InfoTooltip optionnel rendu à côté du titre de la carte Rendement. */
  infoTooltip?: ReactNode
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
  infoTooltip,
}: SquadEfficiencyChartProps) {
  const players = useMemo(
    () => playerOrder.filter((p) => rowsByPlayer[p] && hasEfficiencyData(rowsByPlayer[p])),
    [playerOrder, rowsByPlayer],
  )

  // Barème PV-pour-tuer du titre courant (225 Infinite, 115 Halo 5) → repère
  // « 1 vie » title-aware (sinon H5 serait jaugé sur 225).
  const hp = useEffectiveHpToKill()
  // Libellé du repère avec le barème du titre injecté (« 1 vie (115) » en H5).
  const refLabel = substituteHpToken(labels.refLabel, hp)

  // Résistance disponible ? (dégâts/mort calculable sur au moins un point). Halo 5
  // ne fournit pas damage_taken → false → seule la carte Rendement est rendue.
  const hasResistance = useMemo(
    () =>
      players.some((p) =>
        (rowsByPlayer[p] ?? []).some((pt) => damagePerDeath(pt.damage_taken, pt.deaths) != null),
      ),
    [players, rowsByPlayer],
  )

  // 1 entrée / joueur : sert l'état vide + la clé de mémo du ChartCard ; le builder
  // lit rowsByPlayer directement (le ChartCard ignore l'argument série).
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => players.map((p) => ({ key: p, datapoints: rowsByPlayer[p] ?? [] })),
    [players, rowsByPlayer],
  )

  const buildRendement = useCallback(
    () =>
      buildSquadEfficiencyMultiOption(rowsByPlayer, players, {
        metric: 'damagePerKill',
        refLabel,
        oneLife: hp,
        colorByPlayer,
        showXAxis: true,
      }),
    [rowsByPlayer, players, refLabel, hp, colorByPlayer],
  )
  const buildResistance = useCallback(
    () =>
      buildSquadEfficiencyMultiOption(rowsByPlayer, players, {
        metric: 'damagePerDeath',
        refLabel,
        oneLife: hp,
        colorByPlayer,
        showXAxis: true,
      }),
    [rowsByPlayer, players, refLabel, hp, colorByPlayer],
  )

  return (
    <div className="space-y-4">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {labels.rendementCardTitle}
            {infoTooltip}
          </span>
        }
        series={series}
        buildOption={buildRendement}
        height={TRACK_HEIGHT}
        emptyMessage={labels.noData}
      />
      {hasResistance && (
        <ChartCard
          title={labels.resistanceCardTitle}
          series={series}
          buildOption={buildResistance}
          height={TRACK_HEIGHT}
          emptyMessage={labels.noData}
        />
      )}
    </div>
  )
}
