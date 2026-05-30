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
import { Card, CardContent } from '@/components/ui/card'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetProfile } from '@/lib/api/types'
import { ExplorerTargetIdentityBanner } from './ExplorerTargetIdentityBanner'
import { ExplorerTargetCareerStats } from './ExplorerTargetCareerStats'
import { ExplorerTargetSampleStats } from './ExplorerTargetSampleStats'
import { ExplorerTargetMedals } from './ExplorerTargetMedals'
import { ExplorerTargetSeasonCSR } from './ExplorerTargetSeasonCSR'
import { ExplorerTargetSeasonMatches } from './ExplorerTargetSeasonMatches'

interface ExplorerTargetProfileCardProps {
  profile: ExplorerTargetProfile
  gamertag: string
}

/** formatPlaytime convertit des secondes en "Xj Yh" / "Yh Zm" / "Zm" (FR/EN). */
function formatPlaytime(seconds: number, locale: string): string {
  const totalMin = Math.floor(seconds / 60)
  const days = Math.floor(totalMin / 1440)
  const hours = Math.floor((totalMin % 1440) / 60)
  const mins = totalMin % 60
  const dUnit = locale === 'en' ? 'd' : 'j'
  if (days > 0) return `${days}${dUnit} ${hours}h`
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

export function ExplorerTargetProfileCard({ profile, gamertag }: ExplorerTargetProfileCardProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, appLocale, values)

  const identity = profile.identity ?? null
  const careerStats = profile.career_stats ?? null
  const sampleStats = profile.sample_stats ?? null
  const topMedals = profile.top_medals ?? []
  const seasonCSRs = profile.season_csrs ?? []
  const matchesPerSeason = profile.matches_per_season ?? []
  const showNoAuthHint = !profile.auth_available && careerStats == null
  const showSample = sampleStats != null && sampleStats.sample_size > 0
  const timePlayed = careerStats?.time_played_seconds ?? 0

  return (
    <div className="flex flex-col gap-4" data-testid="explorer-target-profile-card">
      <ExplorerTargetIdentityBanner
        identity={identity}
        gamertag={gamertag}
        identityUnavailableLabel={t('explorer.target_profile.identity_unknown_title')}
        identityUnavailableDescription={t('explorer.target_profile.identity_unknown_description')}
      />

      {careerStats != null && <ExplorerTargetCareerStats careerStats={careerStats} />}

      {timePlayed > 0 && (
        <Card data-testid="explorer-target-time-played">
          <CardContent className="flex items-baseline justify-between py-3">
            <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t('explorer.target_profile.time_played_label')}
            </span>
            <span className="text-lg font-bold tabular-nums text-foreground">
              {formatPlaytime(timePlayed, appLocale)}
            </span>
          </CardContent>
        </Card>
      )}

      {seasonCSRs.length > 0 && (
        <ExplorerTargetSeasonCSR
          csrs={seasonCSRs}
          title={t('explorer.target_profile.season_csr_title')}
        />
      )}

      {matchesPerSeason.length > 0 && (
        <ExplorerTargetSeasonMatches
          seasons={matchesPerSeason}
          title={t('explorer.target_profile.matches_per_season_title')}
        />
      )}

      {topMedals.length > 0 && <ExplorerTargetMedals medals={topMedals} />}

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
