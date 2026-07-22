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
import { useEffect, useRef } from 'react'
import { useSearch } from '@tanstack/react-router'
import { useAppShellStore } from '@/stores/appShellStore'
import { getAscensionText } from './i18n'
import { useChallenges, useChallengeHistory } from '@/features/prestige/hooks'
import { StatsGlobales } from '@/features/prestige/components/StatsGlobales'
import { MomentCard } from '@/features/prestige/components/MomentCard'
import type { Challenge } from '@/lib/prestige'
import { StreakDashboard } from './StreakDashboard'
import { RecordsTimeline } from './RecordsTimeline'
import { MilestonesGrid } from './MilestonesGrid'
import { HistorySection } from './HistorySection'
import { PrestigeSquadProgress } from './PrestigeSquadProgress'

export function AscensionRealisationsTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)

  // Ancrage AM-5 : les notifs objective_completed/challenge_completed passent
  // l'id de l'item complété → on surligne et scrolle sa carte moment.
  const search = useSearch({ strict: false }) as {
    selectedObjectiveId?: string
    selectedChallengeId?: string
  }
  const selectedId = search.selectedChallengeId ?? search.selectedObjectiveId

  // Défis actifs + terminaux : StatsGlobales veut l'ensemble (« complétés /
  // créés »), la célébration et l'historique n'exploitent que les terminaux.
  const { data: activeData } = useChallenges(playerSlug, titleSlug)
  const { data: historyData } = useChallengeHistory(playerSlug, titleSlug)

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {getAscensionText(locale).realisationsSelectPlayer}
      </p>
    )
  }

  const activeChallenges: Challenge[] = activeData?.challenges ?? []
  const pastChallenges: Challenge[] = historyData?.challenges ?? []
  const allChallenges = [...activeChallenges, ...pastChallenges]
  const completed = pastChallenges
    .filter((c) => c.status === 'completed' && c.completed_at)
    .sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? ''))

  return (
    <div className="space-y-6">
      <PrestigeSquadProgress />

      <StreakDashboard playerSlug={playerSlug} />
      <RecordsTimeline playerSlug={playerSlug} />
      <MilestonesGrid playerSlug={playerSlug} />

      <HistorySection playerSlug={playerSlug} />

      <StatsGlobales challenges={allChallenges} />

      <MomentsSection completed={completed} locale={locale} selectedId={selectedId} />
    </div>
  )
}

// ─── Moments marquants ──────────────────────────────────────────────────────

interface MomentsSectionProps {
  completed: Challenge[]
  locale: 'fr' | 'en'
  /** id de l'item complété à ancrer/surligner (AM-5), depuis la notif. */
  selectedId?: string
}

function MomentsSection({ completed, locale, selectedId }: MomentsSectionProps) {
  const t = getAscensionText(locale)
  const selectedRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (selectedId && selectedRef.current) {
      selectedRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, [selectedId])

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
          {completed.map((c) => {
            const isSelected = !!selectedId && c.id === selectedId
            return (
              <div
                key={c.id}
                ref={isSelected ? selectedRef : undefined}
                className={
                  isSelected
                    ? 'rounded-lg ring-2 ring-primary ring-offset-2 ring-offset-background'
                    : undefined
                }
              >
                <MomentCard
                  challenge={c}
                  achievedValue={c.target}
                  matchCount={0}
                  compact
                />
              </div>
            )
          })}
        </div>
      )}
    </section>
  )
}
