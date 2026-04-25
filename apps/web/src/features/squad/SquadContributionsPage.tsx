/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 * Consomme le contexte SquadContext fourni par SquadLayout.
 */
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { TeammateRow, TeammateKPIs, PlotlyFigurePayload } from '@/lib/api/types'
import { useSquadContext } from './SquadLayout'
import { resolveToken, getSeriesColors } from '@/lib/accessibility'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'

// ─── Helper graphique ─────────────────────────────────────────────────────────

const SERIES_TOKENS = ['narrative-dominant', 'perf-tier-3', 'perf-tier-1'] as const

function buildRadarChart(
  rows: TeammateRow[],
  soloRef: TeammateKPIs | null,
  fieldMappings?: FieldMappingsResponse,
): PlotlyFigurePayload {
  // Phase D plan multi-titres : libellés axes du radar issus du backend TOML
  // avec fallback sur les valeurs FR locales si l'endpoint est absent.
  const winRate = fieldMappings?.fields['win_rate']?.label ?? 'Taux de victoire'
  const kdr = fieldMappings?.fields['kdr']?.label ?? 'K/D'
  const kills = fieldMappings?.fields['kills']?.label ?? 'Kills'
  const assists = fieldMappings?.fields['assists']?.label ?? 'Assists'
  const accuracy = fieldMappings?.fields['accuracy']?.label ?? 'Précision'
  const axes = [winRate, kdr, `${kills}/partie`, `${assists}/partie`, accuracy]
  const norm = (v: number | null, max: number) =>
    v != null ? Math.min(100, (v / max) * 100) : 0

  const makeVals = (k: TeammateKPIs) => [
    k.win_rate * 100,
    norm(k.kd_ratio, 3),
    norm(k.kills_per_game, 20),
    norm(k.assists_per_game, 10),
    norm(k.accuracy, 1) * 100,
  ]

  const colors = getSeriesColors(rows.length, [...SERIES_TOKENS])
  const traces: PlotlyFigurePayload['data'] = rows.map((row, i) => {
    const vals = makeVals(row.with_kpis)
    return {
      type: 'scatterpolar',
      name: `Avec ${row.gamertag}`,
      r: [...vals, vals[0]],
      theta: [...axes, axes[0]],
      fill: 'toself',
      marker: { color: colors[i] },
      line: { color: colors[i] },
    }
  })

  if (soloRef) {
    const vals = makeVals(soloRef)
    const refColor = resolveToken('perf-tier-2')
    traces.push({
      type: 'scatterpolar',
      name: 'Solo ref',
      r: [...vals, vals[0]],
      theta: [...axes, axes[0]],
      fill: 'toself',
      opacity: 0.4,
      marker: { color: refColor },
      line: { color: refColor, dash: 'dot' },
    })
  }

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

// ─── Composant ────────────────────────────────────────────────────────────────

export function SquadContributionsPage() {
  const { selectedRows, soloReference } = useSquadContext()
  const { data: fieldMappings } = useFieldMappings()

  const chart =
    selectedRows.length > 0 ? buildRadarChart(selectedRows, soloReference, fieldMappings) : null

  return (
    <Card>
      <CardContent className="pt-4">
        {selectedRows.length === 0 ? (
          <EmptyStateNotice
            title="Comparaison inactive"
            description="Sélectionne au moins un coéquipier pour afficher les contributions."
          />
        ) : chart ? (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Profil de contribution normalisé (violet) vs ta référence solo (cyan pointillé).
            </p>
            <PlotlyChart figure={chart} />
          </div>
        ) : (
          <EmptyStateNotice
            title="Contributions indisponibles"
            description="Le radar de contribution n'a pas pu être calculé pour la sélection en cours."
          />
        )}
      </CardContent>
    </Card>
  )
}
