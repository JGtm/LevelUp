/**
 * PlayerProfileCard — Sections A1/A2/B/C du profil joueur V1.
 *
 * Orchestre les 5 sous-composants. Affiche le placeholder si has_enough_data=false.
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4 + §5.2.
 */
import { usePlayerProfile } from '../hooks/usePlayerProfile'
import { useProfileI18n } from '../hooks/useProfileI18n'
import { IdentitySection } from './profile/IdentitySection'
import { InsufficientDataPlaceholder } from './profile/InsufficientDataPlaceholder'
import { PerformanceSection } from './profile/PerformanceSection'
import { ProgressionSection } from './profile/ProgressionSection'
import { StyleDisciplineSection } from './profile/StyleDisciplineSection'

const MIN_MATCHES_FOR_PROFILE = 30

interface PlayerProfileCardProps {
  playerSlug: string
  windowDays?: number
  onStartCampaign?: (axis: string, axisKind: 'radar' | 'lusr_component') => void
  onLaunchTemplate?: (templateId: string) => void
}

export function PlayerProfileCard({
  playerSlug,
  windowDays = 30,
  onStartCampaign,
  onLaunchTemplate,
}: PlayerProfileCardProps) {
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
      />
      <ProgressionSection
        leverages={profile.leverages}
        suggestions={profile.suggested_challenges}
        onStartCampaign={onStartCampaign}
        onLaunchTemplate={onLaunchTemplate}
      />
    </div>
  )
}
