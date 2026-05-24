/**
 * Page Ascension (V2 progression) — agrège les vues Streaks / Records /
 * Milestones / Profil de jeu / Patterns pour un joueur donné.
 *
 * Structure :
 *   ├── Streaks Dashboard           (existant)
 *   ├── Records Timeline            (existant)
 *   ├── Milestones Grid             (existant)
 *   └── Profil de jeu               (phases 0.2-3 Pattern Engine)
 *       ├── Radar 6 axes + forces/faiblesses  (A1)
 *       ├── Badge style de jeu                (A2)
 *       ├── Composantes LUSR                  (B)
 *       ├── Leviers agrégats + défis          (C)
 *       ├── Patterns contextuels              (phase 1)
 *       ├── Patterns comportementaux          (phase 2)
 *       └── Leviers calibrés                  (phase 3)
 *
 * Cf. PLAN_PATTERN_ENGINE.md §0.2 §1 §2 §3.
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { getAscensionText } from './i18n'
import { StreakDashboard } from './StreakDashboard'
import { RecordsTimeline } from './RecordsTimeline'
import { MilestonesGrid } from './MilestonesGrid'
import { useProfile, usePatterns } from './queries'
import { ProfileRadarSection } from './ProfileRadarSection'
import { StyleBadge } from './StyleBadge'
import { LUSRComponentsGrid } from './LUSRComponentsGrid'
import { LeveragePanel } from './LeveragePanel'
import { PatternContextGrid } from './PatternContextGrid'
import { SquadVsSoloCard } from './SquadVsSoloCard'
import { BehaviorAlertList } from './BehaviorAlertList'
import { LeverList } from './LeverList'

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
      <GameProfileSection playerSlug={playerSlug} locale={locale} />
    </main>
  )
}

// ── Section "Profil de jeu" ───────────────────────────────────────────────────

interface GameProfileSectionProps {
  playerSlug: string
  locale: 'fr' | 'en'
}

function GameProfileSection({ playerSlug, locale }: GameProfileSectionProps) {
  const t = getAscensionText(locale)
  const { data: profile, isLoading: profileLoading } = useProfile(playerSlug)
  const { data: patterns, isLoading: patternsLoading } = usePatterns(playerSlug)

  if (profileLoading) {
    return (
      <section>
        <h2 className="mb-3 text-lg font-semibold">{t.profileSectionTitle}</h2>
        <p className="text-sm text-muted-foreground" role="status">{t.loading}</p>
      </section>
    )
  }

  if (!profile) return null

  const contextPatterns = patterns?.context_patterns ?? []
  const behaviorPatterns = patterns?.behavior_patterns ?? []
  const levers = patterns?.levers ?? []

  return (
    <section aria-labelledby="game-profile-heading">
      <h2 id="game-profile-heading" className="mb-4 text-lg font-semibold">
        {t.profileSectionTitle}
      </h2>

      {!profile.has_enough_data ? (
        <div className="rounded-md border border-border bg-card p-6 text-center text-muted-foreground">
          {t.profileNotEnoughData}
        </div>
      ) : (
        <div className="space-y-6">
          {/* A1 — Radar narrative */}
          {(profile.radar_axes?.length ?? 0) > 0 && (
            <ProfileCard title={null}>
              <ProfileRadarSection profile={profile} t={t} />
            </ProfileCard>
          )}

          {/* A2 — Style + Engagement */}
          <ProfileCard title={null}>
            <StyleBadge
              style={profile.style_signature}
              engagement={profile.engagement_snap}
              t={t}
            />
          </ProfileCard>

          {/* B — Composantes LUSR */}
          {(profile.lusr_components?.length ?? 0) > 0 && (
            <ProfileCard title={null}>
              <LUSRComponentsGrid
                components={profile.lusr_components!}
                skillRating={profile.skill_rating}
                t={t}
              />
            </ProfileCard>
          )}

          {/* C — Leviers agrégats */}
          {((profile.leverages?.length ?? 0) > 0 || (profile.suggested_challenges?.length ?? 0) > 0) && (
            <ProfileCard title={null}>
              <LeveragePanel
                leverages={profile.leverages ?? []}
                challenges={profile.suggested_challenges ?? []}
                locale={locale}
                t={t}
              />
            </ProfileCard>
          )}

          {/* Phase 1 — Patterns contextuels */}
          {!patternsLoading && contextPatterns.length > 0 && (
            <ProfileCard title={t.patternsSectionTitle}>
              <PatternContextGrid patterns={contextPatterns} t={t} />
              <SquadVsSoloCard patterns={contextPatterns} t={t} />
            </ProfileCard>
          )}

          {/* Phase 2 — Patterns comportementaux */}
          {!patternsLoading && behaviorPatterns.length > 0 && (
            <ProfileCard title={t.behaviorsSectionTitle}>
              <BehaviorAlertList patterns={behaviorPatterns} t={t} />
            </ProfileCard>
          )}

          {/* Phase 3 — Leviers calibrés */}
          {!patternsLoading && levers.length > 0 && (
            <ProfileCard title={t.leversSectionTitle}>
              <LeverList levers={levers} t={t} />
            </ProfileCard>
          )}
        </div>
      )}
    </section>
  )
}

function ProfileCard({ title, children }: { title: string | null; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      {title && (
        <h3 className="mb-3 text-sm font-semibold text-muted-foreground uppercase tracking-wide">
          {title}
        </h3>
      )}
      {children}
    </div>
  )
}
