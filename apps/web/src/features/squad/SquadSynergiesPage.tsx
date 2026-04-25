/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 * Consomme le contexte SquadContext fourni par SquadLayout.
 */
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { TeammateRow, TeammateKPIs, PlotlyFigurePayload } from '@/lib/api/types'
import { useSquadContext } from './SquadLayout'
import { buildHsPkChart } from './charts/hsPkChart'
import { buildTimelineChart } from './charts/timelineChart'
import { buildHeatmapChart } from './charts/heatmapChart'
import { getSeriesColors, resolveToken } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'

// ─── Helpers graphiques ───────────────────────────────────────────────────────

const CHART_COLORS = getSeriesColors(3, ['narrative-dominant', 'perf-tier-3', 'divergent-pos'])

function buildSynergiesChart(
  rows: TeammateRow[],
  soloRef: TeammateKPIs | null,
  fieldMappings?: FieldMappingsResponse,
): PlotlyFigurePayload {
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  const winRate = labelOf('win_rate', 'Taux de victoire')
  const kdr = labelOf('kdr', 'K/D')
  const kills = labelOf('kills', 'Kills')
  const assists = labelOf('assists', 'Assists')
  const metrics = [winRate, kdr, `${kills}/partie`, `${assists}/partie`]
  const extract = (k: TeammateKPIs) => [
    k.win_rate * 100,
    k.kd_ratio ?? 0,
    k.kills_per_game ?? 0,
    k.assists_per_game ?? 0,
  ]

  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => ({
    type: 'bar',
    name: `Avec ${row.gamertag}`,
    x: metrics,
    y: extract(row.with_kpis),
    marker: { color: CHART_COLORS[i % CHART_COLORS.length] },
  }))

  if (soloRef)
    traces.push({
      type: 'bar',
      name: 'Référence solo',
      x: metrics,
      y: extract(soloRef),
      marker: { color: resolveToken('divergent-neutral') },
    })

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

// ─── Composant ────────────────────────────────────────────────────────────────

export function SquadSynergiesPage() {
  const { selectedRows, soloReference, pageData } = useSquadContext()
  const { data: fieldMappings } = useFieldMappings()

  const chart =
    selectedRows.length > 0 ? buildSynergiesChart(selectedRows, soloReference, fieldMappings) : null

  const hsPkChart = selectedRows.length > 0 ? buildHsPkChart(selectedRows) : null
  const timelineChart =
    pageData?.timeseries && pageData.timeseries.length > 0
      ? buildTimelineChart(pageData.timeseries)
      : null
  const heatmapChart =
    pageData?.map_breakdown && pageData.map_breakdown.length > 0
      ? buildHeatmapChart(pageData.map_breakdown)
      : null

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="pt-4">
          {selectedRows.length === 0 ? (
            <EmptyStateNotice
              title="Comparaison inactive"
              description="Sélectionne au moins un coéquipier pour afficher les synergies."
            />
          ) : chart ? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">
                Comparaison de tes stats <em>avec</em> chaque coéquipier vs ta référence solo.
              </p>
              <PlotlyChart figure={chart} />
            </div>
          ) : (
            <EmptyStateNotice
              title="Synergies indisponibles"
              description="Le graphique de synergie n'a pas pu être construit avec les données actuelles."
            />
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
