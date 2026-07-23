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
import { HomeSkillPeakCard } from '@/features/home/HomeSkillPeakCard'
import { resolveSkillPeakState } from '@/features/home/skillPeakState'
import { getSpartanIdentityText } from '@/features/home/spartanIdentity.i18n'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { HomeSpartanIdentity } from '@/lib/api/types'
import { RecoloredMask } from '@/features/spartan-customizer/RecoloredMask'
import { useSpartanAppearance } from '@/features/spartan-customizer/store'

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
  const numberLocale = intlLocale(locale)
  // Labels localisés réutilisés du Home (source unique : home.spartan.*) :
  // « Meilleur CSR/LUSR », « Rang max », « Progression vers … ».
  const { labels } = getSpartanIdentityText(locale)
  // Halo 5 : bandeau composé (emblème carré + nameplate recolorisés, apparence
  // choisie-ou-défaut #160) — parité avec HomeSpartanIdentityBanner.
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Gating par CAPABILITY (pas par slug) — parité HomeSpartanIdentityBanner.
  const synthesizeBanner = useCapability('spartan_customizer')
  const spartanApp = useSpartanAppearance(currentTitleSlug)
  const spartanColors = {
    primary: spartanApp.primary,
    secondary: spartanApp.secondary,
    tertiary: spartanApp.tertiary,
  }
  const spartanEmblemId = spartanApp.emblemId ?? '160'
  const spartanNameplateId = spartanApp.nameplateId ?? '160'

  const monogram = gamertag.trim().slice(0, 1).toUpperCase() || 'S'
  const bannerUrl = identity?.banner_image_url ?? null
  const emblemUrl = identity?.emblem_image_url ?? null
  const spartanID = identity?.spartan_id ?? null
  const careerRank = identity?.career_rank ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const highestCSR = identity?.highest_csr ?? null
  const highestLUSR = identity?.highest_lusr ?? null
  const hasSkillPeaks = highestCSR != null || highestLUSR != null
  const csrState = highestCSR != null ? resolveSkillPeakState(highestCSR, true, 'ranked') : null
  const lusrState = highestLUSR != null ? resolveSkillPeakState(highestLUSR, true, 'unranked') : null

  const formatXP = (n: number) => n.toLocaleString(numberLocale)

  // Cas dégradé : aucune identité.
  if (identity == null) {
    // Halo 5 : même sans identité (nouveau joueur), on affiche le bandeau par défaut
    // (emblème + nameplate recolorisés #160) plutôt qu'un placeholder vide.
    if (synthesizeBanner) {
      return (
        <div className="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
          <div className="relative overflow-hidden">
            <RecoloredMask
              src={`/titles/${currentTitleSlug}/spartan/nameplates/${spartanNameplateId}.png`}
              colors={spartanColors}
              alt=""
              className="pointer-events-none absolute inset-0 h-full w-full object-cover"
            />
            <div
              className="pointer-events-none absolute inset-0 bg-gradient-to-r from-background/85 via-background/40 to-background/15"
              aria-hidden="true"
            />
            <div className="relative z-[2] flex min-h-[7rem] items-center gap-4 px-5 py-5">
              <RecoloredMask
                src={`/titles/${currentTitleSlug}/spartan/emblems/${spartanEmblemId}.png`}
                colors={spartanColors}
                alt=""
                className="h-16 w-16 shrink-0 object-contain drop-shadow-[0_2px_6px_rgba(8,15,28,0.5)] sm:h-20 sm:w-20"
              />
              <div className="flex min-w-0 flex-col">
                <h2 className="truncate text-2xl font-bold text-foreground text-shadow-adaptive sm:text-3xl">
                  {gamertag}
                </h2>
                <p className="text-sm font-semibold text-muted-foreground">
                  {identityUnavailableLabel}
                </p>
              </div>
            </div>
          </div>
        </div>
      )
    }
    return (
      <div className="overflow-hidden rounded-lg border border-dashed border-border bg-muted/30 px-5 py-6">
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
      <div
        data-testid="explorer-target-banner-image"
        className="overflow-hidden rounded-lg border border-border bg-card bg-cover bg-center shadow-sm"
        style={bannerUrl ? { backgroundImage: `url('${bannerUrl}')` } : undefined}
      >
        {/* Wrapper hero interne (parité HomeSpartanIdentityBanner) : fournit un
            contexte de clip RECTANGULAIRE pour les enfants absolus (gradient +
            adornment), tandis que le background-image vit sur l'outer div ARRONDI.
            Sans ce wrapper, les couches absolues étaient clippées par les coins
            arrondis de l'outer → liseré/trou à gauche sur certaines nameplates
            (régression récurrente du fork dégradé du banner Home). */}
        <div className="relative overflow-hidden">
          {synthesizeBanner && (
            <>
              <RecoloredMask
                src={`/titles/${currentTitleSlug}/spartan/nameplates/${spartanNameplateId}.png`}
                colors={spartanColors}
                alt=""
                className="pointer-events-none absolute inset-0 h-full w-full object-cover"
              />
              <div
                className="pointer-events-none absolute inset-0 bg-gradient-to-r from-background/85 via-background/40 to-background/15"
                aria-hidden="true"
              />
            </>
          )}
          {bannerUrl && (
            <div
              className="pointer-events-none absolute inset-0 bg-gradient-to-b from-background/30 via-background/10 to-background/50"
              aria-hidden="true"
            />
          )}

          {/* Adornment carrière (parité Home) : couche décorative à droite,
              derrière l'identité (z-[1] vs z-[2]), bornée à la zone hero (sibling du
              shell dans le wrapper interne) pour ne pas chevaucher la barre XP. Le
              drop-shadow rgba(...) est une couleur structurelle (tolérée, règle 20
              CLAUDE.md) — recopié du banner Home. */}
          {careerAdornmentUrl && (
            <div className="pointer-events-none absolute right-2 top-0 z-[1] flex h-full items-start">
              <img
                data-testid="explorer-target-adornment-image"
                src={careerAdornmentUrl}
                alt=""
                aria-hidden="true"
                className="h-full w-auto object-contain object-top drop-shadow-[0_14px_20px_rgba(8,15,28,0.48)]"
                loading="lazy"
                decoding="async"
              />
            </div>
          )}

          <div
            className={`relative z-[2] flex flex-col gap-4 px-5 py-5 sm:flex-row sm:items-center ${
              careerAdornmentUrl ? 'pr-24 sm:pr-28' : ''
            } lg:min-h-[9rem]`}
          >
            {/* Emblem */}
            <div className="relative z-[2] flex-shrink-0">
              {synthesizeBanner ? (
                <RecoloredMask
                  src={`/titles/${currentTitleSlug}/spartan/emblems/${spartanEmblemId}.png`}
                  colors={spartanColors}
                  alt=""
                  className="h-16 w-16 object-contain drop-shadow-[0_2px_6px_rgba(8,15,28,0.5)] sm:h-20 sm:w-20"
                />
              ) : emblemUrl ? (
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
            <div className="relative z-[2] flex min-w-0 flex-1 flex-col gap-1">
              <h2 className="truncate text-2xl font-bold text-foreground sm:text-3xl">
                {gamertag}
              </h2>
              {spartanID && (
                <p className="font-mono text-sm italic text-muted-foreground sm:text-base">
                  {spartanID}
                </p>
              )}
            </div>

            {/* Grade carrière dans un bloc semi-transparent (parité Home). */}
            {careerRank?.rank_title && (
              <div className="relative z-[2] shrink-0 self-start rounded-xl bg-background/40 px-3 py-2 text-right backdrop-blur-sm">
                <p className="text-sm font-semibold text-foreground sm:text-base">
                  {careerRank.rank_title}
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Progression rang carrière — calquée sur le Home (HomeSpartanIdentityBanner) :
            visible dès que careerRank existe ; current_xp + barre (progress_pct, calculé
            backend) + borne cible. Au rang max (Héros), « Rang max » remplace la cible et
            progress_pct=100 → barre verte pleine (success token). Plus de branche qui
            masque tout (régression du fix Héros précédent). */}
        {careerRank && (
          <div className="relative border-t border-border/70 bg-card px-5 py-4 sm:px-6">
            <div className="space-y-3">
              {/* Ligne du haut (progression vers le rang suivant + %) : masquée au
                  rang max — pas de "suivant", et on ne garde qu'UN "Rang max", celui
                  en bout de barre composite (cf. retour user). */}
              {!careerRank.is_max_rank && (
                <div className="flex items-center justify-between gap-3 text-2xs text-muted-foreground sm:text-xs">
                  <span>{labels.progressTowardsRank(careerRank.next_rank_title ?? '')}</span>
                  <span>
                    {`${careerRank.progress_pct.toLocaleString(numberLocale, { maximumFractionDigits: 0 })} %`}
                  </span>
                </div>
              )}
              <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
                {/* Gauche : progression XP dans le rang courant. Au rang max, cette
                    progression intra-rang vaut 0 → on affiche à la place l'XP de
                    carrière CUMULÉE (total_xp, « le grand nombre qui fait rêver »). */}
                <span className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 sm:text-xs">
                  {careerRank.is_max_rank
                    ? `${formatXP(careerRank.total_xp ?? 0)} XP`
                    : `${formatXP(careerRank.current_xp)} XP`}
                </span>
                <div className="min-w-0">
                  <CompositeProgressBar
                    value={careerRank.progress_pct}
                    fillTestId="explorer-target-rank-progress-fill"
                  />
                </div>
                <span className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 sm:text-xs">
                  {careerRank.is_max_rank ? labels.maxRank : `${formatXP(careerRank.xp_for_next_rank)} XP`}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Panneau skill peaks à droite (optionnel) */}
      {hasSkillPeaks && (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 lg:grid-rows-2">
          {csrState && (
            <HomeSkillPeakCard
              label={labels.highestCsr}
              peak={highestCSR}
              numberLocale={numberLocale}
              locale={locale}
              testIdPrefix="explorer-target-csr"
              state={csrState.state}
              detail={csrState.detail}
              reachedOnLabel={labels.peakReachedOn}
            />
          )}
          {lusrState && (
            <HomeSkillPeakCard
              label={labels.highestLusr}
              peak={highestLUSR}
              numberLocale={numberLocale}
              locale={locale}
              testIdPrefix="explorer-target-lusr"
              state={lusrState.state}
              detail={lusrState.detail}
              reachedOnLabel={labels.peakReachedOn}
            />
          )}
        </div>
      )}
    </div>
  )
}
