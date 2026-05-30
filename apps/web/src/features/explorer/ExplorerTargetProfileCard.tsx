/**
 * ExplorerTargetProfileCard — encart "Profil joueur cible" en haut des
 * résultats Explorer mode Joueur.
 *
 * Logique local-first : l'identité vient de la DB locale de la cible si elle est
 * un joueur suivi (sinon gamertag seul) ; la carrière agrégée est le seul fetch
 * live. Aucune privacy (supprimée — bruit sans valeur). Sous-blocs :
 *  - ExplorerTargetIdentityBanner (bandeau hero ; gamertag seul si non suivi)
 *  - ExplorerTargetCareerStats (si career_stats dispo)
 *  - ExplorerTargetSampleStats (si sample_stats dispo et sample_size > 0)
 *
 * Cas no-tokens (auth_available=false) :
 *  - Identity locale toujours résolue (indépendante des tokens)
 *  - CareerStats masquée + hint "Connexion Halo requise"
 *  - SampleStats reste affiché (calcul local)
 */
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
