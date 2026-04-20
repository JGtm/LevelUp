/**
 * SquadSynergiesPage — onglet Synergies de l'Escouade.
 * Consomme le contexte SquadContext fourni par SquadLayout.
 */
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { TeammateRow, TeammateKPIs, PlotlyFigurePayload } from '@/lib/api/types'
import { useSquadContext } from './SquadLayout'

// ─── Helpers graphiques ───────────────────────────────────────────────────────

const CHART_COLORS = ['#7C3AED', '#F59E0B', '#10B981']

function buildSynergiesChart(
  rows: TeammateRow[],
  soloRef: TeammateKPIs | null,
): PlotlyFigurePayload {
  const metrics = ['Taux de victoire', 'K/D', 'Kills/partie', 'Assists/partie']
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
      marker: { color: '#94A3B8' },
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
  const { selectedRows, soloReference } = useSquadContext()

  const chart =
    selectedRows.length > 0 ? buildSynergiesChart(selectedRows, soloReference) : null

  return (
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
  )
}
