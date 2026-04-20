/**
 * PrivacyBanner — bannière d'avertissement sur la privacy des matchs.
 * Sprint 54-B.
 *
 * Affiche un message contextuel lorsque le compte Halo du joueur
 * est privé (total) ou partiellement privé.
 *
 * Usage :
 *   <PrivacyBanner warning={matchHistoryData?.privacy_warning} />
 */
import type { MatchPrivacyWarning } from '@/lib/api/types'

interface PrivacyBannerProps {
  warning: MatchPrivacyWarning | undefined | null
  /** Classe CSS additionnelle. */
  className?: string
}

const LEVEL_STYLES: Record<string, { container: string; icon: string }> = {
  partial: {
    container: 'bg-warning/10 border border-warning text-warning-foreground',
    icon: '⚠️',
  },
  full: {
    container: 'bg-destructive/10 border border-destructive text-destructive',
    icon: '🔒',
  },
}

/**
 * PrivacyBanner — n'affiche rien si warning est null/undefined ou level === 'none'.
 */
export function PrivacyBanner({ warning, className = '' }: PrivacyBannerProps) {
  if (!warning || warning.level === 'none') return null

  const styles = LEVEL_STYLES[warning.level] ?? LEVEL_STYLES.partial

  return (
    <div
      role="alert"
      className={`flex items-start gap-2 rounded-md px-4 py-3 text-sm ${styles.container} ${className}`}
    >
      <span aria-hidden="true" className="mt-0.5 flex-shrink-0">
        {styles.icon}
      </span>
      <p>{warning.message}</p>
    </div>
  )
}
