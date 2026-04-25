/**
 * TimeseriesLineChart — courbe(s) multi-séries sur des CumulativePoints.
 *
 * Usage : cumul K/D, cumul net kills, rolling K/D (onglet Cumul)
 *         EWMA K/D (onglet Forme), score/min (onglet Intensité).
 * Construit côté client depuis CumulativePoint[].
 */
import { Suspense, lazy, useMemo } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { CumulativePoint } from '@/lib/api/types'
import { resolveToken, useColorPaletteVersion } from '@/lib/accessibility'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface TimeseriesLineSeries {
  name: string
  points: CumulativePoint[]
  color: string
  dash?: 'solid' | 'dash' | 'dot'
  fill?: 'none' | 'tozeroy'
}

export interface TimeseriesLineChartProps {
  series: TimeseriesLineSeries[]
  yAxisLabel?: string
  referenceY?: number
  referenceLabel?: string
  height?: number
}

const BG = '#1d2328'
const GRID = '#2a3038'
const TEXT = '#9ba3af'

export function TimeseriesLineChart({
  series,
  yAxisLabel,
  referenceY,
  referenceLabel,
  height = 300,
}: TimeseriesLineChartProps) {
  const paletteVersion = useColorPaletteVersion()
  const { traces, layout } = useMemo(() => {
    const traces: Plotly.Data[] = series.map((s) => ({
      type: 'scatter',
      mode: 'lines',
      name: s.name,
      x: s.points.map((p) => p.start_time),
      y: s.points.map((p) => p.value),
      line: { color: s.color, width: 2, dash: s.dash ?? 'solid' },
      fill: s.fill ?? 'none',
      fillcolor: s.fill ? `${s.color}22` : undefined,
      hovertemplate: `%{x|%d/%m/%Y}<br>${s.name} : <b>%{y:.2f}</b><extra></extra>`,
    }))

    if (referenceY !== undefined) {
      traces.push({
        type: 'scatter',
        mode: 'lines',
        name: referenceLabel ?? 'Référence',
        x: series[0]?.points.map((p) => p.start_time) ?? [],
        y: series[0]?.points.map(() => referenceY) ?? [],
        line: { color: resolveToken('divergent-neutral'), width: 1, dash: 'dot' },
        hoverinfo: 'skip',
      } as Plotly.Data)
    }

    const layout: Partial<Plotly.Layout> = {
      paper_bgcolor: BG,
      plot_bgcolor: BG,
      margin: { t: 8, b: 48, l: 48, r: 12 },
      height,
      font: { color: TEXT, size: 11 },
      xaxis: {
        showgrid: false,
        zeroline: false,
        tickformat: '%d/%m',
        tickfont: { color: TEXT, size: 10 },
      },
      yaxis: {
        showgrid: true,
        gridcolor: GRID,
        zeroline: true,
        zerolinecolor: GRID,
        title: yAxisLabel ? { text: yAxisLabel, font: { color: TEXT, size: 10 } } : undefined,
        tickfont: { color: TEXT, size: 10 },
      },
      legend: {
        font: { color: TEXT, size: 10 },
        bgcolor: 'transparent',
        orientation: 'h',
        y: -0.18,
      },
    }

    return { traces, layout }
  }, [series, yAxisLabel, referenceY, referenceLabel, height, paletteVersion])

  const empty = series.every((s) => s.points.length === 0)
  if (empty) {
    return (
      <EmptyStateNotice
        title="Données insuffisantes"
        description="Aucun point disponible pour cette période."
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
