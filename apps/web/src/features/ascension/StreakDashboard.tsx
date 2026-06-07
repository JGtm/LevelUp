/**
 * Vue détaillée des streaks d'un joueur (page Ascension).
 *
 * Affiche une carte par streak (active + historique) via `StreakCard` (mode
 * complet). Met en avant la longueur courante, le multiplicateur PP, les
 * boucliers disponibles et le prochain palier. La carte est partagée avec le
 * widget home (`HomeAscensionWidget`, mode compact) pour éviter un fork.
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §4 + §8.3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useStreaks } from './queries'
import { getAscensionText } from './i18n'
import { StreakCard } from './StreakCard'
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

function statusOrder(s: Streak): number {
  if (s.status === 'active') return 0
  if (s.status === 'paused') return 1
  return 2
}
