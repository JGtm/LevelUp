/**
 * ArcSummary — résumé d'un arc avec progression visuelle.
 *
 * Référence : Axe 1 du plan PLAN_challenges_xp_system.md.
 * Affiche le titre narratif, la description courte, et la progression
 * (étape courante / total) avec une mini-barre.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import type { Arc } from '@/lib/prestige'
import { getPrestigeText } from '../i18n'

interface ArcSummaryProps {
  arc: Arc
  /** Étapes complétées sur le total (vient des challenges liés à l'arc). */
  completedSteps?: number
  /** Total des étapes de l'arc. */
  totalSteps?: number
  /** PP cumulés des objectifs (actifs) de l'arc — récompense disponible. */
  totalPP?: number
  onClick?: () => void
}

export function ArcSummary({
  arc,
  completedSteps = 0,
  totalSteps = 0,
  totalPP = 0,
  onClick,
}: ArcSummaryProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getPrestigeText(locale)
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
        <h3 className="min-w-0 flex-1 font-semibold">{arc.title}</h3>
        <div className="flex shrink-0 items-center gap-1.5">
          {totalPP > 0 && (
            <span
              className="rounded-full bg-primary/10 px-2 py-0.5 text-2xs font-semibold tabular-nums text-primary"
              title={t.arcTotalPPTooltip}
            >
              {totalPP} PP
            </span>
          )}
          {isComplete && (
            <span className="rounded-full border border-border px-2 py-0.5 text-2xs uppercase text-muted-foreground">
              {t.arcCompleted}
            </span>
          )}
          {arc.is_preset && !isComplete && (
            <span className="rounded-full border border-border px-2 py-0.5 text-2xs uppercase text-muted-foreground">
              {t.arcPreset}
            </span>
          )}
        </div>
      </header>

      {totalSteps > 0 ? (
        /* Flanking (même structure que le StreakCard) : grand seuil gauche | colonne
           centrale [description JUSTE au-dessus de la barre, en position du « nom de
           série »] | grand seuil droit. Cf. demande user. */
        <div className="mt-2 flex items-center gap-2.5">
          <span className="shrink-0 text-2xl font-bold leading-none tabular-nums text-foreground">{completedSteps}</span>
          <div className="min-w-0 flex-1 space-y-1">
            {arc.description && (
              <p className="truncate text-xs text-muted-foreground">{arc.description}</p>
            )}
            <div className="h-1.5 overflow-hidden rounded-full bg-muted">
              <div className="h-full bg-primary transition-all" style={{ width: `${progress}%` }} />
            </div>
          </div>
          <span className="shrink-0 text-2xl font-bold leading-none tabular-nums text-foreground">{totalSteps}</span>
        </div>
      ) : (
        arc.description && (
          <p className="mt-1 text-xs text-muted-foreground">{arc.description}</p>
        )
      )}
    </Wrapper>
  )
}
