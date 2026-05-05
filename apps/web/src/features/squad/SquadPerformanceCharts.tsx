/**
 * SquadPerformanceCharts — wrapper teammates.16 : 8 sous-charts performance.
 *
 * Spec : .ai/charts_specs/teammates/16_trio_performance_charts.yaml (étendue
 *        à 8 charts par demande utilisateur).
 *
 * Sous-charts dans l'ordre :
 *   1+2. Frags / Morts (butterfly) + Assistances (dual-grid)
 *   3.   Précision (pleine largeur)
 *   4+5. FDA / KDA + Durée de vie moyenne (dual-grid)
 *   6+7. Performance + Rang / MMR (dual-grid)
 *   8+9. Folie meurtrière max + Tirs à la tête / Frags parfaits (dual-grid)
 *
 * Layout dual-grid : < 14 matchs → côte à côte, sinon → empilé.
 * Sur mobile (< 768px) → toujours empilé.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  buildDualGridOption,
  dualPanelHeight,
  DUAL_LAYOUT_THRESHOLD,
  type DualLayout,
} from '@/components/charts/_utils'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildHsPerfectOption,
  buildKillsDeathsButterflyOption,
  buildPerformanceLineOption,
  buildTeamMMROption,
} from './charts/squadPerformanceLineCharts'

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
}

interface SquadPerformanceChartsProps {
  rowsByPlayer: Record<string, SquadPerformanceSeriesPoint[]>
  /** Ordre stable des joueurs (main d'abord, puis coéquipiers). */
  playerOrder: string[]
  /** gamertag → couleur hex. */
  colorByPlayer: Record<string, string>
  labels: I18nLabels
}

const SUBCHART_HEIGHT = 280

export function SquadPerformanceCharts({
  rowsByPlayer,
  playerOrder,
  colorByPlayer,
  labels,
}: SquadPerformanceChartsProps) {
  const isMobile = useMediaQuery('(max-width: 768px)')

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

  const layout: DualLayout =
    isMobile || xMatchLabels.length >= DUAL_LAYOUT_THRESHOLD ? 'stacked' : 'side-by-side'
  const dualH = dualPanelHeight(layout)

  const commonOpts = { colorByPlayer, playerOrder, xLabels: xMatchLabels }

  const buildKillsAssists = useCallback(
    () =>
      buildDualGridOption(
        buildKillsDeathsButterflyOption(rowsByPlayer, {
          ...commonOpts,
          killsLabel: labels.killsLabel,
          deathsLabel: labels.deathsLabel,
        }),
        buildPerformanceLineOption(rowsByPlayer, {
          ...commonOpts,
          metric: 'assists',
          decimals: 0,
          chartType: 'bar',
        }),
        labels.killsDeathsTitle,
        labels.assistsTitle,
        layout,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, layout, labels],
  )

  const buildAccuracy = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        ...commonOpts,
        metric: 'accuracy',
        decimals: 1,
        unitSuffix: ' %',
        scale: 100,
        chartType: 'bar',
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildKdaAvgLife = useCallback(
    () =>
      buildDualGridOption(
        buildPerformanceLineOption(rowsByPlayer, {
          ...commonOpts,
          metric: 'kda',
          decimals: 2,
          chartType: 'bar',
          complementBelowValue: 1,
        }),
        buildPerformanceLineOption(rowsByPlayer, {
          ...commonOpts,
          metric: 'avg_life_seconds',
          decimals: 1,
          unitSuffix: ' s',
          chartType: 'bar',
        }),
        labels.kdaTitle,
        labels.avgLifeTitle,
        layout,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, layout, labels],
  )

  const buildPerfRank = useCallback(
    () =>
      buildDualGridOption(
        buildPerformanceLineOption(rowsByPlayer, {
          ...commonOpts,
          metric: 'performance_score',
          decimals: 1,
          chartType: 'bar',
          showPerformanceZones: true,
        }),
        buildTeamMMROption(rowsByPlayer, {
          ...commonOpts,
          mmrLabel: labels.mmrLabel,
        }),
        labels.performanceTitle,
        labels.rankTitle,
        layout,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, layout, labels],
  )

  const buildSpreeHsPerfect = useCallback(
    () =>
      buildDualGridOption(
        buildPerformanceLineOption(rowsByPlayer, {
          ...commonOpts,
          metric: 'max_killing_spree',
          decimals: 0,
          chartType: 'bar',
        }),
        buildHsPerfectOption(rowsByPlayer, {
          ...commonOpts,
          hsLabel: labels.hsLabel,
          perfectLabel: labels.perfectLabel,
        }),
        labels.maxSpreeTitle,
        labels.hsPerfectTitle,
        layout,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, layout, labels],
  )

  if (series.length === 0) return null

  return (
    <div className="space-y-4" data-testid="squad-performance-charts">
      <ChartCard series={series} buildOption={buildKillsAssists} height={dualH} />
      <ChartCard
        title={labels.accuracyTitle}
        series={series}
        buildOption={buildAccuracy}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard series={series} buildOption={buildKdaAvgLife} height={dualH} />
      <ChartCard series={series} buildOption={buildPerfRank} height={dualH} />
      <ChartCard series={series} buildOption={buildSpreeHsPerfect} height={dualH} />
    </div>
  )
}
