/**
 * TimeseriesKdaBars — graphe K/D/A par match (barres groupées + ratio K/D ligne).
 *
 * Usage : onglet Résumé — timeline K/D/Assists par match coloré par outcome.
 * Construit côté client depuis TimeseriesMatchRow[].
 */
import { Suspense, lazy, useMemo } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { TimeseriesMatchRow } from '@/lib/api/types'
import { resolveToken, useColorPaletteVersion } from '@/lib/accessibility'
import { outcomeScale } from '@/lib/accessibility/scales'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface TimeseriesKdaBarsProps {
  rows: TimeseriesMatchRow[]
  height?: number
}

const BG = '#1d2328'
const GRID = '#2a3038'
const TEXT = '#9ba3af'
const OUTCOME_INT_KEY: Record<number, string> = { 2: 'win', 1: 'draw', 3: 'loss', 4: 'dnf' }

function outcomeColor(outcome: number | null): string {
  const key = outcome != null ? OUTCOME_INT_KEY[outcome] : null
  const token = key ? outcomeScale(key) : null
  return token ? resolveToken(token) : resolveToken('outcome-draw')
}

export function TimeseriesKdaBars({ rows, height = 320 }: TimeseriesKdaBarsProps) {
  const paletteVersion = useColorPaletteVersion()
  const { data: fieldMappings } = useFieldMappings()
  const killsLabel = fieldMappings?.fields['kills']?.label ?? 'Kills'
  const deathsLabel = fieldMappings?.fields['deaths']?.label ?? 'Morts'
  const { traces, layout } = useMemo(() => {
    const xs = rows.map((r) => r.start_time)

    const killsTrace: Plotly.Data = {
      type: 'bar',
      name: killsLabel,
      x: xs,
      y: rows.map((r) => r.kills),
      marker: {
        color: rows.map((r) => `${outcomeColor(r.outcome)}cc`),
      },
      hovertemplate: `%{x|%d/%m/%Y}<br>${killsLabel} : <b>%{y}</b><extra></extra>`,
    }

    const deathsTrace: Plotly.Data = {
      type: 'bar',
      name: deathsLabel,
      x: xs,
      y: rows.map((r) => -r.deaths),
      marker: { color: `${resolveToken('outcome-loss')}44` },
      hovertemplate: `%{x|%d/%m/%Y}<br>${deathsLabel} : <b>%{customdata}</b><extra></extra>`,
      customdata: rows.map((r) => r.deaths),
    }

    const kdLine: Plotly.Data = {
      type: 'scatter',
      mode: 'lines',
      name: 'K/D',
      x: xs,
      y: rows.map((r) => (r.deaths > 0 ? r.kills / r.deaths : r.kills)),
      yaxis: 'y2',
      line: { color: resolveToken('perf-tier-2'), width: 1.5 },
      hovertemplate: '%{x|%d/%m/%Y}<br>K/D : <b>%{y:.2f}</b><extra></extra>',
    }

    const layout: Partial<Plotly.Layout> = {
      paper_bgcolor: BG,
      plot_bgcolor: BG,
      margin: { t: 8, b: 48, l: 40, r: 40 },
      height,
      barmode: 'relative',
      font: { color: TEXT, size: 11 },
      xaxis: {
        showgrid: false,
        zeroline: false,
        tickformat: '%d/%m',
        tickfont: { color: TEXT, size: 9 },
      },
      yaxis: {
        showgrid: true,
        gridcolor: GRID,
        zeroline: true,
        zerolinecolor: GRID,
        title: { text: 'Kills / Morts', font: { color: TEXT, size: 10 } },
        tickfont: { color: TEXT, size: 10 },
      },
      yaxis2: {
        overlaying: 'y',
        side: 'right',
        showgrid: false,
        title: { text: 'K/D', font: { color: resolveToken('perf-tier-2'), size: 10 } },
        tickfont: { color: resolveToken('perf-tier-2'), size: 10 },
        rangemode: 'tozero',
      },
      legend: {
        font: { color: TEXT, size: 10 },
        bgcolor: 'transparent',
        orientation: 'h',
        y: -0.18,
      },
    }

    return { traces: [killsTrace, deathsTrace, kdLine], layout }
  }, [rows, height, paletteVersion, killsLabel, deathsLabel])

  if (rows.length === 0) {
    return (
      <EmptyStateNotice
        title="Aucun match"
        description="Aucun match disponible pour cette période."
      />
    )
  }

  return (
    <Suspense fallback={<Spinner size="sm" />}>
      <Plot
        data={traces}
        layout={layout}
        config={CLEAN_CONFIG}
        style={{ width: '100%', height: `${height}px` }}
      />
    </Suspense>
  )
}
