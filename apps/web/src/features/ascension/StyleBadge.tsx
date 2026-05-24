/**
 * StyleBadge — Section A2 : badge style de jeu (FK/FD ratio → StyleKey).
 *
 * 4 styles : opportunistic_finisher / overextended / hyper_engaged / passive.
 * Chaque style a une icône emoji + label i18n + description courte.
 */
import type { StyleSignature } from './types'
import type { AscensionText } from './i18n'

interface StyleBadgeProps {
  style: StyleSignature
  engagement: { score: number; tier: string; matches_per_day_avg: number }
  t: AscensionText
}

const STYLE_ICON: Record<string, string> = {
  opportunistic_finisher: '🎯',
  overextended: '⚡',
  hyper_engaged: '🔥',
  passive: '🛡️',
}

export function StyleBadge({ style, engagement, t }: StyleBadgeProps) {
  const key = style.style_key ?? ''
  const icon = STYLE_ICON[key] ?? '🎮'
  const label = t.styleKey?.[key] ?? key

  return (
    <div className="flex flex-wrap items-start gap-4">
      <div className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-2">
        <span className="text-xl" aria-hidden="true">{icon}</span>
        <div>
          <p className="text-sm font-semibold">{label}</p>
          <p className="text-xs text-muted-foreground">
            FK {style.first_kill_count} / FD {style.first_death_count}
            {' '}(ratio {style.fkfd_ratio.toFixed(2)})
          </p>
        </div>
      </div>

      <div className="flex items-center gap-2 rounded-md border border-border bg-card px-3 py-2">
        <span className="text-xl" aria-hidden="true">📅</span>
        <div>
          <p className="text-sm font-semibold">{t.engagementTier?.[engagement.tier] ?? engagement.tier}</p>
          <p className="text-xs text-muted-foreground">
            {engagement.matches_per_day_avg.toFixed(1)} {t.profileMatchesPerDay}
          </p>
        </div>
      </div>
    </div>
  )
}
