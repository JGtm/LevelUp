/**
 * SquadEfficiencyChart — deux cartes multi-joueurs de la page Dynamique escouade :
 *   - « Rendement » (indicateur `rendement_offensif` du titre, en % d'une vie) ;
 *   - « Résistance » (indicateur `resistance_defensive`, même unité).
 *
 * Une seule grille par carte, les courbes joueurs SUPERPOSÉES, fenêtre fixe
 * 50…200 % et repère « 1 vie » à 100 % : les deux cartes se lisent avec le même
 * cadre, et deux sessions se comparent entre elles. La définition complète
 * (formule, pivot une vie) vit dans l'aide ⓘ portée par les DEUX titres de carte.
 *
 * Titres sans résistance (Halo 5 : pas de damage_taken, donc pas de DR) : seule
 * la carte Rendement est rendue.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildSquadEfficiencyOption,
  EFFICIENCY_CHART_HEIGHT,
  type EfficiencyChartLabels,
} from './charts/squadEfficiencyChart'

interface EfficiencyLabels extends EfficiencyChartLabels {
  rendementCardTitle: string
  resistanceCardTitle: string
  /** Aide ⓘ : définition des deux indicateurs et du pivot « une vie ». */
  help: string
  noData: string
}

interface SquadEfficiencyChartProps {
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  playerOrder: string[]
  /** gamertag → couleur hex résolue depuis semantic tokens. */
  colorByPlayer: Record<string, string>
  labels: EfficiencyLabels
}

function hasEfficiencyData(pts: SquadPerformanceSeriesPoint[]): boolean {
  return pts.some((p) => p.rendement_offensif != null || p.resistance_defensive != null)
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

  // Résistance disponible ? Halo 5 ne fournit pas damage_taken → l'API ne sert
  // aucune `resistance_defensive` → la carte défensive n'est pas rendue.
  const hasResistance = useMemo(
    () => players.some((p) => (rowsByPlayer[p] ?? []).some((pt) => pt.resistance_defensive != null)),
    [players, rowsByPlayer],
  )

  // 1 entrée / joueur : sert l'état vide + la clé de mémo du ChartCard ; le
  // builder lit rowsByPlayer directement (le ChartCard ignore l'argument série).
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => players.map((p) => ({ key: p, datapoints: rowsByPlayer[p] ?? [] })),
    [players, rowsByPlayer],
  )

  const buildRendement = useCallback(
    () =>
      buildSquadEfficiencyOption(rowsByPlayer, players, {
        metric: 'offensive',
        colorByPlayer,
        labels,
      }),
    [rowsByPlayer, players, colorByPlayer, labels],
  )
  const buildResistance = useCallback(
    () =>
      buildSquadEfficiencyOption(rowsByPlayer, players, {
        metric: 'defensive',
        colorByPlayer,
        labels,
      }),
    [rowsByPlayer, players, colorByPlayer, labels],
  )

  const cardTitle = (text: string) => (
    <span className="flex items-center gap-1.5">
      {text}
      <InfoTooltip content={labels.help} />
    </span>
  )

  return (
    // Rendement + Résistance côte à côte sur desktop (empilés en mobile). En mode
    // sans résistance (Halo 5 : pas de damage_taken), la carte Rendement reste
    // pleine largeur (pas de grille → bloc simple).
    <div className={hasResistance ? 'grid gap-4 md:grid-cols-2' : ''}>
      <ChartCard
        title={cardTitle(labels.rendementCardTitle)}
        series={series}
        buildOption={buildRendement}
        height={EFFICIENCY_CHART_HEIGHT}
        emptyMessage={labels.noData}
      />
      {hasResistance && (
        <ChartCard
          title={cardTitle(labels.resistanceCardTitle)}
          series={series}
          buildOption={buildResistance}
          height={EFFICIENCY_CHART_HEIGHT}
          emptyMessage={labels.noData}
        />
      )}
    </div>
  )
}
