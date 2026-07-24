/**
 * SVG inline mapping catégorie → icône.
 * Pas de dépendance icon lib — cohérent avec les autres icônes du shell (NavL1).
 * Les couleurs sont injectées via `currentColor` pour s'adapter au thème.
 */
import type { NotificationCategory } from './types'

export interface CategoryIconProps {
  category: NotificationCategory
  className?: string
}

export function CategoryIcon({ category, className = 'h-4 w-4' }: CategoryIconProps) {
  const Icon = ICONS[category] ?? IconBell
  return <Icon className={className} />
}

const ICONS: Record<NotificationCategory, React.ComponentType<{ className?: string }>> = {
  app_release: IconSparkles,
  match_synced: IconCheck,
  media_added: IconImage,
  media_liked: IconHeart,
  objective_assigned: IconTarget,
  objective_completed: IconCheck,
  challenge_added: IconFlag,
  challenge_completed: IconTrophy,
  season_pass_level: IconStar,
  sync_error: IconAlert,
  personal_record: IconTrophy,
  threshold_crossed: IconTrending,
  friend_added: IconUser,           // §6 Squad/Sessions overhaul
  friend_sync_completed: IconCheck, // §6 — récap silencieux
  // 2026-05-16 — catégories Halo (V7 sharedprovider + V2 progression alignées).
  // Convention : Star = niveau/rank ; Trophy = completion/achievement.
  data_health_warning: IconAlert,
  career_rank: IconStar,
  skill_tier: IconTrending,
  battlepass_completed: IconTrophy,
  citation_tier: IconStar,
  citation_mastery: IconTrophy,
  // 2026-05-18 — Progression V2 (Ascension), coach proactif.
  record_near_miss: IconTrending,
  milestone_unlocked: IconTrophy,
  milestone_near_miss: IconTarget,
  lusr_tier_approach: IconTrending,
  streak_milestone: IconFlame,
  comeback_welcome: IconSparkles,
  trend_consolidate: IconTarget, // axe à consolider — neutre (pas IconTrending montant)
  title_ready: IconSparkles, // MT-19 / axe E — titre prêt (célébration, comme app_release)
  rival_encounter: IconUser, // relations-E — rival croisé (un joueur recroisé en duel)
  medal_first_earned: IconTrophy, // V72-20 — médaille inédite (récompense décrochée)
}

function svg(props: { className?: string; children: React.ReactNode }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      className={props.className}
      viewBox="0 0 20 20"
      fill="currentColor"
      aria-hidden="true"
    >
      {props.children}
    </svg>
  )
}

function IconBell({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path d="M10 2a6 6 0 00-6 6v3.586l-.707.707A1 1 0 004 14h12a1 1 0 00.707-1.707L16 11.586V8a6 6 0 00-6-6zm0 16a3 3 0 003-3H7a3 3 0 003 3z" />
    ),
  })
}
function IconSparkles({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path d="M10 2l1.5 4.5L16 8l-4.5 1.5L10 14l-1.5-4.5L4 8l4.5-1.5L10 2zM4 14l.75 2.25L7 17l-2.25.75L4 20l-.75-2.25L1 17l2.25-.75L4 14z" />
    ),
  })
}
function IconCheck({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path
        fillRule="evenodd"
        d="M16.704 4.153a.75.75 0 01.143 1.052l-8 10.5a.75.75 0 01-1.127.075l-4.5-4.5a.75.75 0 011.06-1.06l3.894 3.893 7.48-9.817a.75.75 0 011.05-.143z"
        clipRule="evenodd"
      />
    ),
  })
}
function IconImage({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path
        fillRule="evenodd"
        d="M4 3a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V5a2 2 0 00-2-2H4zm12 12H4l2.5-3.5 2 2.5L12 9l4 6z"
        clipRule="evenodd"
      />
    ),
  })
}
function IconHeart({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path d="M10 18s-7-4.35-7-9.5A4.5 4.5 0 0110 4.5 4.5 4.5 0 0117 8.5C17 13.65 10 18 10 18z" />
    ),
  })
}
function IconTarget({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path
        fillRule="evenodd"
        d="M10 2a8 8 0 100 16 8 8 0 000-16zm0 3a5 5 0 100 10 5 5 0 000-10zm0 3a2 2 0 100 4 2 2 0 000-4z"
        clipRule="evenodd"
      />
    ),
  })
}
function IconFlag({ className }: { className?: string }) {
  return svg({
    className,
    children: <path d="M3 3a1 1 0 011-1h12l-3 4 3 4H5v7a1 1 0 11-2 0V3z" />,
  })
}
function IconTrophy({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path d="M5 3h10v3a5 5 0 01-5 5 5 5 0 01-5-5V3zm1 11h8v3H6v-3zm-1 4h10v1H5v-1z" />
    ),
  })
}
function IconStar({ className }: { className?: string }) {
  return svg({
    className,
    children: <path d="M10 1.5l2.6 5.3 5.9.9-4.25 4.15 1 5.85L10 14.9l-5.25 2.8 1-5.85L1.5 7.7l5.9-.9L10 1.5z" />,
  })
}
function IconAlert({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path
        fillRule="evenodd"
        d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.169 2.625-1.515 2.625H3.72c-1.346 0-2.188-1.458-1.515-2.625L8.485 2.495zM10 6a1 1 0 011 1v3a1 1 0 11-2 0V7a1 1 0 011-1zm0 7a1 1 0 100 2 1 1 0 000-2z"
        clipRule="evenodd"
      />
    ),
  })
}
function IconTrending({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path d="M2.293 13.707a1 1 0 010-1.414L7 7.586l3 3 5.293-5.293a1 1 0 111.414 1.414L10.414 12.4l-3-3-4.707 4.707a1 1 0 01-1.414 0z" />
    ),
  })
}
function IconUser({ className }: { className?: string }) {
  return svg({
    className,
    children: (
      <path
        fillRule="evenodd"
        d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z"
        clipRule="evenodd"
      />
    ),
  })
}
function IconFlame({ className }: { className?: string }) {
  // Cohérent avec features/ascension/StreakBadge.tsx (même path).
  return svg({
    className,
    children: (
      <path d="M10 1.5c-.4 1.5-1.5 2.6-3 3.5-1.5.9-2.5 2.4-2.5 4 0 3 2.5 5.5 5.5 5.5s5.5-2.5 5.5-5.5c0-1.6-.8-3-2-4-1.4 1.3-3.5 0-3.5-3.5z" />
    ),
  })
}
