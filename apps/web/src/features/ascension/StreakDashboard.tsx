/**
 * Vue détaillée des streaks d'un joueur (page Ascension).
 *
 * Affiche un card par streak (active + historique). Met en avant la longueur
 * courante, le multiplicateur PP, les shields disponibles, et le prochain palier.
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §4 + §8.3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useStreaks } from './queries'
import { getAscensionText } from './i18n'
import { formatDate, formatMultiplier, interpolate, nextPPTier } from './format'
import type { Streak } from './types'

export interface StreakDashboardProps {
  playerSlug: string
}

export function StreakDashboard({ playerSlug }: StreakDashboardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const { data, isLoading, isError } = useStreaks(playerSlug)

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
  // Tri : active d'abord, puis paused, puis broken (par best_length desc).
  const sorted = [...items].sort((a, b) => statusOrder(a) - statusOrder(b) || b.best_length - a.best_length)

  if (sorted.length === 0) {
    return (
      <div className="rounded-md border border-border bg-card p-6 text-center text-muted-foreground">
        {t.streaksEmpty}
      </div>
    )
  }

  return (
    <section aria-labelledby="streaks-section-heading">
      <h2 id="streaks-section-heading" className="mb-3 text-lg font-semibold">
        {t.streaksSectionTitle}
      </h2>
      <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {sorted.map((s) => (
          <li key={s.id}>
            <StreakCard streak={s} locale={locale} t={t} />
          </li>
        ))}
      </ul>
    </section>
  )
}

interface StreakCardProps {
  streak: Streak
  locale: 'fr' | 'en'
  t: ReturnType<typeof getAscensionText>
}

function StreakCard({ streak: s, locale, t }: StreakCardProps) {
  const statusLabel = s.status === 'active' ? t.streakActive : s.status === 'paused' ? t.streakPaused : t.streakBroken
  const statusToneClass =
    s.status === 'active'
      ? 'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300' // color-allow: emerald status badge active (CLAUDE.md §20 badge UI)
      : s.status === 'paused'
        ? 'bg-amber-500/10 text-amber-700 dark:text-amber-300' // color-allow: amber status badge paused (CLAUDE.md §20)
        : 'bg-muted text-muted-foreground'

  const nextTier = s.status !== 'broken' ? nextPPTier(s.current_length) : null

  return (
    <article className="flex flex-col gap-2 rounded-md border border-border bg-card p-4">
      <header className="flex items-start justify-between gap-2">
        <h3 className="text-sm font-medium">{t.streakTypeName[s.type]}</h3>
        <span
          className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${statusToneClass}`}
        >
          {statusLabel}
        </span>
      </header>

      <div className="flex items-baseline gap-2">
        <span className="text-3xl font-bold">{s.current_length}</span>
        <span className="text-xs text-muted-foreground">
          {interpolate(t.streakCurrentLength, { n: s.current_length, plural: '' }).replace(
            String(s.current_length),
            '',
          ).trim()}
        </span>
      </div>

      <p className="text-xs text-muted-foreground">
        {interpolate(t.streakBestLength, { n: s.best_length })}
      </p>

      <dl className="mt-1 space-y-1 text-xs">
        <div>
          <dt className="inline text-muted-foreground">{interpolate(t.streakPPMultiplier, {
            value: formatMultiplier(s.pp_multiplier).replace('×', ''),
          })}</dt>
        </div>
        {nextTier && s.status === 'active' && (
          <div className="text-muted-foreground">
            {interpolate(t.streakNextMilestone, {
              n: nextTier.length,
              mul: formatMultiplier(nextTier.multiplier).replace('×', ''),
            })}
          </div>
        )}
        {!nextTier && s.status === 'active' && (
          <div className="text-emerald-700 dark:text-emerald-300"> {/* color-allow: emerald streak active at max multiplier (CLAUDE.md §20) */}
            {t.streakAtMaxMultiplier}
          </div>
        )}
        <div className="text-muted-foreground">
          {interpolate(t.streakShieldsAvailable, {
            n: s.shields_available - s.shields_used,
          })}
        </div>
        {s.status === 'broken' && s.broken_at && (
          <div className="text-muted-foreground">
            {interpolate(t.streakBrokenAt, { date: formatDate(s.broken_at, locale) })}
          </div>
        )}
        <div className="text-muted-foreground">
          {interpolate(t.streakStarted, { date: formatDate(s.started_at, locale) })}
        </div>
      </dl>
    </article>
  )
}

function statusOrder(s: Streak): number {
  if (s.status === 'active') return 0
  if (s.status === 'paused') return 1
  return 2
}
