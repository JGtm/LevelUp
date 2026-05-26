/**
 * TipsTicker — bandeau horizontal défilant de tips contextuels.
 *
 * Affiche une suite de petites pills cliquables qui défilent automatiquement
 * de droite à gauche (style ticker). Pause au hover/focus. Respecte
 * `prefers-reduced-motion` (rendu statique en grille scrollable).
 *
 * Usage typique : indices visuels sur une page complexe, pointant vers le
 * glossaire (`/help?tab=glossary#glossary-entry-<slug>`).
 */
import { useMemo, type ReactNode } from 'react'

export interface Tip {
  id: string
  term: string
  shortDef: string
  href?: string
}

interface TipsTickerProps {
  tips: Tip[]
  /** Durée d'un cycle complet en secondes. Défaut : 60s. */
  durationSeconds?: number
  /** Pause l'animation au hover ou focus. Défaut : true. */
  pauseOnHover?: boolean
  /** Label aria pour la région du ticker. */
  ariaLabel?: string
  /** Pictogramme optionnel affiché en tête de chaque pill. */
  leadingIcon?: ReactNode
}

const KEYFRAMES = `
@keyframes tips-ticker-scroll {
  0% { transform: translateX(0); }
  100% { transform: translateX(-50%); }
}
`

export function TipsTicker({
  tips,
  durationSeconds = 60,
  pauseOnHover = true,
  ariaLabel,
  leadingIcon,
}: TipsTickerProps) {
  const doubled = useMemo(() => [...tips, ...tips], [tips])

  if (tips.length === 0) return null

  const trackClasses = [
    'flex w-max items-stretch gap-2 py-1.5 px-2',
    'motion-safe:[animation:tips-ticker-scroll_var(--ticker-duration)_linear_infinite]',
    pauseOnHover ? 'motion-safe:group-hover:[animation-play-state:paused]' : '',
    pauseOnHover ? 'motion-safe:group-focus-within:[animation-play-state:paused]' : '',
    'motion-reduce:flex-wrap motion-reduce:w-full motion-reduce:overflow-x-auto',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <section
      role="region"
      aria-label={ariaLabel}
      className="group relative w-full overflow-hidden rounded-md border border-border bg-card/50"
    >
      <style>{KEYFRAMES}</style>
      <div
        className={trackClasses}
        style={{ ['--ticker-duration' as never]: `${durationSeconds}s` }}
      >
        {doubled.map((tip, idx) => (
          <TipPill
            key={`${tip.id}-${idx}`}
            tip={tip}
            leadingIcon={leadingIcon}
            ariaHidden={idx >= tips.length}
          />
        ))}
      </div>
    </section>
  )
}

interface TipPillProps {
  tip: Tip
  leadingIcon?: ReactNode
  /** True pour les copies dupliquées (animation seamless) — masqué des lecteurs d'écran. */
  ariaHidden: boolean
}

function TipPill({ tip, leadingIcon, ariaHidden }: TipPillProps) {
  const content = (
    <>
      {leadingIcon && (
        <span className="shrink-0 text-muted-foreground" aria-hidden="true">
          {leadingIcon}
        </span>
      )}
      <span className="shrink-0 text-xs font-semibold">{tip.term}</span>
      <span className="hidden text-xs text-muted-foreground sm:inline">·</span>
      <span className="hidden max-w-[28rem] truncate text-xs text-muted-foreground sm:inline">
        {tip.shortDef}
      </span>
    </>
  )

  const pillClasses =
    'inline-flex items-center gap-1.5 whitespace-nowrap rounded-full border border-border bg-background px-3 py-1 transition-colors hover:border-foreground/40 hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40'

  if (tip.href) {
    return (
      <a
        href={tip.href}
        className={pillClasses}
        aria-hidden={ariaHidden || undefined}
        tabIndex={ariaHidden ? -1 : 0}
      >
        {content}
      </a>
    )
  }
  return (
    <span
      className={pillClasses}
      aria-hidden={ariaHidden || undefined}
    >
      {content}
    </span>
  )
}
