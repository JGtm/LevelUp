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
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateRow } from '@/lib/api/types'
import { useSquadContext } from './SquadContext'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_RADAR_METRICS, type SquadMetric } from './metrics'
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'

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

function buildRadarSeries(
  rows: TeammateRow[],
  metrics: SquadMetric[],
  withGamertagLabel: (gt: string) => string,
): RadarSeriesPayload[] {
  return rows.map((row) => ({
    key: row.gamertag,
    axes: metrics.map((m) => {
      const v = m.extract(row.with_kpis) ?? 0
      return { axis: m.key, value: v, raw: v }
    }),
    meta: { gamertag: withGamertagLabel(row.gamertag) },
  }))
}

export function SquadContributionsPage() {
  const { selectedRows, confirmedGamertags } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)

  const metrics = availableMetrics(SQUAD_RADAR_METRICS, mappings)
  const axisLabels = Object.fromEntries(
    metrics.map((m) => [m.key, metricLabel(m, mappings, t.units.perGame)]),
  )

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  const radarSeries = hasRows
    ? buildRadarSeries(selectedRows, metrics, t.table.withTeammate)
    : []

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
  } else if (radarSeries.length === 0 || metrics.length === 0) {
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
            <RadarChart
              series={radarSeries}
              height={380}
              axisLabels={axisLabels}
            />
          </div>
        )}
      </CardContent>
    </Card>
  )
}
