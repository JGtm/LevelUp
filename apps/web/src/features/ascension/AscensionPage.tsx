/**
 * Page Ascension (V2 progression) — agrège les vues Streaks / Records /
 * Milestones pour un joueur donné.
 *
 * Pour le commit 8, seuls les Streaks sont rendus. Records + Milestones
 * arrivent au commit 9.
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { getAscensionText } from './i18n'
import { StreakDashboard } from './StreakDashboard'

export function AscensionPage() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''

  if (!playerSlug) {
    return null
  }

  return (
    <main className="container mx-auto max-w-6xl px-4 py-6">
      <h1 className="mb-6 text-2xl font-bold">{t.pageTitle}</h1>
      <StreakDashboard playerSlug={playerSlug} />
    </main>
  )
}
