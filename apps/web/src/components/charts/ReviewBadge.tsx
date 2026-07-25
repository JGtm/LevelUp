/**
 * ReviewBadge — pastille de tournée de revue accolée au titre d'un graphe.
 *
 * INERTE PAR DÉFAUT : le badge n'est rendu que si la clé passée figure dans le
 * manifeste `lib/review/chart-review`. Manifeste vide (ou clé absente) → `null`,
 * donc aucun nœud DOM et aucun impact visuel. C'est ce qui permet de garder
 * l'outillage en place entre deux tournées.
 *
 * Aucune couleur en dur : `warning` / `info` / `destructive` sont des tokens
 * sémantiques résolus par `tokenCssVar` (skill color-tokens).
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { chartReview, type ChartReviewStatus } from '@/lib/review/chart-review'
import { REVIEW_TEXT } from '@/lib/review/i18n'
import { useAppShellStore } from '@/stores/appShellStore'

const STATUS_TOKEN: Record<ChartReviewStatus, SemanticToken> = {
  verify: 'warning',
  new: 'info',
  removal: 'destructive',
}

export interface ReviewBadgeProps {
  /** Identifiant stable du graphe (clé du manifeste de revue). */
  reviewKey?: string
}

export function ReviewBadge({ reviewKey }: ReviewBadgeProps) {
  const locale = useAppShellStore((s) => s.locale)
  const review = chartReview(reviewKey)
  if (!review) return null

  const text = REVIEW_TEXT[locale][review.status]
  const note = review.note[locale]
  const color = tokenCssVar(STATUS_TOKEN[review.status])

  return (
    <span
      data-testid="chart-review-badge"
      data-review-status={review.status}
      title={`${text.aria} - ${note}`}
      aria-label={`${text.aria} : ${note}`}
      className="ml-2 inline-flex shrink-0 items-center whitespace-nowrap rounded-full border px-2 py-0.5 align-middle text-2xs font-bold uppercase leading-none tracking-wider"
      style={{
        backgroundColor: `color-mix(in oklab, ${color} 18%, transparent)`,
        borderColor: `color-mix(in oklab, ${color} 55%, transparent)`,
        color,
      }}
    >
      {text.label}
    </span>
  )
}
