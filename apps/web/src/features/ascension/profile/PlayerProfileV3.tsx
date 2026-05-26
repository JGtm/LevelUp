/**
 * PlayerProfileV3 — orchestrateur des sections A1/A2/B/C du profil joueur.
 *
 * Refonte Ascension : remplace `prestige/components/PlayerProfileCard` (V1)
 * et `ascension/GameProfileSection` (V2). Aucune régression :
 * - reprend toutes les sections V1 (Identité, Style/Discipline,
 *   Performance avec mu_trend et progress tier, Progression avec CTA
 *   "Démarrer campagne" et "Lancer template")
 * - ajoute le trend icon par composante LUSR (apport V2)
 *
 * Les patterns V2 (PatternContextGrid, BehaviorAlertList, LeverList)
 * restent rendus séparément par le tab Profil & objectifs.
 *
 * Cf. PLAN_PLAYER_PROFILE_ASCENSION.md §4 + §5.2.
 */
import { usePlayerProfile } from './queries'
import { useProfileI18n } from './useProfileI18n'
import { IdentitySection } from './IdentitySection'
import { InsufficientDataPlaceholder } from './InsufficientDataPlaceholder'
import { PerformanceSection } from './PerformanceSection'
import { ProgressionSection } from './ProgressionSection'
import { StyleDisciplineSection } from './StyleDisciplineSection'

const MIN_MATCHES_FOR_PROFILE = 30

interface PlayerProfileV3Props {
  playerSlug: string
  windowDays?: number
  onStartCampaign?: (axis: string, axisKind: 'radar' | 'lusr_component') => void
  onLaunchTemplate?: (templateId: string) => void
}

export function PlayerProfileV3({
  playerSlug,
  windowDays = 30,
  onStartCampaign,
  onLaunchTemplate,
}: PlayerProfileV3Props) {
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
