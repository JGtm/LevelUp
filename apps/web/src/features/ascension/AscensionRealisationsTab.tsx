// cross-feature-allow: tab Réalisations Ascension — agrège StatsGlobales,
// MomentCard, PRESTIGE_LEVEL_NAMES_FALLBACK depuis features/prestige et
// useAssetLabel depuis lib/i18n.
/**
 * AscensionRealisationsTab — tab "Réalisations" (refonte 2026-05-26).
 *
 * Composition (du global au détaillé) :
 *   1. PrestigeBadge      — niveau + total PP
 *   2. StreakDashboard    — séries actives / cassées
 *   3. RecordsTimeline    — records personnels (PB)
 *   4. MilestonesGrid     — paliers franchis
 *   5. StatsGlobales      — compteurs (challenges terminés, PP gagnés…)
 *   6. Moments marquants  — historique MomentCards
 *
 * Tout le contenu rétrospectif de l'ancienne page Parcours + Séries.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { useChallenges, useMyPrestige } from '@/features/prestige/hooks'
import { StatsGlobales } from '@/features/prestige/components/StatsGlobales'
import { MomentCard } from '@/features/prestige/components/MomentCard'
import { PRESTIGE_LEVEL_NAMES_FALLBACK } from '@/features/prestige/fallback.i18n'
import { useAssetLabel } from '@/lib/i18n/fieldMappings'
import type { Challenge, UserPrestige } from '@/lib/prestige'
import { StreakDashboard } from './StreakDashboard'
import { RecordsTimeline } from './RecordsTimeline'
import { MilestonesGrid } from './MilestonesGrid'

const TITLE_SLUG = 'halo_infinite'

export function AscensionRealisationsTab() {
  const currentPlayer = useAppShellStore((s) => s.currentPlayer)
  const playerSlug = currentPlayer?.player_slug ?? ''
  const locale = useAppShellStore((s) => s.locale)

  const { data: prestige, isError: prestigeErr } = useMyPrestige(playerSlug, TITLE_SLUG)
  const { data: challengesData } = useChallenges(playerSlug, TITLE_SLUG)

  if (!playerSlug) {
    return (
      <p className="p-6 text-sm text-muted-foreground">
        {locale === 'en' ? 'Select a player.' : 'Sélectionne un joueur.'}
      </p>
    )
  }

  const challenges: Challenge[] = challengesData?.challenges ?? []
  const completed = challenges
    .filter((c) => c.status === 'completed' && c.completed_at)
    .sort((a, b) => (b.completed_at ?? '').localeCompare(a.completed_at ?? ''))

  return (
    <div className="space-y-6">
      {!prestigeErr && <PrestigeBadge prestige={prestige} locale={locale} />}

      <StreakDashboard playerSlug={playerSlug} />
      <RecordsTimeline playerSlug={playerSlug} />
      <MilestonesGrid playerSlug={playerSlug} />

      <StatsGlobales challenges={challenges} />

      <MomentsSection completed={completed} locale={locale} />
    </div>
  )
}

// ─── PrestigeBadge ───────────────────────────────────────────────────────────

function PrestigeBadge({ prestige, locale }: { prestige?: UserPrestige; locale: 'fr' | 'en' }) {
  const totalPP = prestige?.total_pp ?? 0
  const level = prestige?.current_level ?? 0
  const levelKey = String(level)
  const levelLabel = useAssetLabel('prestige_level', levelKey)
  const displayLevelName =
    levelLabel !== levelKey
      ? levelLabel
      : PRESTIGE_LEVEL_NAMES_FALLBACK[level] ?? PRESTIGE_LEVEL_NAMES_FALLBACK[0]
  const ppFormat = totalPP.toLocaleString(locale === 'en' ? 'en-US' : 'fr-FR')

  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xs uppercase tracking-widest text-muted-foreground">
            {locale === 'en' ? 'Prestige tier' : 'Niveau Prestige'}
          </h2>
          <p className="text-2xl font-bold">{displayLevelName}</p>
        </div>
        <div className="text-right">
          <h2 className="text-xs uppercase tracking-widest text-muted-foreground">
            {locale === 'en' ? 'Prestige Points' : 'Points de Prestige'}
          </h2>
          <p className="text-2xl font-bold">{ppFormat} PP</p>
        </div>
      </div>
    </div>
  )
}

// ─── Moments marquants ──────────────────────────────────────────────────────

function MomentsSection({ completed, locale }: { completed: Challenge[]; locale: 'fr' | 'en' }) {
  return (
    <section className="rounded-lg border border-border bg-card p-4">
      <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-muted-foreground">
        {locale === 'en' ? 'Highlights' : 'Moments marquants'}
      </h2>
      {completed.length === 0 ? (
        <div className="rounded-lg border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          {locale === 'en'
            ? 'Moment cards will appear here as you complete your first objectives.'
            : 'Les moment cards apparaîtront ici à la validation de tes premiers objectifs.'}
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
