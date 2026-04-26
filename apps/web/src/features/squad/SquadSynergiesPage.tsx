/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Distingue 3
 * états "vides" pour rendre le bug "Comparaison inactive même après
 * sélection" diagnosticable :
 *  - no_selection : aucun coéquipier confirmé.
 *  - invalid_selection : confirmedGts > 0 mais teammates renvoyé vide
 *    (gamertag pas dans LoadTopTeammates côté backend).
 *  - no_chart_data : données présentes mais le chart Plotly retourne null.
 *
 * Multi-titres : les libellés métier (FieldKey) viennent de useFieldMappings ;
 * les strings UI viennent de getSquadText. La liste des métriques affichées
 * est filtrée sur les FieldKeys présents dans le titre courant (graceful
 * degradation pour les titres minimalistes).
 */
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateRow, PlotlyFigurePayload } from '@/lib/api/types'
import { useSquadContext } from './SquadContext'
import { buildHsPkChart } from './charts/hsPkChart'
import { buildTimelineChart } from './charts/timelineChart'
import { buildHeatmapChart } from './charts/heatmapChart'
import { getSeriesColors } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_SYNERGY_METRICS, SQUAD_HSPK_METRICS, type SquadMetric } from './metrics'

const CHART_COLORS = getSeriesColors(3, ['narrative-dominant', 'perf-tier-3', 'divergent-pos'])

/**
 * Compose le label affiché d'une métrique : libellé canonique du FieldKey
 * (résolu via fieldMappings) + suffixe d'unité quand pertinent.
 */
function metricLabel(
  m: SquadMetric,
  mappings: FieldMappingsResponse | undefined,
  perGameSuffix: string,
): string {
  const base = mappings?.fields[m.key]?.label ?? m.key
  return m.format === 'per_game' ? `${base}${perGameSuffix}` : base
}

/**
 * Filtre les métriques absentes du fields.toml du titre courant pour la
 * dégradation gracieuse multi-titres.
 */
function availableMetrics(
  metrics: readonly SquadMetric[],
  mappings: FieldMappingsResponse | undefined,
): SquadMetric[] {
  if (!mappings) return [...metrics]
  return metrics.filter((m) => !!mappings.fields[m.key])
}

interface BarChartArgs {
  rows: TeammateRow[]
  metrics: SquadMetric[]
  labels: string[]
  withGamertagLabel: (gt: string) => string
}

function buildSynergiesChart({
  rows,
  metrics,
  labels,
  withGamertagLabel,
}: BarChartArgs): PlotlyFigurePayload | null {
  if (rows.length === 0 || metrics.length === 0) return null

  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => ({
    type: 'bar',
    name: withGamertagLabel(row.gamertag),
    x: labels,
    y: metrics.map((m) => m.extract(row.with_kpis) ?? 0),
    marker: { color: CHART_COLORS[i % CHART_COLORS.length] },
  }))

  return {
    data: traces,
    layout: {
      barmode: 'group',
      height: 320,
      margin: { l: 40, r: 20, t: 20, b: 60 },
      legend: { orientation: 'h', x: 0, y: -0.2 },
      plot_bgcolor: 'rgba(0,0,0,0)',
      paper_bgcolor: 'rgba(0,0,0,0)',
    },
  }
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

  // Construction conditionnelle des graphes pour ne pas dépenser de cycles
  // en cas de sélection vide / invalide.
  const chart =
    hasRows
      ? buildSynergiesChart({
          rows: selectedRows,
          metrics,
          labels,
          withGamertagLabel: t.table.withTeammate,
        })
      : null
  // HS/PK : libellés composés via fieldMappings + perGameSuffix.
  // L'absence de l'un des FieldKeys (`headshot_kills`, `perfect_kills`)
  // côté titre courant fait passer son label en fallback "key" — la trace
  // reste affichée mais avec un libellé brut, signalant la dégradation.
  const hsLabel =
    (mappings?.fields[SQUAD_HSPK_METRICS.hs.key]?.label ?? SQUAD_HSPK_METRICS.hs.key) +
    t.units.perGame
  const pkLabel =
    (mappings?.fields[SQUAD_HSPK_METRICS.pk.key]?.label ?? SQUAD_HSPK_METRICS.pk.key) +
    t.units.perGame
  const hsPkChart = hasRows
    ? buildHsPkChart({
        rows: selectedRows,
        hsMetric: SQUAD_HSPK_METRICS.hs,
        pkMetric: SQUAD_HSPK_METRICS.pk,
        hsLabel,
        pkLabel,
        title: t.charts.hsPkTitle,
      })
    : null
  const timelineChart =
    pageData?.timeseries && pageData.timeseries.length > 0
      ? buildTimelineChart({
          points: pageData.timeseries,
          title: t.charts.timelineTitle,
          perfName: t.charts.timelinePerfName,
          winRateName: t.charts.timelineWinRateName,
          perfAxis: t.charts.timelinePerfAxis,
          winRateAxis: t.charts.timelineWinRateAxis,
        })
      : null
  // mapLabelOf : résout le `map_ui` brut backend vers le libellé localisé
  // de assets.toml (kind = "map") du titre courant. Fallback sur l'ID brut
  // pour les cartes pas (encore) mappées — graceful degradation.
  const mapAssets = mappings?.assets?.['map']
  const mapLabelOf = (mapId: string): string => mapAssets?.[mapId]?.label ?? mapId
  const heatmapChart =
    pageData?.map_breakdown && pageData.map_breakdown.length > 0
      ? buildHeatmapChart({
          rows: pageData.map_breakdown,
          title: t.charts.heatmapTitle,
          winAxis: t.charts.heatmapWinAxis,
          matchesLabel: t.charts.heatmapMatchesLabel,
          mapLabelOf,
        })
      : null

  // ── Empty states 3-états ────────────────────────────────────────────────
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
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-4">
          {emptyContent ?? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">{t.synergies.description}</p>
              <PlotlyChart figure={chart!} />
            </div>
          )}
        </CardContent>
      </Card>

      {hsPkChart && (
        <Card>
          <CardContent className="pt-4">
            <PlotlyChart figure={hsPkChart} />
          </CardContent>
        </Card>
      )}

      {timelineChart && (
        <Card>
          <CardContent className="pt-4">
            <PlotlyChart figure={timelineChart} />
          </CardContent>
        </Card>
      )}

      {heatmapChart && (
        <Card>
          <CardContent className="pt-4">
            <PlotlyChart figure={heatmapChart} />
          </CardContent>
        </Card>
      )}
    </div>
  )
}
