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
import { useCapability } from '@/lib/capabilities/capabilities'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerEncounterStats, ExplorerTargetProfile } from '@/lib/api/types'
import { ExplorerTargetIdentityBanner } from './ExplorerTargetIdentityBanner'
import { ExplorerTargetCareerStats } from './ExplorerTargetCareerStats'
import { ExplorerTargetSampleStats, ExplorerTargetSampleKpis, ExplorerTargetOutcome } from './ExplorerTargetSampleStats'
import { ExplorerTargetCadence } from './ExplorerTargetCadence'
import { ExplorerTargetVersusDonuts } from './ExplorerTargetVersusDonuts'
import { ExplorerTargetMedals } from './ExplorerTargetMedals'
import { ExplorerTargetSeasonCSR } from './ExplorerTargetSeasonCSR'
import { ExplorerTargetSeasonMatches } from './ExplorerTargetSeasonMatches'
import { ExplorerLiveStatusBadge } from './ExplorerLiveStatusBadge'

interface ExplorerTargetProfileCardProps {
  profile: ExplorerTargetProfile
  gamertag: string
  /** Stats de rencontre (WR ensemble/face à lui, moyenne perso, écart de frags) —
   *  alimente les donuts + la courbe rendus en fin de section « matchs joués
   *  ensemble ». Optionnel : sans lui, cette dernière rangée n'est pas rendue. */
  encounterStats?: ExplorerEncounterStats | null
}

