/**
 * SquadFdaGapCumulativeCard — « Écart cumulé au FDA attendu » (onglet Synergies).
 *
 * Décisions D3/D4/D5 du plan PLAN_EXPECTED_FDA_2026-07 : une courbe cumulée par
 * joueur (FDA réel − FDA attendu, cumul par match_order) + une rangée de pastilles
 * KPI « écart moyen par match » (ex. « +0,7/match »), colorées par joueur (couleurs
 * getSquadPlayerColors, cohérentes pill/combobox).
 *
 * Self-gate `useCapability('expected_stats')` (retour null) : Halo 5 n'a pas
 * d'attendu → carte masquée sans trou de mise en page. Même pattern que les charts
 * du Lot B (SessionFdaGapCumulative / TimeseriesFdaGapTrend).
 */
import { useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import type { SquadText } from './i18n'
import { buildFdaGapCumulativeOption, meanFdaGapPerMatch } from './charts/squadFdaGapChart'

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

export function SquadFdaGapCumulativeCard({
  rowsByPlayer,
  playerOrder,
  colorByPlayer,
  t,
  emptyMessage,
  height = SUBCHART_HEIGHT,
}: Props) {
  const hasExpectedStats = useCapability('expected_stats')

  // Joueurs affichés (ordre stable), restreints à ceux ayant au moins un point.
  const players = useMemo(
    () => playerOrder.filter((p) => (rowsByPlayer[p]?.length ?? 0) > 0),
    [playerOrder, rowsByPlayer],
  )

  // Série sentinelle plate → évite l'empty-state de ChartCard ; le builder relit
  // directement rowsByPlayer (comme SquadPerformanceCharts).
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(() => {
    const merged = players.flatMap((p) => rowsByPlayer[p] ?? [])
    return merged.length > 0 ? [{ key: 'fda-gap-flat', datapoints: merged }] : []
  }, [players, rowsByPlayer])

  const kpis = useMemo(
    () =>
      players.map((player) => ({
        player,
        color: colorByPlayer[player],
        mean: meanFdaGapPerMatch(rowsByPlayer[player] ?? []),
      })),
    [players, rowsByPlayer, colorByPlayer],
  )

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null

  const nf = new Intl.NumberFormat(t.intlLocale, {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
    signDisplay: 'always',
  })

  return (
    <ChartCard
      title={t.fdaGap.title}
      series={series}
      height={height}
      emptyMessage={emptyMessage}
      buildOption={() =>
        buildFdaGapCumulativeOption(rowsByPlayer, {
          colorByPlayer,
          playerOrder: players,
        })
      }
    >
      {kpis.length > 0 && (
        <div className="flex flex-col gap-1.5 border-t border-border px-3 py-2" data-testid="fda-gap-kpis">
          <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {t.fdaGap.averageCaption}
          </span>
          <div className="flex flex-wrap gap-2">
            {kpis.map(({ player, color, mean }) => (
              <span
                key={player}
                className="inline-flex items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2 py-1 text-xs"
              >
                <span
                  className="inline-block h-2 w-2 shrink-0 rounded-full"
                  style={{ backgroundColor: color }}
                  aria-hidden
                />
                <span className="font-medium text-foreground">{player}</span>
                <span className="tabular-nums font-semibold" style={{ color }}>
                  {mean == null ? '—' : `${nf.format(mean)}${t.units.perGame}`}
                </span>
              </span>
            ))}
          </div>
        </div>
      )}
    </ChartCard>
  )
}
