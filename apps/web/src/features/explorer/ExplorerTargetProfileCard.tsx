/**
 * ExplorerTargetProfileCard — encart "Profil joueur cible" en haut des
 * résultats Explorer mode Joueur.
 *
 * Compose 4 sous-blocs :
 *  - ExplorerTargetIdentityBanner (bandeau hero)
 *  - PrivacyBanner conditionnel (si privacy_warning != none)
 *  - ExplorerTargetCareerStats (si career_stats dispo)
 *  - ExplorerTargetSampleStats (si sample_stats dispo et sample_size > 0)
 *
 * Cas no-tokens (auth_available=false) :
 *  - Identity peut venir de la DB locale ou placeholder
 *  - CareerStats masquée + hint "Connexion Halo requise"
 *  - SampleStats reste affiché (calcul local indépendant des tokens)
 */
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetProfile } from '@/lib/api/types'
import { ExplorerTargetIdentityBanner } from './ExplorerTargetIdentityBanner'
import { ExplorerTargetCareerStats } from './ExplorerTargetCareerStats'
import { ExplorerTargetSampleStats } from './ExplorerTargetSampleStats'

interface ExplorerTargetProfileCardProps {
  profile: ExplorerTargetProfile
  gamertag: string
}

export function ExplorerTargetProfileCard({ profile, gamertag }: ExplorerTargetProfileCardProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, appLocale, values)

  const identity = profile.identity ?? null
  const careerStats = profile.career_stats ?? null
  const sampleStats = profile.sample_stats ?? null
  const privacyWarning = profile.privacy_warning ?? null
  const showPrivacyBanner = privacyWarning != null && privacyWarning.level !== 'none'
  const showNoAuthHint = !profile.auth_available && careerStats == null
  const showSample = sampleStats != null && sampleStats.sample_size > 0

  return (
    <div className="flex flex-col gap-4" data-testid="explorer-target-profile-card">
      <ExplorerTargetIdentityBanner
        identity={identity}
        gamertag={gamertag}
        identityUnavailableLabel={t('explorer.target_profile.identity_unknown_title')}
        identityUnavailableDescription={t('explorer.target_profile.identity_unknown_description')}
      />

      {showPrivacyBanner && <PrivacyBanner warning={privacyWarning} />}

      {careerStats != null && <ExplorerTargetCareerStats careerStats={careerStats} />}

      {showNoAuthHint && (
        <div
          className="rounded-md border border-dashed border-border bg-muted/30 px-4 py-3 text-sm text-muted-foreground"
          data-testid="explorer-target-no-auth-hint"
        >
          {t('explorer.target_profile.no_auth_hint')}
        </div>
      )}

      {showSample && <ExplorerTargetSampleStats sampleStats={sampleStats} />}
    </div>
  )
}
