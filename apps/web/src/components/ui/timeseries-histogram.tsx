/**
 * TimeseriesHistogram — histogramme générique depuis DistributionBucket[].
 *
 * Usage : distribution K/D (onglet Distributions), kills/match, précision,
 *         score/min, win rate glissant.
 * Construit côté client depuis DistributionBucket[].
 */
import { Suspense, lazy, useMemo } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { DistributionBucket } from '@/lib/api/types'
import { resolveToken, useColorPaletteVersion } from '@/lib/accessibility'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface TimeseriesHistogramProps {
  buckets: DistributionBucket[]
  color?: string
  xAxisLabel?: string
  height?: number
}

const BG = '#1d2328'
const GRID = '#2a3038'
const TEXT = '#9ba3af'

export function TimeseriesHistogram({
  buckets,
  color,
  xAxisLabel,
  height = 280,
}: TimeseriesHistogramProps) {
  const paletteVersion = useColorPaletteVersion()
  const { traces, layout } = useMemo(() => {
    const resolvedColor = color ?? resolveToken('perf-tier-2')
    const labels = buckets.map((b) => `${b.bin_start}–${b.bin_end}`)
    const counts = buckets.map((b) => b.count)

    const traces: Plotly.Data[] = [
      {
        type: 'bar',
        x: labels,
        y: counts,
        marker: { color: resolvedColor, opacity: 0.85 },
        hovertemplate: '%{x}<br>Matchs : <b>%{y}</b><extra></extra>',
      },
    ]

    const layout: Partial<Plotly.Layout> = {
      paper_bgcolor: BG,
      plot_bgcolor: BG,
      margin: { t: 8, b: 52, l: 40, r: 8 },
      height,
      bargap: 0.1,
      font: { color: TEXT, size: 11 },
      xaxis: {
        showgrid: false,
        zeroline: false,
        title: xAxisLabel ? { text: xAxisLabel, font: { color: TEXT, size: 10 } } : undefined,
        tickfont: { color: TEXT, size: 9 },
        tickangle: -30,
      },
      yaxis: {
        showgrid: true,
        gridcolor: GRID,
        zeroline: false,
        title: { text: 'Matchs', font: { color: TEXT, size: 10 } },
        tickfont: { color: TEXT, size: 10 },
      },
    }

    return { traces, layout }
  }, [buckets, color, xAxisLabel, height, paletteVersion])

  if (buckets.length === 0) {
    return (
      <EmptyStateNotice
        title="Données insuffisantes"
        description="Aucun bucket disponible pour cette période."
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
