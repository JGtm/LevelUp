/**
 * ChartFrame — coque visuelle alignée sur ChartCard (charts/ChartCard.tsx).
 *
 * Utilisé sur les onglets KPIs + Cumul pour les charts à `buildOption` custom
 * (TimeseriesKdaTrend, TimeseriesKdaDensity, TimeseriesScatterWithTrend, etc.)
 * afin que le visuel soit identique aux cards de la page Escouade Synergies.
 */
import type { ReactNode } from 'react'

export interface ChartFrameProps {
  title?: ReactNode
  children: ReactNode
  className?: string
}

export function ChartFrame({ title, children, className = '' }: ChartFrameProps) {
  return (
    <div
      className={`relative rounded-lg border border-border bg-card ${className}`}
      data-testid="chart-frame"
    >
      {title && (
        <div className="border-b border-border px-3 py-2 text-sm font-medium">
          {title}
        </div>
      )}
      <div className="p-3">{children}</div>
    </div>
  )
}
