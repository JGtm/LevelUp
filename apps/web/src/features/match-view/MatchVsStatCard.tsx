/**
 * MatchVsStatCard — carte stat générique "X vs Y + delta" (MMR, frags, morts, vie).
 *
 * Extrait de MatchStatCards.tsx (C6) — règle des 500 lignes par fichier.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { KpiCard } from '@/components/cards/KpiCard'

interface MatchVsStatCardProps {
  label: string
  /** Valeur principale (gauche ou seule) */
  primary: number | string | null
  /** Tooltip natif (title) au survol de la valeur principale — lève l'ambiguïté d'un MM:SS. */
  primaryTitle?: string
  /** Valeur secondaire (droite, optionnelle) */
  secondary?: number | string | null
  /** Libellé sous la valeur primaire */
  primaryLabel?: string
  /** Libellé sous la valeur secondaire */
  secondaryLabel?: string
  /** Delta affiché en dessous */
  delta?: number | null
  /** Si true, un delta négatif est favorable (morts, durée de vie basse) */
  lowerIsBetter?: boolean
  /** Formater la valeur (ex. décimales) */
  precision?: number
  /**
   * Accent FIXE (type 2 du catalogue) appliqué quand il n'y a pas de delta —
   * ex. métrique sans comparaison (vie moyenne). Ignoré dès qu'un delta existe
   * (l'accent dynamique type 4 prend le dessus).
   */
  fixedAccent?: SemanticToken
}

export function MatchVsStatCard({
  label,
  primary,
  primaryTitle,
  secondary,
  primaryLabel,
  secondaryLabel,
  delta,
  lowerIsBetter = false,
  precision = 0,
  fixedAccent,
}: MatchVsStatCardProps) {
  const fmt = (v: number | string | null | undefined) => {
    if (v == null) return '—'
    if (typeof v === 'string') return v
    return precision > 0 ? v.toFixed(precision) : Math.round(v).toString()
  }

  const isFavorable =
    delta == null ? null : lowerIsBetter ? delta < 0 : delta > 0

  // Accent dynamique (type 4 du catalogue) : barre 3px verte si favorable,
  // rouge si défavorable. Sans delta, on retombe sur l'accent fixe éventuel
  // (type 2 — ex. barre neutre pour une métrique sans comparaison).
  const accent: SemanticToken | undefined =
    isFavorable === null ? fixedAccent : isFavorable ? 'divergent-pos' : 'divergent-neg'

  const deltaStyle =
    isFavorable === null
      ? undefined
      : { color: tokenCssVar(isFavorable ? 'divergent-pos' : 'divergent-neg') }

  return (
    <KpiCard accent={accent} className="h-full">
      <div className="px-3 py-2.5">
        <p className="text-2xs text-muted-foreground uppercase tracking-wide mb-1.5">{label}</p>
        <div className="flex items-baseline gap-1.5">
          <div>
            <span className="text-lg font-bold text-foreground leading-none" title={primaryTitle}>{fmt(primary)}</span>
            {primaryLabel && (
              <p className="text-2xs text-muted-foreground mt-0.5">{primaryLabel}</p>
            )}
          </div>
          {secondary != null && (
            <>
              <span className="text-muted-foreground text-xs font-light">vs</span>
              <div>
                <span className="text-lg font-bold text-foreground leading-none">{fmt(secondary)}</span>
                {secondaryLabel && (
                  <p className="text-2xs text-muted-foreground mt-0.5">{secondaryLabel}</p>
                )}
              </div>
            </>
          )}
          {delta != null && (
            <span className="ml-auto text-xs font-semibold" style={deltaStyle}>
              {delta > 0 ? '+' : ''}{fmt(delta)}
            </span>
          )}
        </div>
      </div>
    </KpiCard>
  )
}
