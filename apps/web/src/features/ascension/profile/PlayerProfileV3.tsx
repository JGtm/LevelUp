/**
 * PlayerProfileV3 — orchestrateur des sections A1/A2/B du profil joueur.
 *
 * Refonte Ascension : remplace `prestige/components/PlayerProfileCard` (V1)
 * et `ascension/GameProfileSection` (V2). Sections rendues :
 * - Identité (radar 6 axes + rôles)
 * - Style/Discipline (FK/FD + engagement)
 * - Performance (tier LUSR + composantes + tendance μ)
 *
 * Restructuration 4 onglets (2026-07, DEC-3) : la section « Pistes de
 * progression » (ProgressionSection : leviers + défis suggérés + CTA campagne)
 * a été extraite vers l'onglet « Entraînement » (AscensionCoachingTab). Ce
 * composant ne porte plus que l'identité, le style et la performance —
 * rendus dans l'onglet « Profil » (index).
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4 + §5.2.
 */
import { usePlayerProfile } from './queries'
import { useProfileI18n } from './useProfileI18n'
import { IdentitySection } from './IdentitySection'
import { InsufficientDataPlaceholder } from './InsufficientDataPlaceholder'
import { PerformanceSection } from './PerformanceSection'
import { StyleDisciplineSection } from './StyleDisciplineSection'

const MIN_MATCHES_FOR_PROFILE = 30

interface PlayerProfileV3Props {
  playerSlug: string
  windowDays?: number
}

export function PlayerProfileV3({ playerSlug, windowDays = 30 }: PlayerProfileV3Props) {
  const { data: profile, isLoading, isError } = usePlayerProfile(playerSlug, windowDays)
  const { t } = useProfileI18n()

  if (isLoading) {
    return (
      <section className="rounded-lg border border-border bg-card p-6">
        <p className="text-sm text-muted-foreground">{t('profile.loading')}</p>
      </section>
    )
  }
  if (isError || !profile) {
    return (
      <section className="rounded-lg border border-dashed border-border bg-card p-6">
        <p className="text-sm text-muted-foreground">{t('profile.load_error')}</p>
      </section>
    )
  }

  if (!profile.has_enough_data) {
    return (
      <InsufficientDataPlaceholder
        matchesAnalyzed={profile.matches_analyzed}
        required={MIN_MATCHES_FOR_PROFILE}
      />
    )
  }

  return (
    <div className="space-y-4">
      <IdentitySection profile={profile} />
      <StyleDisciplineSection
        style={profile.style_signature}
        engagement={profile.engagement_snap}
      />
      <PerformanceSection
        skillRating={profile.skill_rating}
        components={profile.lusr_components}
        muTrend={profile.mu_trend}
        skillTrend={profile.skill_trend}
      />
    </div>
  )
}
