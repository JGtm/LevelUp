/**
 * SquadEfficiencyChart — deux cartes multi-joueurs de la page Dynamique escouade :
 *   - « Rendement — écart à la frontière élite » (indicateur OC du titre) ;
 *   - « Résistance — écart à la frontière élite » (indicateur DR du titre).
 *
 * Chaque carte empile UNE PISTE PAR JOUEUR (décision d'artefact « C ») : l'écart
 * signé en % à la frontière élite du titre, remplissage vert au-dessus de zéro /
 * rouge en dessous, axe symétrique partagé par toutes les pistes. Les deux
 * frontières viennent du contrat (`TitleSummary.offensive_conversion_p80` et
 * `defensive_resistance_p80`) — aucune constante de gameplay dupliquée côté web.
 *
 * Titres sans résistance (Halo 5 : pas de damage_taken, donc pas de DR) : seule la
 * carte Rendement est rendue.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { useDefensiveResistanceP80, useOffensiveConversionP80 } from '@/lib/damage/effectiveHp'
import {
  buildSquadEfficiencyTracksOption,
  tracksChartHeight,
  type EfficiencyTrackLabels,
} from './charts/squadEfficiencyChart'

interface EfficiencyLabels extends EfficiencyTrackLabels {
  rendementCardTitle: string
  resistanceCardTitle: string
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

function hasEfficiencyData(pts: SquadPerformanceSeriesPoint[]): boolean {
  return pts.some((p) => p.rendement_offensif != null || p.resistance_defensive != null)
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

  // Frontières élite du titre COURANT (title-aware, servies par le bootstrap) :
  // 0.90 / 1.264 en OC selon le titre, 1.65 en DR (Infinite).
  const ocP80 = useOffensiveConversionP80()
  const drP80 = useDefensiveResistanceP80()

  // Résistance disponible ? Halo 5 ne fournit pas damage_taken → l'API ne sert
  // aucune `resistance_defensive` → la carte défensive n'est pas rendue.
  const hasResistance = useMemo(
    () => players.some((p) => (rowsByPlayer[p] ?? []).some((pt) => pt.resistance_defensive != null)),
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
      buildSquadEfficiencyTracksOption(rowsByPlayer, players, {
        metric: 'offensive',
        eliteP80: ocP80,
        colorByPlayer,
        labels,
      }),
    [rowsByPlayer, players, ocP80, colorByPlayer, labels],
  )
  const buildResistance = useCallback(
    () =>
      buildSquadEfficiencyTracksOption(rowsByPlayer, players, {
        metric: 'defensive',
        eliteP80: drP80,
        colorByPlayer,
        labels,
      }),
    [rowsByPlayer, players, drP80, colorByPlayer, labels],
  )

  // Hauteur pilotée par le nombre de pistes : les deux cartes en ont autant, donc
  // elles restent alignées côte à côte sur la grille desktop.
  const height = tracksChartHeight(players.length)

  return (
    // Rendement + Résistance côte à côte sur desktop (empilés en mobile). En mode
    // sans résistance (Halo 5 : pas de damage_taken), la carte Rendement reste
    // pleine largeur (pas de grille → bloc simple).
    <div className={hasResistance ? 'grid gap-4 md:grid-cols-2' : ''}>
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {labels.rendementCardTitle}
            {infoTooltip}
          </span>
        }
        series={series}
        buildOption={buildRendement}
        height={height}
        emptyMessage={labels.noData}
      />
      {hasResistance && (
        <ChartCard
          title={labels.resistanceCardTitle}
          series={series}
          buildOption={buildResistance}
          height={height}
          emptyMessage={labels.noData}
        />
      )}
    </div>
  )
}
