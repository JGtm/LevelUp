/**
 * ChartCard — base reutilisable pour tous les wrappers ECharts.
 *
 * Conformement au PLAN_META_FOUNDATIONS_GO § 5.2.2 : tous les wrappers
 * specialises (TimeseriesLine, BarStacked, Heatmap2D, Radar, ...) etendent
 * cette base via une fonction `buildOption(series) => echarts.EChartsCoreOption`.
 *
 * Responsabilites :
 *   - 4 etats : loading / error / empty / data
 *   - Lazy import de echarts-for-react (reduit le bundle initial ~600KB)
 *   - Theme via tokens couleur (cf. apps/web/src/lib/accessibility/palettes/)
 *   - Cleanup de l'instance ECharts au unmount
 *
 * Le composant est volontairement minimal : aucune logique de chart specifique.
 * Toute la specificite (axes, series, tooltips) vit dans `buildOption` du
 * wrapper consommateur.
 */
import { Suspense, lazy, useMemo, type ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { Spinner } from '@/components/ui/spinner'

// echarts-for-react lazy : evite de payer le cout du bundle echarts (~600KB)
// avant qu'un chart soit reellement rendu.
const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

/**
 * ChartSeries miroir TypeScript du domain.ChartSeries[T] cote Go.
 * Le type T est le type d'un datapoint (ex. ChartPoint2D, ChartPointStacked).
 */
export interface ChartSeries<T = unknown> {
  key: string
  labelKey?: string
  colorToken?: string
  datapoints: T[]
  meta?: Record<string, unknown>
}

export interface ChartCardProps<T = unknown> {
  /** Titre i18n-resolu en amont (le ChartCard n'i18n pas lui-meme). */
  title?: string
  /** Donnees du backend, projetees en serie(s). */
  series: ChartSeries<T>[]
  /** True pendant le chargement initial des donnees. */
  loading?: boolean
  /** Erreur de fetch a afficher (texte humain attendu, deja localise). */
  error?: Error | null
  /** Message a afficher si series est vide. */
  emptyMessage?: string
  /** Hauteur fixe en pixels (default 320). */
  height?: number
  /**
   * Builder de l'option ECharts. Appele a chaque rendu avec les series
   * courantes (ne pas y faire de side-effect).
   */
  buildOption: (series: ChartSeries<T>[]) => EChartsCoreOption
  /** ClassName optionnel pour la card racine. */
  className?: string
  /** Slot enfant (legende custom, footer note...). */
  children?: ReactNode
}

/**
 * ChartCard render switch : loading > error > empty > data.
 * Le rendu data fait du lazy-load echarts-for-react via Suspense.
 */
export function ChartCard<T = unknown>({
  title,
  series,
  loading,
  error,
  emptyMessage = 'Aucune donnée à afficher',
  height = 320,
  buildOption,
  className = '',
  children,
}: ChartCardProps<T>) {
  const isEmpty = !loading && !error && series.length === 0
  const option = useMemo(
    () => (isEmpty || loading || error ? null : buildOption(series)),
    [isEmpty, loading, error, buildOption, series],
  )

  return (
    <div
      className={`relative rounded-lg border border-border bg-card ${className}`}
      data-testid="chart-card"
    >
      {title && <div className="border-b border-border px-3 py-2 text-sm font-medium">{title}</div>}
      <div className="p-3" style={{ minHeight: height }}>
        {loading ? (
          <ChartCardLoading height={height} />
        ) : error ? (
          <ChartCardError error={error} height={height} />
        ) : isEmpty ? (
          <ChartCardEmpty message={emptyMessage} height={height} />
        ) : (
          <Suspense fallback={<ChartCardLoading height={height} />}>
            <ReactECharts
              option={option}
              style={{ height, width: '100%' }}
              notMerge
              lazyUpdate
              theme={undefined}
              data-testid="chart-card-echarts"
            />
          </Suspense>
        )}
      </div>
      {children}
    </div>
  )
}

function ChartCardLoading({ height }: { height: number }) {
  return (
    <div
      className="flex items-center justify-center"
      style={{ minHeight: height }}
      data-testid="chart-card-loading"
    >
      <Spinner size="md" />
    </div>
  )
}

function ChartCardError({ error, height }: { error: Error; height: number }) {
  return (
    <div
      className="flex items-center justify-center text-sm text-destructive"
      style={{ minHeight: height }}
      data-testid="chart-card-error"
      role="alert"
    >
      {error.message || 'Erreur de chargement'}
    </div>
  )
}

function ChartCardEmpty({ message, height }: { message: string; height: number }) {
  return (
    <div
      className="flex items-center justify-center text-sm text-muted-foreground"
      style={{ minHeight: height }}
      data-testid="chart-card-empty"
    >
      {message}
    </div>
  )
}
