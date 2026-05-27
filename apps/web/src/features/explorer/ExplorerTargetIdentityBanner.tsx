/**
 * ExplorerTargetIdentityBanner — bandeau hero du target dans l'encart Explorer.
 *
 * Version compacte de HomeSpartanIdentityBanner :
 *  - banner image en arrière-plan + gradient overlay
 *  - emblem rond à gauche (fallback monogramme)
 *  - gamertag + service tag + rang carrière au centre
 *  - barre XP composite en bas si rang carrière dispo
 *
 * Skill peaks (highest CSR / LUSR) à droite si l'identity les porte —
 * sinon le bandeau s'auto-collapse en 1 colonne.
 *
 * Cas dégradé (identity null) : affiche un placeholder textuel avec le
 * gamertag uniquement + message "Identité Spartan indisponible".
 */
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { HomeSkillPeakCard, resolveSkillPeakState } from '@/features/home/HomeSkillPeakCard'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSpartanIdentity } from '@/lib/api/types'

interface ExplorerTargetIdentityBannerProps {
  identity: HomeSpartanIdentity | null
  /** Gamertag fallback affiché quand identity == null ou sans spartan_id. */
  gamertag: string
  /** Label localisé pour l'état "identité indisponible" (manifest). */
  identityUnavailableLabel: string
  /** Description localisée pour l'état "identité indisponible" (manifest). */
  identityUnavailableDescription: string
}

export function ExplorerTargetIdentityBanner({
  identity,
  gamertag,
  identityUnavailableLabel,
  identityUnavailableDescription,
}: ExplorerTargetIdentityBannerProps) {
  const locale = useAppShellStore((s) => s.locale)
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'

  const monogram = gamertag.trim().slice(0, 1).toUpperCase() || 'S'
  const bannerUrl = identity?.banner_image_url ?? null
  const emblemUrl = identity?.emblem_image_url ?? null
  const spartanID = identity?.spartan_id ?? null
  const careerRank = identity?.career_rank ?? null
  const highestCSR = identity?.highest_csr ?? null
  const highestLUSR = identity?.highest_lusr ?? null
  const hasSkillPeaks = highestCSR != null || highestLUSR != null
  const csrState = highestCSR != null ? resolveSkillPeakState(highestCSR, true, 'ranked') : null
  const lusrState = highestLUSR != null ? resolveSkillPeakState(highestLUSR, true, 'unranked') : null

  const formatXP = (n: number) => n.toLocaleString(numberLocale)

  // Cas dégradé : aucune identité.
  if (identity == null) {
    return (
      <div className="overflow-hidden rounded-2xl border border-dashed border-border bg-muted/30 px-5 py-6">
        <div className="flex items-center gap-4">
          <div
            className="flex h-16 w-16 items-center justify-center rounded-full border border-border bg-background text-2xl font-bold text-muted-foreground"
            aria-hidden="true"
          >
            {monogram}
          </div>
          <div className="flex flex-col">
            <h2 className="text-2xl font-bold text-foreground">{gamertag}</h2>
            <p className="text-sm font-semibold text-muted-foreground">
              {identityUnavailableLabel}
            </p>
            <p className="text-xs text-muted-foreground">
              {identityUnavailableDescription}
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div
      className={`grid gap-3 ${hasSkillPeaks ? 'lg:grid-cols-[minmax(0,1fr)_minmax(16rem,18rem)]' : ''} lg:items-stretch`}
    >
      <div className="relative overflow-hidden rounded-2xl border border-border bg-card shadow-sm">
        {bannerUrl && (
          <img
            data-testid="explorer-target-banner-image"
            src={bannerUrl}
            alt=""
            aria-hidden="true"
            className="absolute inset-0 h-full w-full object-cover"
            loading="lazy"
            decoding="async"
          />
        )}
        {bannerUrl && (
          <div
            className="pointer-events-none absolute inset-0 bg-gradient-to-b from-background/30 via-background/10 to-background/50"
            aria-hidden="true"
          />
        )}

        <div className="relative flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center">
          {/* Emblem */}
          <div className="flex-shrink-0">
            {emblemUrl ? (
              <img
                data-testid="explorer-target-emblem"
                src={emblemUrl}
                alt=""
                aria-hidden="true"
                className="h-16 w-16 rounded-full border-2 border-border bg-background object-cover sm:h-20 sm:w-20"
                loading="lazy"
                decoding="async"
              />
            ) : (
              <div
                className="flex h-16 w-16 items-center justify-center rounded-full border-2 border-border bg-background text-2xl font-bold sm:h-20 sm:w-20 sm:text-3xl"
                aria-hidden="true"
              >
                {monogram}
              </div>
            )}
          </div>

          {/* Identité */}
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <h2 className="truncate text-2xl font-bold text-foreground sm:text-3xl">
              {gamertag}
            </h2>
            {spartanID && (
              <p className="font-mono text-sm italic text-muted-foreground sm:text-base">
                {spartanID}
              </p>
            )}
            {careerRank?.rank_title && (
              <p className="text-sm font-semibold text-foreground sm:text-base">
                {careerRank.rank_title}
              </p>
            )}
          </div>
        </div>

        {/* Barre XP rang carrière */}
        {careerRank?.current_xp != null && careerRank?.xp_for_next_rank != null && careerRank.xp_for_next_rank > 0 && (
          <div className="relative px-5 pb-4">
            <CompositeProgressBar value={(careerRank.current_xp / careerRank.xp_for_next_rank) * 100} />
            <div className="mt-1 flex justify-between text-xs text-muted-foreground">
              <span>{formatXP(careerRank.current_xp)}</span>
              <span>{formatXP(careerRank.xp_for_next_rank)}</span>
            </div>
          </div>
        )}
      </div>

      {/* Panneau skill peaks à droite (optionnel) */}
      {hasSkillPeaks && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 lg:grid-rows-2">
          {csrState && (
            <HomeSkillPeakCard
              label="Highest CSR"
              peak={highestCSR}
              numberLocale={numberLocale}
              testIdPrefix="explorer-target-csr"
              state={csrState.state}
              detail={csrState.detail}
            />
          )}
          {lusrState && (
            <HomeSkillPeakCard
              label="Highest LUSR"
              peak={highestLUSR}
              numberLocale={numberLocale}
              testIdPrefix="explorer-target-lusr"
              state={lusrState.state}
              detail={lusrState.detail}
            />
          )}
        </div>
      )}
    </div>
  )
}
