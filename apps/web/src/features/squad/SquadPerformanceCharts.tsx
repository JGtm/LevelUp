/**
 * SquadPerformanceCharts — wrapper teammates.16 : 8 sous-charts performance.
 *
 * Spec : .ai/charts_specs/teammates/16_trio_performance_charts.yaml (étendue
 *        à 8 charts par demande utilisateur).
 *
 * Sous-charts dans l'ordre :
 *   1+2. Frags / Morts (butterfly) + Assistances (paire)
 *   3.   Précision (pleine largeur)
 *   4+5. FDA / KDA + Durée de vie moyenne (paire)
 *   6+7. Performance + Rang / MMR (paire)
 *   8+9. Folie meurtrière max + Tirs à la tête / Frags parfaits (paire)
 *
 * Layout des paires : < 14 matchs → côte à côte (grid-cols-2), sinon → empilé
 * (chaque chart pleine largeur pour respirer avec beaucoup de matchs). Mobile
 * (< 768px) → toujours empilé.
 *
 * Chaque sous-chart est rendu dans son propre ChartCard / instance ECharts :
 * tooltips et légendes restent isolés à leur grille, contrairement à la
 * version dual-grid précédente qui fusionnait les deux dans un canvas unique.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'
import { useProvidesMaxKillingSpree } from '@/lib/damage/effectiveHp'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildHsPerfectOption,
  buildKillsDeathsButterflyOption,
  buildPerformanceLineOption,
  buildTeamMMROption,
} from './charts/squadPerformanceLineCharts'
import { buildFragBreakdownOption } from './charts/squadFragBreakdownChart'
import { SquadToggleLegendChart } from './SquadToggleLegendChart'

interface I18nLabels {
  killsDeathsTitle: string
  killsLabel: string
  deathsLabel: string
  assistsTitle: string
  kdaTitle: string
  accuracyTitle: string
  avgLifeTitle: string
  performanceTitle: string
  maxSpreeTitle: string
  hsPerfectTitle: string
  hsLabel: string
  perfectLabel: string
  rankTitle: string
  mmrLabel: string
  fragBreakdownTitle: string
  meleeLabel: string
  powerWeaponLabel: string
  grenadeLabel: string
  otherLabel: string
}

interface SquadPerformanceChartsProps {
  emptyMessage?: string
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  /** Ordre stable des joueurs (main d'abord, puis coéquipiers). */
  playerOrder: string[]
  /** gamertag → couleur hex. */
  colorByPlayer: Record<string, string>
  labels: I18nLabels
}

const SUBCHART_HEIGHT = 280
/** Au-delà de ce nombre de matchs, les paires basculent en empilé pour donner toute la largeur à chaque chart. */
const PAIR_LAYOUT_THRESHOLD = 14

