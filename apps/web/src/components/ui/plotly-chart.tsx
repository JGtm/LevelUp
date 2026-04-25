/**
 * Composant PlotlyChart — rendu de figures Plotly JSON depuis le backend.
 *
 * Encapsule react-plotly.js avec une config propre (pas de mode bar, pas de logo).
 */
import { Suspense, lazy } from 'react'
import type { PlotlyFigurePayload } from '@/lib/api/types'
import { Spinner } from './spinner'

// react-plotly.js est lourd — on le lazy charge
// CJS→ESM interop Vite : m.default est l'objet exports CJS { __esModule, default: Component }
// On doit détecter et déballer le double-default pour obtenir le vrai composant React.
const Plot = lazy(() =>
  import('react-plotly.js').then((m) => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const mod = m.default as any
    return { default: (mod?.__esModule ? mod.default : mod) as typeof m.default }
  }),
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