export function ExplorerTargetProfileCard({ profile, gamertag, encounterStats }: ExplorerTargetProfileCardProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  // Classements CSR = surface "ranked" : masquée pour un titre sans rang
  // (fail-open mono-titre, NO-OP halo_infinite qui déclare 'ranked').
  const hasRanked = useCapability('ranked')
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
  // Top médailles rendu à côté du donut "Répartition des modes" (ExplorerCombatProfile)
  // quand un profil de combat existe ; sinon repli ici pour ne jamais les perdre.
  const hasCombatProfile = (profile.combat_profile?.length ?? 0) > 0
  // Statuts par section live (Lot A3 — fin de la dégradation muette). Champ
  // optionnel côté type (fixtures/tests antérieurs) : undefined → aucun badge
  // (ExplorerLiveStatusBadge est nil-safe).
  const liveStatus = profile.live_status
  const identityStatus = liveStatus?.identity
  const careerStatus = liveStatus?.career
  const seasonCSRsStatus = liveStatus?.season_csrs
  const seasonsStatus = liveStatus?.seasons
  // Career et top médailles partagent le même fetch live (service record) : pas
  // de statut dédié "top_medals" côté DTO, la raison d'un vide est déjà portée
  // par le badge de la section Carrière ci-dessous (pas de second badge
  // redondant — ils seraient TOUJOURS simultanés : top_medals n'est jamais
  // rempli sans career_stats, cf. fetchTargetServiceRecord côté Go).
  const showCareerSection = careerStats != null || (!!careerStatus && careerStatus !== 'ok')

  return (
    <div className="flex flex-col gap-4" data-testid="explorer-target-profile-card">
      <ExplorerTargetIdentityBanner
        identity={identity}
        gamertag={gamertag}
        identityUnavailableLabel={t('explorer.target_profile.identity_unknown_title')}
        identityUnavailableDescription={t('explorer.target_profile.identity_unknown_description')}
      />
      {/* Identité indisponible ET raison connue (auth/erreur/repli) : badge discret
          sous le bandeau plutôt qu'un silence — A3. */}
      {identity == null && identityStatus && identityStatus !== 'ok' && (
        <div className="flex justify-end">
          <ExplorerLiveStatusBadge status={identityStatus} />
        </div>
      )}

      {/* Carrière complète : titre en en-tête de section hors bloc (style "Profil
          de combat"). La section reste rendue (header + badge) même sans
          career_stats quand le statut explique pourquoi (A3) — jamais un vide
          silencieux entre l'identité et les saisons. */}
      {showCareerSection && (
        <section className="space-y-3">
          <header className="flex items-center gap-2">
            <h3 className="text-base font-semibold text-foreground">
              {t('explorer.target_profile.section_career_title')}
            </h3>
            <ExplorerLiveStatusBadge status={careerStatus} />
          </header>
          {careerStats != null && <ExplorerTargetCareerStats careerStats={careerStats} />}
        </section>
      )}

      {/* Section saisons (placeholders titrés si vide) : Classements CSR (1/3,
          gauche, surface 'ranked') + Matchs par saison (2/3, droite). Le bloc CSR
          est masqué pour un titre sans rang ; les matchs par saison passent alors
          pleine largeur (pas de colonne vide). Chaque sous-bloc gère son état vide. */}
      <div className="grid gap-4 lg:grid-cols-3">
        {hasRanked && (
          <ExplorerTargetSeasonCSR
            csrs={seasonCSRs}
            title={t('explorer.target_profile.season_csr_title')}
            emptyMessage={t('explorer.target_profile.season_csr_empty')}
            liveStatus={seasonCSRsStatus}
          />
        )}
        <div className={hasRanked ? 'lg:col-span-2' : 'lg:col-span-3'}>
          <ExplorerTargetSeasonMatches
            seasons={matchesPerSeason}
            title={t('explorer.target_profile.matches_per_season_title')}
            liveStatus={seasonsStatus}
          />
        </div>
      </div>

      {/* Top médailles : repli ici uniquement si pas de profil de combat (sinon
          rendu à côté du donut Répartition des modes, cf. ExplorerCombatProfile).
          Vide sans raison distincte à afficher : le badge de la section Carrière
          ci-dessus couvre déjà ce cas (même fetch, cf. commentaire showCareerSection). */}
      {topMedals.length > 0 && !hasCombatProfile && <ExplorerTargetMedals medals={topMedals} />}

      {showNoAuthHint && (
        <div
          className="rounded-md border border-dashed border-border bg-muted/30 px-4 py-3 text-sm text-muted-foreground"
          data-testid="explorer-target-no-auth-hint"
        >
          {t('explorer.target_profile.no_auth_hint')}
        </div>
      )}

      {/* Cible résolue (identité ou carrière présente) mais aucun match en commun
          (sample_size == 0) : au lieu de masquer la section en silence (l'utilisateur
          croit à un bug), on rend une note discrète expliquant que les comparaisons
          directes viendront avec des parties partagées (V72-21). */}
      {!showSample && (identity != null || careerStats != null) && (
        <p
          className="rounded-md border border-dashed border-border bg-muted/20 px-4 py-3 text-sm text-muted-foreground"
          data-testid="explorer-target-no-shared-matches"
        >
          {t('explorer.target_profile.section_sample_empty')}
        </p>
      )}

      {/* "Sur N matchs joués ensemble" : titre en en-tête de section hors bloc
          (style "Profil de combat"), puis stats (2/3) + Cadence (1/3) à droite. */}
      {showSample && sampleStats && (
        <section className="space-y-3">
          <header>
            <h3 className="text-base font-semibold text-foreground">
              {t('explorer.target_profile.section_sample_title', { count: sampleStats.sample_size })}
            </h3>
          </header>
          {/* Rangée de KPI cards sous le titre (parité "Carrière complète"). */}
          <ExplorerTargetSampleKpis sampleStats={sampleStats} />
          <div className="grid gap-4 lg:grid-cols-3">
            {/* Colonne gauche (2/3) : donut "Répartition des frags" + bilan V/N/D empilés. */}
            <div className="flex flex-col gap-4 lg:col-span-2">
              <ExplorerTargetSampleStats sampleStats={sampleStats} />
              <ExplorerTargetOutcome sampleStats={sampleStats} />
            </div>
            <div className="lg:col-span-1">
              <ExplorerTargetCadence sampleStats={sampleStats} />
            </div>
          </div>

          {/* En DERNIER : donuts « taux de victoires ensemble / face à lui »
              (repère = moyenne perso historique) + écart de frags cumulé. Briques
              réutilisées du hub Relations. Rendu seulement si encounter_stats fourni. */}
          {encounterStats && <ExplorerTargetVersusDonuts encounterStats={encounterStats} />}
        </section>
      )}
    </div>
  )
}
