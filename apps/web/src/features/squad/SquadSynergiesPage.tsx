/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Distingue 3
 * états "vides" diagnosticables :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais selectedRows vide.
 *  - no_chart_data : données présentes mais séries vides (métriques filtrées).
 *
 * Multi-titres : libellés métier via useFieldMappings, strings UI via
 * getSquadText. Métriques filtrées sur FieldKeys présents dans le titre
 * courant (graceful degradation).
 */
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateRow } from '@/lib/api/types'
import { useSquadContext } from './SquadContext'
import { buildHsPkSeries } from './charts/hsPkChart'
import { buildTimelineSeries } from './charts/timelineChart'
import { buildHeatmapSeries } from './charts/heatmapChart'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_SYNERGY_METRICS, SQUAD_HSPK_METRICS, type SquadMetric } from './metrics'
import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'

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

function buildSynergiesSeries(
  rows: TeammateRow[],
  metrics: SquadMetric[],
  labels: string[],
  withGamertagLabel: (gt: string) => string,
): ChartSeries<ChartPointStacked>[] {
  if (rows.length === 0 || metrics.length === 0) return []

  const datapoints: ChartPointStacked[] = metrics.map((m, i) => ({
    category: labels[i],
    components: Object.fromEntries(
      rows.map((row) => [withGamertagLabel(row.gamertag), m.extract(row.with_kpis) ?? 0]),
    ),
  }))

  return [{ key: 'synergies', datapoints }]
}

export function SquadSynergiesPage() {
  const { selectedRows, confirmedGamertags, pageData } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)

  const metrics = availableMetrics(SQUAD_SYNERGY_METRICS, mappings)
  const labels = metrics.map((m) => metricLabel(m, mappings, t.units.perGame))

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  const synergieSeries = buildSynergiesSeries(
    hasRows ? selectedRows : [],
    metrics,
    labels,
    t.table.withTeammate,
  )

  const hsLabel =
    (mappings?.fields[SQUAD_HSPK_METRICS.hs.key]?.label ?? SQUAD_HSPK_METRICS.hs.key) +
    t.units.perGame
  const pkLabel =
    (mappings?.fields[SQUAD_HSPK_METRICS.pk.key]?.label ?? SQUAD_HSPK_METRICS.pk.key) +
    t.units.perGame

  const hsPkSeries = hasRows
    ? buildHsPkSeries({
        rows: selectedRows,
        hsMetric: SQUAD_HSPK_METRICS.hs,
        pkMetric: SQUAD_HSPK_METRICS.pk,
        hsLabel,
        pkLabel,
      })
    : []

  const timelineSeries =
    pageData?.timeseries && pageData.timeseries.length > 0
      ? buildTimelineSeries({
          points: pageData.timeseries,
          perfName: t.charts.timelinePerfName,
          winRateName: t.charts.timelineWinRateName,
        })
      : []

  const mapAssets = mappings?.assets?.['map']
  const mapLabelOf = (mapId: string): string => mapAssets?.[mapId]?.label ?? mapId
  const heatmapSeries =
    pageData?.map_breakdown && pageData.map_breakdown.length > 0
      ? buildHeatmapSeries({
          rows: pageData.map_breakdown,
          winAxisLabel: t.charts.heatmapWinAxis,
          mapLabelOf,
        })
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
  } else if (synergieSeries.length === 0) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.noChartTitle}
        description={t.empty.noChartDescription}
      />
    )
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-4">
          {emptyContent ?? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">{t.synergies.description}</p>
              <BarGroupedChart series={synergieSeries} height={320} />
            </div>
          )}
        </CardContent>
      </Card>

      {hsPkSeries.length > 0 && (
        <Card>
          <CardContent className="pt-4">
            <BarGroupedChart
              title={t.charts.hsPkTitle}
              series={hsPkSeries}
              height={280}
            />
          </CardContent>
        </Card>
      )}

      {timelineSeries.length > 0 && (
        <Card>
          <CardContent className="pt-4">
            <TimeseriesLineChart
              title={t.charts.timelineTitle}
              series={timelineSeries}
              xAxisType="category"
              outcomeMarkers={false}
              seriesNameResolver={(s) => (s.meta?.name as string | undefined) ?? s.key}
              height={300}
            />
          </CardContent>
        </Card>
      )}

      {heatmapSeries.length > 0 && (
        <Card>
          <CardContent className="pt-4">
            <Heatmap2DChart
              title={t.charts.heatmapTitle}
              series={heatmapSeries}
              height={160}
              valueRange={[0, 100]}
            />
          </CardContent>
        </Card>
      )}
    </div>
  )
}
