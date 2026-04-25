/**
 * TimeseriesHeatmap — heatmap Jour × Heure d'intensité de jeu.
 *
 * Usage : onglet Intensité — visualise les plages horaires de la semaine
 *         colorées par K/D moyen ou nombre de matchs.
 * Construit côté client depuis IntensityHeatmapPoint[].
 */
import { Suspense, lazy, useMemo } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { IntensityHeatmapPoint } from '@/lib/api/types'
import { buildDivergentColorscale, buildOrdinalColorscale, useColorPaletteVersion } from '@/lib/accessibility'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface TimeseriesHeatmapProps {
  data: IntensityHeatmapPoint[]
  colorBy?: 'count' | 'avg_kd'
  height?: number
}

const DAYS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
const BG = '#1d2328'
const TEXT = '#9ba3af'

export function TimeseriesHeatmap({
  data,
  colorBy = 'count',
  height = 300,
}: TimeseriesHeatmapProps) {
  const paletteVersion = useColorPaletteVersion()
  const { traces, layout } = useMemo(() => {
    // Construire une matrice 7 jours × 24 heures.
    const matrix: (number | null)[][] = Array.from({ length: 7 }, () =>
      Array(24).fill(null),
    )
    const hoverMatrix: string[][] = Array.from({ length: 7 }, () =>
      Array(24).fill(''),
    )

    for (const pt of data) {
      const d = Math.min(Math.max(pt.day_of_week, 0), 6)
      const h = Math.min(Math.max(pt.hour, 0), 23)
      matrix[d][h] = colorBy === 'count' ? pt.count : pt.avg_kd
      hoverMatrix[d][h] =
        `${DAYS[d]} ${String(h).padStart(2, '0')}h<br>Matchs : ${pt.count}<br>K/D moy : ${pt.avg_kd.toFixed(2)}`
    }

    const hours = Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}h`)

    const traces: Plotly.Data[] = [
      {
        type: 'heatmap',
        z: matrix,
        x: hours,
        y: DAYS,
        colorscale: colorBy === 'count'
          ? buildOrdinalColorscale(['perf-tier-5', 'perf-tier-4', 'perf-tier-3', 'perf-tier-2', 'perf-tier-1'])
          : buildDivergentColorscale('divergent-neg', 'divergent-neutral', 'divergent-pos'),
        text: hoverMatrix,
        hovertemplate: '%{text}<extra></extra>',
        showscale: true,
        colorbar: {
          tickfont: { color: TEXT, size: 9 },
          outlinecolor: 'transparent',
          thickness: 12,
        },
        xgap: 2,
        ygap: 2,
      } as Plotly.Data,
    ]

    const layout: Partial<Plotly.Layout> = {
      paper_bgcolor: BG,
      plot_bgcolor: BG,
      margin: { t: 8, b: 40, l: 42, r: 60 },
      height,
      font: { color: TEXT, size: 11 },
      xaxis: {
        showgrid: false,
        zeroline: false,
        tickfont: { color: TEXT, size: 9 },
        tickangle: -45,
      },
      yaxis: {
        showgrid: false,
        zeroline: false,
        tickfont: { color: TEXT, size: 10 },
        autorange: 'reversed',
      },
    }

    return { traces, layout }
  }, [data, colorBy, height, paletteVersion])

  if (data.length === 0) {
    return (
      <EmptyStateNotice
        title="Données insuffisantes"
        description="Aucune donnée d'intensité disponible."
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
