/**
 * SquadPerformanceCharts — wrapper teammates.16 : 8 sous-charts performance.
 *
 * Spec : .ai/charts_specs/teammates/16_trio_performance_charts.yaml (étendue
 *        à 8 charts par demande utilisateur).
 *
 * Sous-charts dans l'ordre :
 *   1. Frags / Morts (butterfly bars)
 *   2. Assistances (line)
 *   3. FDA / KDA (line)
 *   4. Précision (line %)
 *   5. Durée de vie moyenne (line, secondes)
 *   6. Performance (line score 0..100)
 *   7. Folie meurtrière max (line)
 *   8. Tirs à la tête + Frags parfaits (perfect emphasés via line épaisse + area)
 *
 * Tous partagent les mêmes couleurs joueurs (cohérent avec la pill et le
 * combobox) et la même série temporelle backend (matchs partagés ASC).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  buildHsPerfectOption,
  buildKillsDeathsButterflyOption,
  buildPerformanceLineOption,
  buildTeamMMROption,
  type PerformanceMetricKey,
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
  // Cast helper — la série passée à ChartCard est juste un wrapper pour avoir
  // un payload non-vide (évite l'empty-state de ChartCard quand les vraies
  // données sont dans `rowsByPlayer`).
  const series = useMemo<ChartSeries<SquadPerformanceSeriesPoint>[]>(() => {
    const merged = playerOrder.flatMap((p) => rowsByPlayer[p] ?? [])
    return merged.length > 0 ? [{ key: 'perf-flat', datapoints: merged }] : []
  }, [playerOrder, rowsByPlayer])

  // Labels X enrichis sur 2 lignes : "#1\nTribord", "#2\nBazaar…", etc.
  // Noms de carte tronqués à 9 caractères pour éviter le chevauchement.
  const xMatchLabels = useMemo(() => {
    const truncate = (s: string, max = 9): string => {
      if (s.length <= max) return s
      const sepIdx = Math.min(
        ...[' ', '-'].map((c) => { const i = s.indexOf(c); return i > 0 ? i : Infinity }),
      )
      if (sepIdx <= max) return `${s.slice(0, sepIdx)}…` // "Launch Site" → "Launch…", "Couvre-feu" → "Couvre…"
      return `${s.slice(0, max - 1)}…` // "Fragmentation" → "Fragment…"
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

  const buildButterfly = useCallback(
    () =>
      buildKillsDeathsButterflyOption(rowsByPlayer, {
        colorByPlayer,
        playerOrder,
        xLabels: xMatchLabels,
        killsLabel: labels.killsLabel,
        deathsLabel: labels.deathsLabel,
      }),
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.killsLabel, labels.deathsLabel],
  )

  const buildLine = useCallback(
    (metric: PerformanceMetricKey, decimals: number, suffix?: string, scale?: number, complementBelowValue?: number) =>
      buildPerformanceLineOption(rowsByPlayer, {
        colorByPlayer,
        playerOrder,
        xLabels: xMatchLabels,
        metric,
        decimals,
        unitSuffix: suffix,
        scale,
        chartType: 'bar',
        complementBelowValue,
      }),
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildPerformanceZones = useCallback(
    () =>
      buildPerformanceLineOption(rowsByPlayer, {
        colorByPlayer,
        playerOrder,
        xLabels: xMatchLabels,
        metric: 'performance_score',
        decimals: 1,
        chartType: 'bar',
        showPerformanceZones: true,
      }),
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels],
  )

  const buildRankMMR = useCallback(
    () =>
      buildTeamMMROption(rowsByPlayer, {
        colorByPlayer,
        playerOrder,
        xLabels: xMatchLabels,
        mmrLabel: labels.mmrLabel,
      }),
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.mmrLabel],
  )

  const buildHsPerfect = useCallback(
    () =>
      buildHsPerfectOption(rowsByPlayer, {
        colorByPlayer,
        playerOrder,
        xLabels: xMatchLabels,
        hsLabel: labels.hsLabel,
        perfectLabel: labels.perfectLabel,
      }),
    [rowsByPlayer, colorByPlayer, playerOrder, xMatchLabels, labels.hsLabel, labels.perfectLabel],
  )

  if (series.length === 0) return null

  return (
    <div className="space-y-4" data-testid="squad-performance-charts">
      <ChartCard
        title={labels.killsDeathsTitle}
        series={series}
        buildOption={buildButterfly}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard
        title={labels.assistsTitle}
        series={series}
        buildOption={() => buildLine('assists', 0)}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard
        title={labels.kdaTitle}
        series={series}
        buildOption={() => buildLine('kda', 2, undefined, undefined, 1)}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard
        title={labels.accuracyTitle}
        series={series}
        buildOption={() => buildLine('accuracy', 1, ' %', 100)}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard
        title={labels.avgLifeTitle}
        series={series}
        buildOption={() => buildLine('avg_life_seconds', 1, ' s')}
        height={SUBCHART_HEIGHT}
      />
      <div className="grid gap-4 lg:grid-cols-2">
        <ChartCard
          title={labels.performanceTitle}
          series={series}
          buildOption={buildPerformanceZones}
          height={SUBCHART_HEIGHT}
        />
        <ChartCard
          title={labels.rankTitle}
          series={series}
          buildOption={buildRankMMR}
          height={SUBCHART_HEIGHT}
        />
      </div>
      <ChartCard
        title={labels.maxSpreeTitle}
        series={series}
        buildOption={() => buildLine('max_killing_spree', 0)}
        height={SUBCHART_HEIGHT}
      />
      <ChartCard
        title={labels.hsPerfectTitle}
        series={series}
        buildOption={buildHsPerfect}
        height={SUBCHART_HEIGHT}
      />
    </div>
  )
}
