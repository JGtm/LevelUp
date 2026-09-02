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
import { useColorPaletteVersion } from '@/lib/accessibility/useColorPaletteVersion'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import { chartReview } from '@/lib/review/chart-review'

import { ReviewBadge } from './ReviewBadge'

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
  /** Titre i18n-resolu en amont (le ChartCard n'i18n pas lui-meme). Accepte un ReactNode pour injecter un InfoTooltip à côté du titre. */
  title?: ReactNode
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
   * Quand true, la carte s'étire pour remplir la cellule CSS Grid parente
   * (via `h-full flex-col`). `height` devient le minimum garanti.
   * Aucun ResizeObserver — pure CSS : `flex-1` sur le contenu + `height:100%`
   * sur ECharts. Nécessite que le parent soit un conteneur à hauteur définie
   * (CSS Grid avec align-items:stretch ou flex avec hauteur explicite).
   */
  fluid?: boolean
  /**
   * Builder de l'option ECharts. Appele a chaque rendu avec les series
   * courantes (ne pas y faire de side-effect).
   */
  buildOption: (series: ChartSeries<T>[]) => EChartsCoreOption
  /** ClassName optionnel pour la card racine. */
  className?: string
  /** Slot enfant (legende custom, footer note...). */
  children?: ReactNode
  /**
   * Légende HTML rendue en PIED DE CARD systématique (hors canvas, `flex-none`,
   * collée au bord bas via un séparateur discret). Le chrome du footer (bordure +
   * padding) est appliqué ici : l'appelant ne passe que le contenu (typiquement un
   * `<ChartLegend>`). Distinct de `children` (slot brut sans chrome). Les couleurs
   * doivent provenir des MÊMES résolutions de tokens que les séries.
   */
  legend?: ReactNode
  /** Handlers d'événements ECharts (ex. legendselectchanged). Forwarded à ReactECharts onEvents. */
  onEvents?: Record<string, (params: unknown) => void>
  /**
   * Identifiant stable du graphe pour la tournée de revue visuelle
   * (`lib/review/chart-review`). Rend un `<ReviewBadge>` à droite du titre
   * UNIQUEMENT si la clé figure dans le manifeste ; sinon aucun nœud n'est
   * ajouté. Prop absente = rendu strictement identique à l'existant.
   */
  reviewKey?: string
  /**
   * Moteur de rendu ECharts. `canvas` par défaut — c'est le mode historique de tous les
   * graphes de l'app, et le seul qui tienne les séries denses (un point = un pixel).
   *
   * `svg` rend le graphe INDÉPENDANT DE LA RÉSOLUTION : le texte est du vrai texte, peint
   * par le moteur de polices du navigateur, net au zoom comme après un changement d'écran.
   * Le canvas, lui, est un bitmap figé au `devicePixelRatio` du MONTAGE — sur un écran à
   * mise à l'échelle fractionnaire (125 %, 150 %) ses libellés paraissent flous à côté du
   * texte DOM qui les entoure. Retour utilisateur du 2026-09-02 sur « Premier frag /
   * première mort », dont l'essentiel de la lecture EST du texte dans le canvas.
   *
   * À N'ACTIVER QUE SUR LES GRAPHES PAUVRES EN POINTS : le SVG crée un nœud DOM par point.
   * Les deux graphes denses du dossier (Heatmap2DChart, ScatterChart) restent en canvas.
   */
  renderer?: 'canvas' | 'svg'
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
  fluid = false,
  buildOption,
  className = '',
  children,
  legend,
  onEvents,
  reviewKey,
  renderer = 'canvas',
}: ChartCardProps<T>) {
  const isEmpty = !loading && !error && series.length === 0
  // Le themeVersion s'incrémente lors d'un toggle data-theme : on l'inclut
  // dans les deps du useMemo pour forcer le rebuild de l'option et donc le
  // re-render canvas avec les couleurs du nouveau thème.
  const themeVersion = useThemeVersion()
  // La palette d'accessibilité (Okabe-Ito, Cividis, Tol-Bright…) est appliquée via
  // :root style SANS toggler data-theme : on observe aussi son changement pour
  // rebuild l'option ECharts, sinon les couleurs hex « bakées » par buildOption
  // (resolveToken) resteraient périmées jusqu'au prochain remount (revue P1 2026-06-02).
  const paletteVersion = useColorPaletteVersion()
  const option = useMemo(
    () => (isEmpty || loading || error ? null : buildOption(series)),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [isEmpty, loading, error, buildOption, series, themeVersion, paletteVersion],
  )

  // En mode fluid : la carte est h-full (CSS Grid stretch la pousse à la
  // hauteur de la ligne). Le contenu est flex-1 avec minHeight = height + 24
  // (24px = padding p-3 top+bottom, border-box). ECharts reçoit height:100%
  // qui se résout en pixels via la hauteur flex définie. Pas de ResizeObserver
  // → pas de boucle de rétroaction.
  const outerCls = fluid
    ? `relative flex h-full flex-col rounded-lg border border-border bg-card ${className}`
    : `relative rounded-lg border border-border bg-card ${className}`

  const contentStyle = fluid
    ? { minHeight: height + 24 } // +24 = p-3 padding (border-box)
    : { minHeight: height }

  const contentCls = fluid ? 'flex-1 p-3' : 'p-3'

  return (
    <div className={outerCls} data-testid="chart-card">
      {title && (
        <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
          {/* Hors tournée de revue (clé absente du manifeste), le titre est rendu
              TEL QUEL — aucun conteneur ajouté, donc zéro risque de décalage. */}
          {chartReview(reviewKey) ? (
            <span className="flex flex-wrap items-center">
              {title}
              <ReviewBadge reviewKey={reviewKey} />
            </span>
          ) : (
            title
          )}
        </div>
      )}
      <div className={contentCls} style={contentStyle}>
        {loading ? (
          <ChartCardLoading height={height} fluid={fluid} />
        ) : error ? (
          <ChartCardError error={error} height={height} fluid={fluid} />
        ) : isEmpty ? (
          <ChartCardEmpty message={emptyMessage} height={height} fluid={fluid} />
        ) : (
          <Suspense fallback={<ChartCardLoading height={height} fluid={fluid} />}>
            <ReactECharts
              option={option}
              style={fluid ? { height: '100%', width: '100%' } : { height, width: '100%' }}
              notMerge
              lazyUpdate
              theme={undefined}
              opts={{ devicePixelRatio: window.devicePixelRatio, renderer }}
              onEvents={onEvents}
              data-testid="chart-card-echarts"
            />
          </Suspense>
        )}
      </div>
      {children}
      {legend != null && (
        <div
          className="flex-none border-t border-border px-3 py-2"
          data-testid="chart-card-legend"
        >
          {legend}
        </div>
      )}
    </div>
  )
}

function ChartCardLoading({ height, fluid }: { height: number; fluid?: boolean }) {
  return (
    <div
      className="flex items-center justify-center"
      style={fluid ? { height: '100%', minHeight: height } : { minHeight: height }}
      data-testid="chart-card-loading"
    >
      <Spinner size="md" />
    </div>
  )
}

function ChartCardError({ error, height, fluid }: { error: Error; height: number; fluid?: boolean }) {
  return (
    <div
      className="flex items-center justify-center text-sm text-destructive"
      style={fluid ? { height: '100%', minHeight: height } : { minHeight: height }}
      data-testid="chart-card-error"
      role="alert"
    >
      {error.message || 'Erreur de chargement'}
    </div>
  )
}

function ChartCardEmpty({ message, height, fluid }: { message: string; height: number; fluid?: boolean }) {
  return (
    <div
      className="flex items-center justify-center text-sm text-muted-foreground"
      style={fluid ? { height: '100%', minHeight: height } : { minHeight: height }}
      data-testid="chart-card-empty"
    >
      {message}
    </div>
  )
}
