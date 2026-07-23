/**
 * SquadNetLivesChart — « Balance des dégâts cumulée » (onglet Dynamique).
 *
 * Une courbe cumulée par joueur : `netLives = (dégâts infligés − dégâts subis) /
 * PV-pour-tuer`, cumulé par `match_order`, couleur par joueur (getSquadPlayerColors),
 * ligne repère 0 (équilibre).
 *
 * Self-gate `useProvidesDamageTaken()` (capability `damage_taken`, retour null) :
 * un titre sans dégâts subis (ex. Halo 5) ne peut pas calculer la balance → carte
 * masquée sans trou de mise en page (aucun fallback FDA déguisé). Gate title-agnostic
 * (capability, jamais slug==) plutôt que sonder les rows. Même pattern que
 * SquadFdaGapCumulativeCard.
 */
import { useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useEffectiveHpToKill, substituteHpToken, useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import type { SquadText } from './i18n'
import { buildNetLivesCumulativeOption } from './charts/squadNetLivesChart'

const SUBCHART_HEIGHT = 280

interface Props {
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  /** Ordre stable des joueurs (main d'abord, puis coéquipiers). */
  playerOrder: string[]
  /** gamertag → couleur hex (getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  t: SquadText
  emptyMessage?: string
  height?: number
}

export function SquadNetLivesChart({
  rowsByPlayer,
  playerOrder,
  colorByPlayer,
  t,
  emptyMessage,
  height = SUBCHART_HEIGHT,
}: Props) {
  const providesDamageTaken = useProvidesDamageTaken()
  const hp = useEffectiveHpToKill()

  // Joueurs affichés (ordre stable), restreints à ceux ayant au moins un point.
  const players = useMemo(
    () => playerOrder.filter((p) => (rowsByPlayer[p]?.length ?? 0) > 0),
    [playerOrder, rowsByPlayer],
  )

  // Série sentinelle plate → évite l'empty-state de ChartCard ; le builder relit
  // directement rowsByPlayer (comme SquadFdaGapCumulativeCard).
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(() => {
    const merged = players.flatMap((p) => rowsByPlayer[p] ?? [])
    return merged.length > 0 ? [{ key: 'net-lives-flat', datapoints: merged }] : []
  }, [players, rowsByPlayer])

  // Titre sans dégâts subis (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!providesDamageTaken) return null

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {t.netLives.title}
          <InfoTooltip content={substituteHpToken(t.netLives.tooltip, hp)} />
        </span>
      }
      series={series}
      height={height}
      emptyMessage={emptyMessage}
      buildOption={() =>
        buildNetLivesCumulativeOption(rowsByPlayer, {
          colorByPlayer,
          playerOrder: players,
          hp,
        })
      }
    />
  )
}
