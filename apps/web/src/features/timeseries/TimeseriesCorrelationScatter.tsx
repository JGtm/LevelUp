/**
 * TimeseriesCorrelationScatter — picker des paires de corrélation +
 * scatter ECharts coloré par outcome (win / loss / unknown).
 *
 * Phase 3 P3.B : remplace l'ancien wrapper Plotly `TimeseriesScatter`. Le
 * mapping label → titre (axes X/Y) reste local au composant car les paires
 * ne sont pas (toutes) des FieldKey canoniques. Quand un libellé canonique
 * existe (kills, deaths, accuracy, kda, kdr, team_mmr, enemy_mmr), il est
 * résolu via `useFieldMappings`.
 */
import { useMemo, useState } from 'react'

import { ScatterChart } from '@/components/charts/ScatterChart'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { CorrelationDataPair } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { correlationPointsToSeries } from './seriesAdapters'

export interface TimeseriesCorrelationScatterProps {
  points: CorrelationDataPair[]
  height?: number
  /** Libellés résolus i18n pour les noms de séries (win/loss/unknown). */
  outcomeLabels: { win: string; loss: string; unknown: string }
  /** Libellé pour l'état "données insuffisantes". */
  emptyTitle?: string
  emptyDescription?: string
}

interface LabelConfig {
  xLabel: string
  yLabel: string
  title: string
}

function buildLabelConfigs(
  fieldMappings?: { fields: Record<string, { label: string }> },
): Record<string, LabelConfig> {
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  const kills = labelOf('kills')
  const deaths = labelOf('deaths')
  const accuracy = labelOf('accuracy')
  const kda = labelOf('kda')
  const kdr = labelOf('kdr')
  const teamMMR = labelOf('team_mmr')
  const enemyMMR = labelOf('enemy_mmr')
  return {
    kills_vs_kd: { xLabel: kills, yLabel: kdr, title: `${kills} → ${kdr}` },
    lifespan_vs_kills: {
      xLabel: 'Durée de vie (s)',
      yLabel: kills,
      title: `Durée de vie → ${kills}`,
    },
    accuracy_vs_kda: {
      xLabel: `${accuracy} (%)`,
      yLabel: kda,
      title: `${accuracy} → ${kda}`,
    },
    lifespan_vs_deaths: {
      xLabel: 'Durée de vie (s)',
      yLabel: deaths,
      title: `Durée de vie → ${deaths}`,
    },
    kills_vs_deaths: {
      xLabel: kills,
      yLabel: deaths,
      title: `${kills} → ${deaths}`,
    },
    mmr_team_vs_enemy: {
      xLabel: teamMMR,
      yLabel: enemyMMR,
      title: `${teamMMR} vs ${enemyMMR}`,
    },
  }
}

export function TimeseriesCorrelationScatter({
  points,
  height = 320,
  outcomeLabels,
  emptyTitle,
  emptyDescription,
}: TimeseriesCorrelationScatterProps) {
  const availableLabels = useMemo(() => {
    // P7.1 (revue 2026-04-29) : DTO `label` (composite) → metric_x_key/metric_y_key.
    // On reconstruit le composite côté front pour préserver le filtrage existant.
    const seen = new Set<string>()
    for (const p of points) seen.add(`${p.metric_x_key}_vs_${p.metric_y_key}`)
    return Array.from(seen)
  }, [points])

  const [activeLabel, setActiveLabel] = useState<string>(
    () => availableLabels[0] ?? 'kills_vs_kd',
  )
  const { data: fieldMappings } = useFieldMappings()
  const labelConfigs = useMemo(() => buildLabelConfigs(fieldMappings), [fieldMappings])

  const cfg: LabelConfig = labelConfigs[activeLabel] ?? {
    xLabel: 'X',
    yLabel: 'Y',
    title: activeLabel,
  }

  const series = useMemo(
    () => correlationPointsToSeries(points, activeLabel, outcomeLabels),
    [points, activeLabel, outcomeLabels],
  )

  if (points.length === 0) {
    return (
      <EmptyStateNotice
        title={emptyTitle ?? 'Données insuffisantes'}
        description={emptyDescription ?? 'Aucune corrélation disponible.'}
      />
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-1">
        {availableLabels.map((lbl) => (
          <button
            key={lbl}
            type="button"
            onClick={() => setActiveLabel(lbl)}
            className={[
              'rounded px-2 py-0.5 text-xs font-medium transition-colors',
              activeLabel === lbl
                ? 'bg-info/20 text-info ring-1 ring-info/40'
                : 'bg-white/5 text-muted-foreground hover:bg-white/10',
            ].join(' ')}
          >
            {labelConfigs[lbl]?.title ?? lbl}
          </button>
        ))}
      </div>
      <ScatterChart
        series={series}
        height={height}
        xAxisLabel={cfg.xLabel}
        yAxisLabel={cfg.yLabel}
        seriesColorTokens={{
          'outcome.win': 'outcome-win',
          'outcome.loss': 'outcome-loss',
          'outcome.unknown': 'divergent-neutral',
        }}
      />
    </div>
  )
}
