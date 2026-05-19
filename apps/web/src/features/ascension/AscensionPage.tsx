/**
 * Page Ascension (V2 progression) — agrège les vues Streaks / Records /
 * Milestones pour un joueur donné.
 *
 * Cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { getAscensionText } from './i18n'
import { StreakDashboard } from './StreakDashboard'
import { RecordsTimeline } from './RecordsTimeline'
import { MilestonesGrid } from './MilestonesGrid'

export function AscensionPage() {
  const locale = useAppShellStore((s) => s.locale)
  const t = getAscensionText(locale)
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''

  if (!playerSlug) {
    return null
  }

  return (
    <main className="container mx-auto max-w-6xl space-y-8 px-4 py-6">
      <h1 className="text-2xl font-bold">{t.pageTitle}</h1>
      <StreakDashboard playerSlug={playerSlug} />
      <RecordsTimeline playerSlug={playerSlug} />
      <MilestonesGrid playerSlug={playerSlug} />
    </main>
  )
}
