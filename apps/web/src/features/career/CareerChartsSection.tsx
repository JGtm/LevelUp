/**
 * CareerChartsSection — figures Plotly de la page Carrière.
 */
import { Suspense, lazy } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'
import type { CareerCharts } from '@/lib/api/types'
import type { PlotlyFigurePayload } from '@/lib/api/types'

const Plot = lazy(() => import('react-plotly.js').then((m) => ({ default: m.default })))

const CLEAN_CONFIG = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud'] as Plotly.ModeBarDefaultButtons[],
  responsive: true,
}

function PlotCard({ title, figure }: { title: string; figure: PlotlyFigurePayload | null }) {
  if (!figure) return null
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <Suspense fallback={<Spinner size="md" />}>
          <Plot
            data={figure.data as Plotly.Data[]}
            layout={{ ...figure.layout, autosize: true } as Partial<Plotly.Layout>}
            config={CLEAN_CONFIG}
            style={{ width: '100%', minHeight: 200 }}
            useResizeHandler
          />
        </Suspense>
      </CardContent>
    </Card>
  )
}

interface Props {
  charts: CareerCharts
}

export function CareerChartsSection({ charts }: Props) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <PlotCard title="Progression du rang" figure={charts.rank_progress_gauge} />
      <PlotCard title="Progression Héros" figure={charts.hero_progress_gauge} />
      <PlotCard title="Historique XP" figure={charts.xp_history_figure} />
      {charts.lusr_rating_figure && (
        <PlotCard title="Rating LUSR" figure={charts.lusr_rating_figure} />
      )}
    </div>
  )
}
