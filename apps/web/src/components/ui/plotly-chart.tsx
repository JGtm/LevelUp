/**
 * Composant PlotlyChart — rendu de figures Plotly JSON depuis le backend.
 *
 * Encapsule react-plotly.js avec une config propre (pas de mode bar, pas de logo).
 */
import { Suspense, lazy } from 'react'
import type { PlotlyFigurePayload } from '@/lib/api/types'
import { Spinner } from './spinner'

// react-plotly.js est lourd — on le lazy charge
const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

interface PlotlyChartProps {
  figure: PlotlyFigurePayload
  className?: string
  style?: React.CSSProperties
}

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export function PlotlyChart({ figure, className = '', style }: PlotlyChartProps) {
  return (
    <Suspense fallback={<Spinner size="md" />}>
      <Plot
        data={figure.data as Plotly.Data[]}
        layout={figure.layout as Partial<Plotly.Layout>}
        config={CLEAN_CONFIG}
        className={className}
        style={{ width: '100%', ...style }}
        useResizeHandler
      />
    </Suspense>
  )
}
