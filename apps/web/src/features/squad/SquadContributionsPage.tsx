/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Affiche un
 * radar normalisé du profil de chaque coéquipier sélectionné.
 *
 * Multi-titres : axes du radar dérivés de SQUAD_RADAR_METRICS (FieldKeys
 * canoniques) + filtrage des keys absentes du fields.toml du titre courant.
 * Strings UI via getSquadText.
 *
 * Distingue 3 empty kinds (no_selection / invalid_selection / no_chart_data)
 * comme SquadSynergiesPage pour rendre les défauts diagnosticables.
 */
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateRow, PlotlyFigurePayload } from '@/lib/api/types'
import { useSquadContext } from './SquadLayout'
import { getSeriesColors } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_RADAR_METRICS, type SquadMetric } from './metrics'

const SERIES_TOKENS = ['narrative-dominant', 'perf-tier-3', 'perf-tier-1'] as const

function metricLabel(
  m: SquadMetric,
  mappings: FieldMappingsResponse | undefined,
  perGameSuffix: string,
): string {
  const base = mappings?.fields[m.key]?.label ?? m.key
  return m.format === 'per_game' ? `${base}${perGameSuffix}` : base
}

function availableMetrics(
  metrics: readonly SquadMetric[],
  mappings: FieldMappingsResponse | undefined,
): SquadMetric[] {
  if (!mappings) return [...metrics]
  return metrics.filter((m) => !!mappings.fields[m.key])
}

interface RadarArgs {
  rows: TeammateRow[]
  metrics: SquadMetric[]
  axes: string[]
  withGamertagLabel: (gt: string) => string
}

function buildRadarChart({
  rows,
  metrics,
  axes,
  withGamertagLabel,
}: RadarArgs): PlotlyFigurePayload | null {
  if (rows.length === 0 || metrics.length === 0) return null

  const colors = getSeriesColors(rows.length, [...SERIES_TOKENS])
  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => {
    const vals = metrics.map((m) => m.extract(row.with_kpis) ?? 0)
    return {
      type: 'scatterpolar',
      name: withGamertagLabel(row.gamertag),
      r: [...vals, vals[0]],
      theta: [...axes, axes[0]],
      fill: 'toself',
      marker: { color: colors[i] },
      line: { color: colors[i] },
    }
  })

  return {
    data: traces,
    layout: {
      height: 380,
      polar: { radialaxis: { visible: true, range: [0, 100] } },
      legend: { orientation: 'h', x: 0.5, xanchor: 'center', y: -0.08 },
      margin: { l: 40, r: 40, t: 30, b: 60 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
    },
  }
}

export function SquadContributionsPage() {
  const { selectedRows, confirmedGamertags } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)

  const metrics = availableMetrics(SQUAD_RADAR_METRICS, mappings)
  const axes = metrics.map((m) => metricLabel(m, mappings, t.units.perGame))

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  const chart = hasRows
    ? buildRadarChart({
        rows: selectedRows,
        metrics,
        axes,
        withGamertagLabel: t.table.withTeammate,
      })
    : null

  let emptyContent: React.ReactNode = null
  if (!hasSelection) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.noSelectionTitle}
        description={t.empty.noSelectionDescription}
      />
    )
  } else if (!hasRows) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.invalidSelectionTitle}
        description={t.empty.invalidSelectionDescription}
      />
    )
  } else if (!chart) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.noChartTitle}
        description={t.empty.noChartDescription}
      />
    )
  }

  return (
    <Card>
      <CardContent className="pt-4">
        {emptyContent ?? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">{t.contributions.description}</p>
            <PlotlyChart figure={chart!} />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