export function SquadPerformanceCharts({
  rowsByPlayer,
  playerOrder,
  colorByPlayer,
  labels,
  emptyMessage,
}: SquadPerformanceChartsProps) {
  const isMobile = useMediaQuery('(max-width: 768px)')
  // Title-agnostic : la série « Folie meurtrière max » n'est masquée que pour un
  // titre sans events horodatés (provides_max_killing_spree=false). Pour Halo 5
  // le backend CALCULE la spree depuis les events → flag true → série affichée.
  const providesMaxSpree = useProvidesMaxKillingSpree()

  // Cast helper — série sentinelle pour éviter l'empty-state de ChartCard.
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(() => {
    const merged = playerOrder.flatMap((p) => rowsByPlayer[p] ?? [])
    return merged.length > 0 ? [{ key: 'perf-flat', datapoints: merged }] : []
  }, [playerOrder, rowsByPlayer])

  // Labels X enrichis sur 2 lignes : "#1\nTribord", "#2\nBazaar…", etc.
  const xMatchLabels = useMemo(() => {
    const truncate = (s: string, max = 9): string => {
      if (s.length <= max) return s
      const sepIdx = Math.min(
        ...[' ', '-'].map((c) => { const i = s.indexOf(c); return i > 0 ? i : Infinity }),
      )
      if (sepIdx <= max) return `${s.slice(0, sepIdx)}…`
      return `${s.slice(0, max - 1)}…`
    }
    const byOrder = new Map<number, string>()
    for (const pts of Object.values(rowsByPlayer)) {
      for (const pt of pts) {
        if (!byOrder.has(pt.match_order)) {
          const label = pt.map_name
            ? `#${pt.match_order + 1}\n${truncate(pt.map_name)}`
            : `#${pt.match_order + 1}`
          byOrder.set(pt.match_order, label)
        }
      }
    }
    if (byOrder.size === 0) return []
    const maxOrder = Math.max(...byOrder.keys())
    return Array.from({ length: maxOrder + 1 }, (_, i) => byOrder.get(i) ?? `#${i + 1}`)
  }, [rowsByPlayer])

  // Joueurs affichés dans les légendes React (kills/deaths + hs/perfect), dans
  // l'ordre stable, restreints à ceux ayant au moins un point de série.
  const legendPlayers = useMemo(
    () => playerOrder.filter((p) => (rowsByPlayer[p]?.length ?? 0) > 0),
    [playerOrder, rowsByPlayer],
  )

  // DATA-GATE accuracy : pas de capability dédiée → on regarde la donnée. Si
  // aucun point n'a d'accuracy (tout null, cas Halo 5), on NE MONTE PAS la carte
  // « Précision » (pas de carte titrée vide). Reste affichée pour Infinite.
  const hasAccuracy = useMemo(
    () =>
      Object.values(rowsByPlayer).some((pts) =>
        pts.some((p) => p.accuracy != null),
      ),
    [rowsByPlayer],
  )

  const stacked = isMobile || xMatchLabels.length >= PAIR_LAYOUT_THRESHOLD
  const pairClass = stacked ? 'space-y-4' : 'grid grid-cols-1 md:grid-cols-2 gap-4'

  const commonOpts = { colorByPlayer, playerOrder, xLabels: xMatchLabels }

  const buildKillsDeaths = useCallback(
    (hidden: { hiddenPlayers: Set<string>; hiddenTypes: Set<string> }) =>
      buildKillsDeathsButterflyOption(rowsByPlayer, {
        ...commonOpts,
        killsLabel: labels.killsLabel,
        deathsLabel: labels.deathsLabel,
        hiddenPlayers: hidden.hiddenPlayers,
        hiddenTypes: hidden.hiddenTypes,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.killsLabel, labels.deathsLabel],
  )

  const buildAssists = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'assists',
        decimals: 0,
        chartType: 'bar',
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildAccuracy = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'accuracy',
        decimals: 1,
        unitSuffix: ' %',
        // accuracy déjà en 0..100 (match_participants) — pas de scale.
        scale: 1,
        chartType: 'bar',
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildFragBreakdown = useCallback(
    () =>
      buildFragBreakdownOption(rowsByPlayer, {
        playerOrder,
        labels: {
          melee: labels.meleeLabel,
          powerWeapon: labels.powerWeaponLabel,
          grenade: labels.grenadeLabel,
          other: labels.otherLabel,
        },
      }),
    [rowsByPlayer, playerOrder, labels.meleeLabel, labels.powerWeaponLabel, labels.grenadeLabel, labels.otherLabel],
  )

  const buildKda = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'kda',
        decimals: 2,
        chartType: 'bar',
        complementBelowValue: 1,
        redReferenceLineAt: 1,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildAvgLife = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'avg_life_seconds',
        decimals: 1,
        unitSuffix: ' s',
        chartType: 'bar',
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildPerformance = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'performance_score',
        decimals: 1,
        chartType: 'bar',
        showPerformanceZones: true,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildRank = useCallback(
    () =>
      buildTeamMMROption(rowsByPlayer, {
        ...commonOpts,
        mmrLabel: labels.mmrLabel,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.mmrLabel],
  )

  const buildMaxSpree = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'max_killing_spree',
        decimals: 0,
        chartType: 'bar',
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildHsPerfect = useCallback(
    (hidden: { hiddenPlayers: Set<string>; hiddenTypes: Set<string> }) =>
      buildHsPerfectOption(rowsByPlayer, {
        ...commonOpts,
        hsLabel: labels.hsLabel,
        perfectLabel: labels.perfectLabel,
        hiddenPlayers: hidden.hiddenPlayers,
        hiddenTypes: hidden.hiddenTypes,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.hsLabel, labels.perfectLabel],
  )

  // Pas de `return null` quand aucune série : chaque sous-ChartCard affiche son
  // état vide (titre + message) au lieu de faire disparaître toute la grille.
  return (
    <div className="space-y-4" data-testid="squad-performance-charts">
      <div className={pairClass}>
        <SquadToggleLegendChart
          title={labels.killsDeathsTitle}
          series={series}
          players={legendPlayers}
          colorByPlayer={colorByPlayer}
          types={[
            { key: labels.killsLabel, label: labels.killsLabel },
            { key: labels.deathsLabel, label: labels.deathsLabel },
            { key: 'Bonus', label: 'Bonus' },
          ]}
          initialHiddenTypes={new Set(['Bonus'])}
          buildOption={buildKillsDeaths}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
        <ChartCard
          title={labels.assistsTitle}
          series={series}
          buildOption={buildAssists}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
      </div>
      <div className={hasAccuracy ? pairClass : (stacked ? 'space-y-4' : 'grid grid-cols-1 gap-4')}>
        {hasAccuracy && (
          <ChartCard
            title={labels.accuracyTitle}
            series={series}
            buildOption={buildAccuracy}
            height={SUBCHART_HEIGHT}
            emptyMessage={emptyMessage}
          />
        )}
        <ChartCard
          title={labels.fragBreakdownTitle}
          series={series}
          buildOption={buildFragBreakdown}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
      </div>
      <div className={pairClass}>
        <ChartCard
          title={labels.kdaTitle}
          series={series}
          buildOption={buildKda}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
        <ChartCard
          title={labels.avgLifeTitle}
          series={series}
          buildOption={buildAvgLife}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
      </div>
      <div className={pairClass}>
        <ChartCard
          title={labels.performanceTitle}
          series={series}
          buildOption={buildPerformance}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
        <ChartCard
          title={labels.rankTitle}
          series={series}
          buildOption={buildRank}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
      </div>
      <div className={providesMaxSpree ? pairClass : (stacked ? 'space-y-4' : 'grid grid-cols-1 gap-4')}>
        {providesMaxSpree && (
          <ChartCard
            title={labels.maxSpreeTitle}
            series={series}
            buildOption={buildMaxSpree}
            height={SUBCHART_HEIGHT}
            emptyMessage={emptyMessage}
          />
        )}
        <SquadToggleLegendChart
          title={labels.hsPerfectTitle}
          series={series}
          players={legendPlayers}
          colorByPlayer={colorByPlayer}
          types={[
            { key: labels.hsLabel, label: labels.hsLabel },
            { key: labels.perfectLabel, label: labels.perfectLabel },
          ]}
          buildOption={buildHsPerfect}
          height={SUBCHART_HEIGHT}
          emptyMessage={emptyMessage}
        />
      </div>
    </div>
  )
}
