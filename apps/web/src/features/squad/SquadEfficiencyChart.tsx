/**
 * SquadEfficiencyChart — track unique avec switch joueur.
 *
 * Pattern aligné sur SquadIntensityHeatmapChart : boutons segmentés en haut,
 * un seul ECharts pour le joueur sélectionné. Joueur principal par défaut.
 */
import { useCallback, useMemo, useState, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { useEffectiveHpToKill, substituteHpToken } from '@/lib/damage/effectiveHp'
import { damagePerDeath } from '@/lib/charts/oneLifeDamageGradient'
import {
  buildSquadEfficiencyTrackOption,
  buildSquadRendementMultiOption,
} from './charts/squadEfficiencyChart'

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
  /** Titre du ChartCard (barre de titre du catalogue). Accepte un ReactNode pour un InfoTooltip. */
  title?: ReactNode
  /**
   * Titre alternatif en mode mono-métrique (titre sans résistance, ex. Halo 5) :
   * le graphe bascule en « tous les rendements, 1 courbe / joueur » et adopte ce
   * libellé. Défaut : `title`.
   */
  monoTitle?: ReactNode
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
  title,
  monoTitle,
}: SquadEfficiencyChartProps) {
  const players = useMemo(
    () => playerOrder.filter((p) => rowsByPlayer[p] && hasEfficiencyData(rowsByPlayer[p])),
    [playerOrder, rowsByPlayer],
  )

  // Barème PV-pour-tuer du titre courant (225 Infinite, 115 Halo 5) → repère
  // « 1 vie » et dégradés title-aware (sinon H5 serait jaugé sur 225).
  const hp = useEffectiveHpToKill()
  // Libellé du repère avec le barème du titre injecté (« 1 vie (115) » en H5).
  const refLabel = substituteHpToken(labels.refLabel, hp)

  // Résistance disponible ? (dégâts/mort calculable sur au moins un point). Halo 5
  // ne fournit pas damage_taken → false → bascule en mode mono-métrique multi-joueurs.
  const hasResistance = useMemo(
    () =>
      players.some((p) =>
        (rowsByPlayer[p] ?? []).some((pt) => damagePerDeath(pt.damage_taken, pt.deaths) != null),
      ),
    [players, rowsByPlayer],
  )

  const [selectedPlayer, setSelectedPlayer] = useState<string>(players[0] ?? '')

  // Si la sélection courante n'est plus dans la liste (changement de squad),
  // retomber sur le premier joueur disponible.
  const activePlayer = players.includes(selectedPlayer) ? selectedPlayer : (players[0] ?? '')

  const pts = useMemo(() => rowsByPlayer[activePlayer] ?? [], [rowsByPlayer, activePlayer])

  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => (pts.length > 0 ? [{ key: activePlayer, datapoints: pts }] : []),
    [pts, activePlayer],
  )

  // Les lignes sont colorées par dégradé (efficacité), pas par la couleur du
  // joueur : l'identité du joueur reste portée par le bouton actif + le titre.
  const buildOption = useCallback(
    () =>
      buildSquadEfficiencyTrackOption(pts, {
        rendementLabel: labels.rendementLabel,
        resistanceLabel: labels.resistanceLabel,
        refLabel,
        showXAxis: true,
        oneLife: hp,
      }),
    [pts, labels, refLabel, hp],
  )

  // Mode mono-métrique : toutes les courbes Rendement sur un seul graphe,
  // colorées par joueur, toggle via la légende ECharts. `monoSeries` (1 entrée /
  // joueur) sert l'état vide + la clé de mémo du ChartCard ; le builder lit
  // rowsByPlayer directement (le ChartCard ignore l'argument série).
  const monoSeries = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(
    () => players.map((p) => ({ key: p, datapoints: rowsByPlayer[p] ?? [] })),
    [players, rowsByPlayer],
  )
  const buildMonoOption = useCallback(
    () =>
      buildSquadRendementMultiOption(rowsByPlayer, players, {
        refLabel,
        oneLife: hp,
        colorByPlayer,
        showXAxis: true,
      }),
    [rowsByPlayer, players, refLabel, hp, colorByPlayer],
  )

  // Bascule de représentation : sans résistance (ex. Halo 5), on n'a qu'une
  // métrique → tous les joueurs ensemble plutôt qu'un toggle 1-joueur.
  if (!hasResistance) {
    return (
      <ChartCard
        title={monoTitle ?? title}
        series={monoSeries}
        buildOption={buildMonoOption}
        height={TRACK_HEIGHT}
        emptyMessage={labels.noData}
      />
    )
  }

  // Pas de `return null` quand aucun joueur n'a de données : ChartCard rend son
  // emptyMessage (labels.noData) dans le bloc titré.
  return (
    <div className="space-y-2">
      {players.length > 1 && (
        <div className="flex flex-wrap justify-end gap-1">
          {players.map((player) => {
            const accentHex = colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
            return (
              <button
                key={player}
                type="button"
                onClick={() => setSelectedPlayer(player)}
                className={[
                  'inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs transition-colors',
                  player === activePlayer
                    ? 'border-primary bg-primary text-primary-foreground'
                    : 'border-input bg-background hover:bg-muted',
                ].join(' ')}
              >
                <span aria-hidden style={{ background: accentHex, width: 8, height: 8, display: 'inline-block', flexShrink: 0 }} />
                {player}
              </button>
            )
          })}
        </div>
      )}
      <ChartCard
        title={title}
        series={series}
        buildOption={buildOption}
        height={TRACK_HEIGHT}
        emptyMessage={labels.noData}
      >
        {/* Légende des lignes : sous le graphe, centrée, dans le même bloc
            (border-t footer du ChartCard) plutôt qu'au-dessus de la carte. */}
        {series.length > 0 && (
          <div className="flex flex-wrap items-center justify-center gap-x-5 gap-y-1 border-t border-border px-3 py-2 text-xs text-muted-foreground">
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
              {refLabel}
            </span>
          </div>
        )}
      </ChartCard>
    </div>
  )
}
