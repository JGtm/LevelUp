/**
 * Badge cliquable dans la NavL1 affichant la longueur de la streak `daily_play`
 * active du joueur. Lien direct vers la page Ascension.
 *
 * Pattern : structure alignée sur NotificationsBell (icône + badge count).
 * Le multiplicateur PP est inscrit en tooltip via title= (a11y).
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.3.
 */
import { Link } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { useStreaks } from './queries'
import { getAscensionText } from './i18n'
import { formatMultiplier, interpolate } from './format'

export interface StreakBadgeProps {
  playerSlug: string
}

export function StreakBadge({ playerSlug }: StreakBadgeProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const { data } = useStreaks(playerSlug, !!playerSlug)

  // Sélectionne la streak `daily_play` active (la plus visible côté UX).
  // Les autres types sont visibles dans la page dédiée.
  const dailyPlay = (data?.items ?? []).find(
    (s) => s.type === 'daily_play' && s.status !== 'broken',
  )

  const length = dailyPlay?.current_length ?? 0
  const ariaLabel =
    length > 0
      ? interpolate(t.streakBadgeAriaLabel, { count: length })
      : t.streakBadgeAriaEmpty

  // Tooltip = multiplicateur PP courant (si streak active).
  const tooltip = dailyPlay
    ? interpolate(t.streakPPMultiplier, {
        value: formatMultiplier(dailyPlay.pp_multiplier).replace('×', ''),
      })
    : undefined

  return (
    <Link
      to="/players/$playerSlug/ascension"
      params={{ playerSlug }}
      aria-label={ariaLabel}
      title={tooltip}
      className="relative flex items-center rounded-md px-2 py-1.5 text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
    >
      <FlameIcon />
      {length > 0 && (
        <span
          className="absolute -right-0.5 -top-0.5 inline-flex h-4 min-w-[1rem] items-center justify-center rounded-full bg-primary px-1 text-[10px] font-medium leading-none text-primary-foreground"
          aria-hidden="true"
        >
          {length > 99 ? '99+' : length}
        </span>
      )}
    </Link>
  )
}

function FlameIcon({ className = 'h-5 w-5' }: { className?: string }) {
  // Icône feu inline (lucide-style), évite une dépendance lib externe —
  // cohérent avec features/notifications/icons.tsx.
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="20"
      height="20"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d="M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z" />
    </svg>
  )
}
