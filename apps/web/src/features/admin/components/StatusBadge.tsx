/**
 * StatusBadge — badge d'état système (jobs, outcomes scheduler, états
 * génériques). Esthétique flat hard-edge : rounded-sm, caps 10px, dot carré
 * pulsé pour les états actifs. Couleurs exclusivement via tokens sémantiques.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { statusDisplay } from '../statusDisplay'
import { useAdminT } from '../useAdminText'

interface StatusBadgeProps {
  /** AdminStatus ('running', 'failed', 'ok'…) — valeur inconnue → neutre. */
  status: string
  /** Libellé custom (défaut : libellé i18n du statut). */
  label?: string
  title?: string
}

export function StatusBadge({ status, label, title }: StatusBadgeProps) {
  const tA = useAdminT()
  const d = statusDisplay(status)
  const color = d.token ? tokenCssVar(d.token) : undefined
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground"
      title={title}
    >
      {color && (
        <span
          aria-hidden
          className={`inline-block h-2 w-2 flex-none ${d.pulse ? 'animate-pulse' : ''}`}
          style={{ backgroundColor: color }}
        />
      )}
      <span className={color ? 'font-semibold' : undefined} style={color ? { color } : undefined}>
        {label ?? tA(d.labelKey)}
      </span>
    </span>
  )
}
