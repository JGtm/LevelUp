/**
 * SectionHeader — en-tête de section canonique du dashboard admin (A8.3).
 * Unifie les 3 patterns historiques (h3 caps muted nu, h3 + description,
 * h2 semibold) : titre caps + description optionnelle + slot d'actions à
 * droite (boutons de section).
 */
import type { ReactNode } from 'react'

export function SectionHeader({
  title,
  description,
  actions,
}: {
  title: ReactNode
  description?: ReactNode
  /** Slot aligné à droite (boutons/contrôles de section). */
  actions?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-2">
      <div>
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">{title}</h3>
        {description && <p className="mt-0.5 max-w-2xl text-xs text-muted-foreground">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  )
}
