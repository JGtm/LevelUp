/**
 * SquadFdaGapCumulativeCard — « Écart cumulé au FDA attendu » (onglet Dynamique).
 *
 * Décisions D3/D5 du plan PLAN_EXPECTED_FDA_2026-07 : une courbe cumulée par
 * joueur (FDA réel − FDA attendu, cumul par match_order), couleurs
 * getSquadPlayerColors. Forme revue par le lot L1 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : montée à côté de « Balance des
 * dégâts cumulée » (même abscisse), valeur de fin au bout de chaque courbe, légende
 * en bas et centrée ; les pastilles « écart moyen par match » sont retirées (aucun
 * texte de synthèse sous le graphe).
 *
 * Self-gate `useCapability('expected_stats')` (retour null) : Halo 5 n'a pas
 * d'attendu → carte masquée sans trou de mise en page. Même pattern que les charts
 * du Lot B (SessionFdaGapCumulative / TimeseriesFdaGapTrend).
 */
import { useCallback, useMemo } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { FdaGapTooltipText } from '@/components/charts/FdaGapTooltipText'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import type { SquadText } from './i18n'
import { buildFdaGapCumulativeOption } from './charts/squadFdaGapChart'

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

  // `buildOption` fait partie des dépendances du useMemo de ChartCard : une
  // lambda écrite dans le JSX est neuve à chaque rendu, donc l'option ECharts
  // est rebâtie et l'animation d'entrée REJOUÉE même à donnée inchangée.
  // Déclaré avant le retour anticipé (règle des hooks).
  const intlLocale = t.intlLocale
  const buildOption = useCallback(
    () =>
      buildFdaGapCumulativeOption(rowsByPlayer, { colorByPlayer, playerOrder: players, intlLocale }),
    [rowsByPlayer, colorByPlayer, players, intlLocale],
  )

  // Titre sans attendu (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!hasExpectedStats) return null

  return (
    <ChartCard
      title={
        <span className="flex items-center gap-1.5">
          {t.fdaGap.title}
          <InfoTooltip
            content={<FdaGapTooltipText locale={intlLocale.startsWith('en') ? 'en' : 'fr'} />}
          />
        </span>
      }
      series={series}
      height={height}
      emptyMessage={emptyMessage}
      // fluid : la carte s'étire à la hauteur de la rangée (grid align-items:stretch)
      // et le graphe la remplit → centré verticalement, même hauteur que « Balance des
      // dégâts cumulée », sa voisine dans SquadDynamiquePage.
      fluid
      buildOption={buildOption}
    />
  )
}
