/**
 * CombatYieldTimeseries — graphe deux courbes (Sprint 56).
 *
 * Construit côté client depuis les données MatchHistoryRow.
 * Courbes : offensive_conversion (vert) + defensive_resistance (bleu).
 * Lignes de référence p80 tracées en pointillés.
 */
import { Suspense, lazy, useMemo } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { MatchHistoryRow } from '@/lib/api/types'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

/** p80 — miroir des constantes Go combat_yield.go */
const OC_P80 = 0.83
const DR_P80 = 1.59

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface CombatYieldTimeseriesProps {
  rows: MatchHistoryRow[]
}

export function CombatYieldTimeseries({ rows }: CombatYieldTimeseriesProps) {
  const filtered = useMemo(
    () => rows.filter(
      (r) => r.offensive_conversion != null || r.defensive_resistance != null
    ),
    [rows],
  )

  if (filtered.length === 0) {
    return (
      <EmptyStateNotice
        title="Données combat indisponibles"
        description="Les métriques offensive_conversion et defensive_resistance n'ont pas encore été calculées. Lance un backfill pour les générer."
      />
    )
  }

  const dates = filtered.map((r) => r.start_time)
  const ocValues = filtered.map((r) => r.offensive_conversion ?? null)
  const drValues = filtered.map((r) => r.defensive_resistance ?? null)

  const data: Plotly.Data[] = [
    {
      type: 'scatter',
      mode: 'lines+markers',
      name: 'Offensif (OC)',
      x: dates,
      y: ocValues,
      line: { color: '#4ade80', width: 2 },
      marker: { color: '#4ade80', size: 5 },
      connectgaps: false,
      hovertemplate: '%{x|%d/%m/%Y}<br>OC: %{y:.2f}<extra></extra>',
    },
    {
      type: 'scatter',
      mode: 'lines+markers',
      name: 'Défensif (DR)',
      x: dates,
      y: drValues,
      line: { color: '#60a5fa', width: 2 },
      marker: { color: '#60a5fa', size: 5 },
      connectgaps: false,
      hovertemplate: '%{x|%d/%m/%Y}<br>DR: %{y:.2f}<extra></extra>',
    },
    // Ligne de référence p80 offensive
    {
      type: 'scatter',
      mode: 'lines',
      name: `p80 OC (${OC_P80})`,
      x: [dates[0], dates[dates.length - 1]],
      y: [OC_P80, OC_P80],
      line: { color: '#4ade80', width: 1, dash: 'dot' },
      hoverinfo: 'none',
      showlegend: true,
    },
    // Ligne de référence p80 défensive
    {
      type: 'scatter',
      mode: 'lines',
      name: `p80 DR (${DR_P80})`,
      x: [dates[0], dates[dates.length - 1]],
      y: [DR_P80, DR_P80],
      line: { color: '#60a5fa', width: 1, dash: 'dot' },
      hoverinfo: 'none',
      showlegend: true,
    },
  ]

  const layout: Partial<Plotly.Layout> = {
    paper_bgcolor: 'transparent',
    plot_bgcolor: 'transparent',
    font: { color: '#9ca3af', size: 11 },
    margin: { t: 10, r: 10, b: 50, l: 50 },
    xaxis: {
      type: 'date',
      gridcolor: '#374151',
      linecolor: '#4b5563',
      tickformat: '%d/%m',
    },
    yaxis: {
      gridcolor: '#374151',
      linecolor: '#4b5563',
      rangemode: 'tozero',
    },
    legend: {
      orientation: 'h',
      y: -0.15,
      x: 0.5,
      xanchor: 'center',
      font: { size: 10 },
    },
    hovermode: 'x unified',
  }

  return (
    <Suspense fallback={<Spinner size="md" />}>
      <Plot
        data={data}
        layout={layout}
        config={CLEAN_CONFIG}
        style={{ width: '100%', height: 320 }}
        useResizeHandler
      />
    </Suspense>
  )
}
