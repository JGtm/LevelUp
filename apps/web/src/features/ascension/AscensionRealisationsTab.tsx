// cross-feature-allow: tab Réalisations Ascension — agrège StatsGlobales,
// MomentCard et useChallenges depuis features/prestige.
/**
 * AscensionRealisationsTab — tab "Réalisations" (refonte 2026-05-26).
 *
 * Composition (du global au détaillé) :
 *   1. PrestigeSquadProgress — barre progression PP (moi + escouade)
 *   2. StreakDashboard    — séries actives / cassées
 *   3. RecordsTimeline    — records personnels (PB)
 *   4. MilestonesGrid     — paliers franchis
 *   5. StatsGlobales      — compteurs (challenges terminés, PP gagnés…)
 *   6. Moments marquants  — historique MomentCards
 *
 * Tout le contenu rétrospectif de l'ancienne page Parcours + Séries.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { getAscensionText } from './i18n'
import { useChallenges } from '@/features/prestige/hooks'
import { StatsGlobales } from '@/features/prestige/components/StatsGlobales'
import { MomentCard } from '@/features/prestige/components/MomentCard'
import type { Challenge } from '@/lib/prestige'
import { StreakDashboard } from './StreakDashboard'
import { RecordsTimeline } from './RecordsTimeline'
import { MilestonesGrid } from './MilestonesGrid'
import { PrestigeSquadProgress } from './PrestigeSquadProgress'

export function AscensionRealisationsTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)

  const { data: challengesData } = useChallenges(playerSlug, titleSlug)

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {getAscensionText(locale).realisationsSelectPlayer}
      </p>
    )
  }

  const challenges: Challenge[] = challengesData?.challenges ?? []
  const completed = challenges
    .filter((c) => c.status === 'completed' && c.completed_at)
    .sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? ''))

  return (
    <div className="space-y-6">
      <PrestigeSquadProgress />

      <StreakDashboard playerSlug={playerSlug} />
      <RecordsTimeline playerSlug={playerSlug} />
      <MilestonesGrid playerSlug={playerSlug} />

      <StatsGlobales challenges={challenges} />

      <MomentsSection completed={completed} locale={locale} />
    </div>
  )
}

// ─── Moments marquants ──────────────────────────────────────────────────────

function MomentsSection({ completed, locale }: { completed: Challenge[]; locale: 'fr' | 'en' }) {
  const t = getAscensionText(locale)
  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {t.realisationsHighlights}
      </h2>
      {completed.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {t.realisationsEmpty}
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {completed.map((c) => (
            <MomentCard
              key={c.id}
              challenge={c}
              achievedValue={c.target}
              matchCount={0}
              compact
            />
          ))}
        </div>
      )}
    </section>
  )
}
