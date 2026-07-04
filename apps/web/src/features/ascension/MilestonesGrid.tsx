/**
 * MilestonesGrid — grille visuelle des milestones (earned vs locked).
 *
 * Affiche tous les milestones du catalogue du titre, avec :
 *  - une tone "earned" (amber-gold) pour ceux débloqués
 *  - une tone "locked" (muted) pour ceux à atteindre
 *  - le seuil, la date d'achievement si earned, le `condition` text si fourni
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §5.3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useMilestones } from './queries'
import { getAscensionText } from './i18n'
import { formatAscensionDate, interpolate } from './format'
import type { MilestoneItem } from './types'

export interface MilestonesGridProps {
  playerSlug: string
}

export function MilestonesGrid({ playerSlug }: MilestonesGridProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const { data, isLoading, isError } = useMilestones(playerSlug)

  if (isLoading) {
    return (
      <p className="text-sm text-muted-foreground" role="status">
        {t.loading}
      </p>
    )
  }
  if (isError) {
    return (
      <p className="text-sm text-destructive" role="alert">
        {t.errorLoading}
      </p>
    )
  }

  const items = data?.items ?? []
  if (items.length === 0) {
    return (
      <div className="rounded-md border border-border bg-card p-6 text-center text-muted-foreground">
        {t.milestonesEmpty}
      </div>
    )
  }

  // Tri : earned d'abord (par earned_at DESC), puis locked (par metric+threshold ASC).
  const sorted = [...items].sort((a, b) => {
    if (a.earned !== b.earned) return a.earned ? -1 : 1
    if (a.earned && b.earned) {
      const ta = a.earned_at ? new Date(a.earned_at).getTime() : 0
      const tb = b.earned_at ? new Date(b.earned_at).getTime() : 0
      return tb - ta
    }
    if (a.metric !== b.metric) return a.metric.localeCompare(b.metric)
    return a.threshold - b.threshold
  })

  const earnedCount = items.filter((m) => m.earned).length

  return (
    <section aria-labelledby="milestones-section-heading">
      <header className="mb-3 flex items-baseline justify-between gap-2">
        <h2 id="milestones-section-heading" className="text-lg font-semibold">
          {t.milestonesSectionTitle}
        </h2>
        <span className="text-xs text-muted-foreground">
          {interpolate(t.milestonesEarnedCount, {
            n: earnedCount,
            total: items.length,
            plural: earnedCount > 1 ? 's' : '',
          })}
        </span>
      </header>
      <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
        {sorted.map((m) => (
          <li key={m.id}>
            <MilestoneCard milestone={m} locale={locale} t={t} />
          </li>
        ))}
      </ul>
    </section>
  )
}

interface MilestoneCardProps {
  milestone: MilestoneItem
  locale: 'fr' | 'en'
  t: ReturnType<typeof getAscensionText>
}

function MilestoneCard({ milestone: m, locale, t }: MilestoneCardProps) {
  const title = locale === 'fr' ? m.title_fr : m.title_en
  const metricLabel = t.metric[m.metric] ?? m.metric

  const cardTone = m.earned
    ? 'border-amber-500/40 bg-amber-500/10' // color-allow: amber distinction milestone earned (CLAUDE.md §20 badge UI)
    : 'border-border bg-card opacity-70'

  const titleTone = m.earned
    ? 'text-amber-700 dark:text-amber-300' // color-allow: amber distinction milestone earned (CLAUDE.md §20)
    : 'text-muted-foreground'

  return (
    <article
      className={`flex h-full flex-col gap-1 rounded-md border p-3 transition-colors ${cardTone}`}
      aria-label={`${title} · ${m.earned ? t.milestonesEarned : t.milestonesLocked}`}
    >
      <header className="flex items-start justify-between gap-2">
        <h3 className={`text-sm font-semibold ${titleTone}`}>{title}</h3>
        <MilestoneStatusBadge earned={m.earned} t={t} />
      </header>
      <p className="text-xs text-muted-foreground">{metricLabel}</p>
      <p className="text-xs text-muted-foreground">
        {interpolate(t.milestonesThreshold, { n: m.threshold })}
      </p>
      {m.condition && (
        <p className="text-2xs italic text-muted-foreground">{m.condition}</p>
      )}
      {m.earned && m.earned_at && (
        <p className="mt-auto text-2xs text-amber-700 dark:text-amber-300"> {/* color-allow: amber distinction milestone earned (CLAUDE.md §20) */}
          {interpolate(t.milestonesEarnedAt, { date: formatAscensionDate(m.earned_at, locale) })}
        </p>
      )}
    </article>
  )
}

interface StatusBadgeProps {
  earned: boolean
  t: ReturnType<typeof getAscensionText>
}

function MilestoneStatusBadge({ earned, t }: StatusBadgeProps) {
  const label = earned ? t.milestonesEarned : t.milestonesLocked
  const tone = earned
    ? 'bg-amber-500/20 text-amber-700 dark:text-amber-300' // color-allow: amber badge earned (CLAUDE.md §20 badge UI)
    : 'bg-muted text-muted-foreground'
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-2xs font-medium ${tone}`}
    >
      {label}
    </span>
  )
}
