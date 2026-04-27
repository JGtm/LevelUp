/**
 * PeriodFilter — segmented control de période transverse.
 *
 * Conforme à PLAN_META_FOUNDATIONS_GO § 6.2.4 — composant UI partagé Phase 2.
 *
 * Source de vérité : `temporal.Period` côté Go (cf.
 * `internal/analysis/temporal/period.go`). 5 valeurs canoniques :
 *
 *   - "all"  : toutes les périodes (default)
 *   - "2y"   : 2 dernières années
 *   - "1y"   : dernière année
 *   - "1m"   : dernier mois
 *   - "1w"   : dernière semaine
 *
 * Réutilisable par Career, Synthesis, Citations, Timeseries, Explorer.
 *
 * i18n-naïf : labels passés en prop (résolus via le manifest commonManifest
 * par le caller). Tokens couleur via tokenCssVar (aucune couleur en dur).
 */

/** Mirror frontend de temporal.Period Go. */
export type Period = 'all' | '2y' | '1y' | '1m' | '1w'

/** Liste ordonnée des périodes affichées dans le segmented control. */
// eslint-disable-next-line react-refresh/only-export-components
export const ALL_PERIODS: readonly Period[] = ['all', '2y', '1y', '1m', '1w']

export interface PeriodFilterProps {
  /** Période sélectionnée. */
  value: Period
  /** Callback de changement. */
  onChange: (next: Period) => void
  /**
   * Map period -> label déjà localisé. Le caller résout via formatMessage(
   * commonManifest, 'common.period.{key}', locale) avant le passage au composant.
   */
  labels: Record<Period, string>
  /** Aria-label du groupe (déjà localisé). Default "Filtre période". */
  ariaLabel?: string
  /** ClassName additionnel. */
  className?: string
  /**
   * Si true, désactive l'interactivité (utilisé pendant un fetch en cours
   * pour éviter les triggers multiples).
   */
  disabled?: boolean
}

export function PeriodFilter({
  value,
  onChange,
  labels,
  ariaLabel = 'Filtre période',
  className = '',
  disabled = false,
}: PeriodFilterProps) {
  return (
    <div
      role="radiogroup"
      aria-label={ariaLabel}
      className={`inline-flex items-center gap-1 rounded-md border border-border bg-card p-1 ${className}`}
      data-testid="period-filter"
    >
      {ALL_PERIODS.map((period) => {
        const isActive = period === value
        return (
          <button
            key={period}
            type="button"
            role="radio"
            aria-checked={isActive}
            disabled={disabled}
            onClick={() => onChange(period)}
            data-testid={`period-filter-option-${period}`}
            data-active={isActive ? 'true' : 'false'}
            className={[
              'rounded px-3 py-1 text-xs font-medium transition-colors',
              isActive
                ? 'bg-primary text-primary-foreground'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
              disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer',
            ].join(' ')}
          >
            {labels[period]}
          </button>
        )
      })}
    </div>
  )
}
