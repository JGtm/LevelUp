/**
 * ArcSummary — résumé d'un arc avec progression visuelle.
 *
 * Référence : Axe 1 du plan PLAN_challenges_xp_system.md.
 * Affiche le titre narratif, la description courte, et la progression
 * (étape courante / total) avec une mini-barre.
 */
import type { Arc } from '@/lib/prestige'

interface ArcSummaryProps {
  arc: Arc
  /** Étapes complétées sur le total (vient des challenges liés à l'arc). */
  completedSteps?: number
  /** Total des étapes de l'arc. */
  totalSteps?: number
  onClick?: () => void
}

export function ArcSummary({
  arc,
  completedSteps = 0,
  totalSteps = 0,
  onClick,
}: ArcSummaryProps) {
  const progress = totalSteps > 0 ? Math.round((completedSteps / totalSteps) * 100) : 0
  const isComplete = arc.completed_at != null

  const Wrapper = onClick ? 'button' : 'div'

  return (
    <Wrapper
      onClick={onClick}
      className={[
        'block w-full rounded-lg border border-border bg-card p-4 text-left',
        onClick ? 'transition-colors hover:bg-accent/40' : '',
      ].join(' ')}
    >
      <header className="flex items-start justify-between gap-3">
        <div className="flex-1">
          <h3 className="font-semibold">{arc.title}</h3>
          {arc.description && (
            <p className="mt-1 text-xs text-muted-foreground">{arc.description}</p>
          )}
        </div>
        {isComplete && (
          <span className="shrink-0 rounded-full border border-border px-2 py-0.5 text-[10px] uppercase text-muted-foreground">
            Accompli
          </span>
        )}
        {arc.is_preset && !isComplete && (
          <span className="shrink-0 rounded-full border border-border px-2 py-0.5 text-[10px] uppercase text-muted-foreground">
            Preset
          </span>
        )}
      </header>

      {totalSteps > 0 && (
        <div className="mt-3 space-y-1">
          <div className="flex items-baseline justify-between text-xs text-muted-foreground">
            <span>
              Étape {completedSteps} / {totalSteps}
            </span>
            <span>{progress}%</span>
          </div>
          <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
            <div
              className="h-full bg-primary transition-all"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      )}
    </Wrapper>
  )
}
