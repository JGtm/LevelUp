/**
 * HomeSpartanIdentityBanner — bannière hero (banner image + emblem + gamertag
 * + Spartan ID + rang carrière) avec panneau skill peaks adjacent.
 *
 * P8.4 finition (revue 2026-04-29) : extrait de HomePage.tsx (~160L).
 */
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useCapability } from '@/lib/capabilities/capabilities'
import type { HomeSkillPeakSummary, HomeSpartanIdentity } from '@/lib/api/types'
import { getSpartanIdentityText } from './spartanIdentity.i18n'
import { HomeSkillPeakCard, resolveSkillPeakState } from './HomeSkillPeakCard'
import { RecoloredMask } from '@/features/spartan-customizer/RecoloredMask'
import { useSpartanAppearance } from '@/features/spartan-customizer/store'

interface HomeSpartanIdentityBannerProps {
  spartanIdentity: HomeSpartanIdentity
  playerName: string
  highestCSR: HomeSkillPeakSummary | null
  highestLUSR: HomeSkillPeakSummary | null
  hasRankedHistory: boolean
  hasUnrankedHistory: boolean
  hasPrivacyWarning: boolean
  /** Texte fourni par le manifest home.identity.unavailable. */
  identityUnavailableLabel: string
  /** Halo 5 only : ouvre la modale de personnalisation Spartan. Absent ⇒ bannière non cliquable. */
  onSpartanClick?: () => void
  /** aria-label de la bannière cliquable (manifest home.spartan_customizer.open_aria). */
  spartanCustomizeLabel?: string
}

export function HomeSpartanIdentityBanner({
  spartanIdentity,
  playerName,
  highestCSR,
  highestLUSR,
  hasRankedHistory,
  hasUnrankedHistory,
  hasPrivacyWarning,
  identityUnavailableLabel,
  onSpartanClick,
  spartanCustomizeLabel,
}: HomeSpartanIdentityBannerProps) {
  const locale = useAppShellStore((s) => s.locale)
  const numberLocale = intlLocale(locale)
  const spartanText = getSpartanIdentityText(locale)

  // Bannière SYNTHÉTISÉE (emblème + nameplate recolorisés) pour les titres déclarant la
  // capability `spartan_customizer` : leur banner_image_url n'est pas une vraie bannière
  // (Halo 5 = render full-body du Spartan). Gating par CAPABILITY, pas par slug.
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const synthesizeBanner = useCapability('spartan_customizer')
  // Apparence Spartan (emblème + couleurs) choisie-ou-défaut (#160), PAR TITRE → sert à
  // composer le bandeau. Jamais vide.
  const spartanApp = useSpartanAppearance(currentTitleSlug)
  const spartanColors = {
    primary: spartanApp.primary,
    secondary: spartanApp.secondary,
    tertiary: spartanApp.tertiary,
  }
  const spartanEmblemId = spartanApp.emblemId ?? '160'
  const spartanNameplateId = spartanApp.nameplateId ?? '160'
  const activeBannerUrl = synthesizeBanner ? null : (spartanIdentity.banner_image_url ?? null)
  const labels = spartanText.labels

  const careerRank = spartanIdentity.career_rank ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const identityMonogram = playerName.trim().slice(0, 1).toUpperCase() || 'S'

  const hasAnySkillHistory = hasRankedHistory || hasUnrankedHistory
  const csrState = resolveSkillPeakState(highestCSR, hasRankedHistory, 'ranked')
  const lusrState = resolveSkillPeakState(highestLUSR, hasUnrankedHistory, 'unranked')

  // Gating multi-titre (Phase 5) : carte « Meilleur CSR » ⇒ `ranked`, carte
  // « Meilleur LUSR » ⇒ `lusr`. NO-OP pour halo_infinite (déclare les deux).
  const hasRankedCap = useCapability('ranked')
  const hasLusrCap = useCapability('lusr')

  const emptySkillPanelTitle = hasPrivacyWarning
    ? spartanText.emptyPanel.titleUnavailable
    : spartanText.emptyPanel.titleNone
  const emptySkillPanelDescription = hasPrivacyWarning
    ? spartanText.emptyPanel.descriptionUnavailable
    : spartanText.emptyPanel.descriptionNone

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,19.5rem)] lg:items-stretch">
      <div
        data-testid="home-spartan-banner-surface"
        className={[
          'overflow-hidden rounded-lg border border-border bg-muted/60 bg-cover bg-center shadow-sm',
          onSpartanClick
            ? 'cursor-pointer transition-shadow hover:shadow-md focus:outline-none focus-visible:ring-2 focus-visible:ring-ring'
            : '',
        ].join(' ')}
        style={activeBannerUrl ? { backgroundImage: `url('${activeBannerUrl}')` } : undefined}
        role={onSpartanClick ? 'button' : undefined}
        tabIndex={onSpartanClick ? 0 : undefined}
        aria-label={onSpartanClick ? spartanCustomizeLabel : undefined}
        onClick={onSpartanClick}
        onKeyDown={
          onSpartanClick
            ? (e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onSpartanClick()
                }
              }
            : undefined
        }
      >
        <div
          data-testid="home-spartan-identity-banner"
          className="relative overflow-hidden"
        >
          {activeBannerUrl && (
            <div
              className="pointer-events-none absolute inset-0 bg-gradient-to-b from-background/30 via-background/10 to-background/40"
              aria-hidden="true"
            />
          )}
          {synthesizeBanner ? (
            // Halo 5 : le nameplate recolorisé (apparence choisie-ou-défaut) sert de fond,
            // + un scrim sémantique à gauche pour la lisibilité du gamertag/rang par-dessus.
            <>
              <RecoloredMask
                src={`/titles/${currentTitleSlug}/spartan/nameplates/${spartanNameplateId}.png`}
                colors={spartanColors}
                alt=""
                className="pointer-events-none absolute inset-0 h-full w-full object-cover"
              />
              <div
                data-testid="home-spartan-nameplate-scrim"
                className="pointer-events-none absolute inset-0 bg-gradient-to-r from-background/85 via-background/40 to-background/15"
                aria-hidden="true"
              />
            </>
          ) : !activeBannerUrl ? (
            // Backdrop synthétisé (titre non-H5 sans image de bannière) : dégradé sémantique.
            <div
              data-testid="home-spartan-synthesized-backdrop"
              className="pointer-events-none absolute inset-0 bg-gradient-to-br from-primary/25 via-muted/45 to-background"
              aria-hidden="true"
            />
          ) : null}
          {careerAdornmentUrl && (
            <div className="pointer-events-none absolute right-2 top-0 z-[1] flex h-full items-start">
              <img
                data-testid="home-spartan-adornment-image"
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
            data-testid="home-spartan-banner-shell"
            className="text-shadow-adaptive relative flex flex-col gap-6 pt-1 pb-5 pl-5 pr-28 text-foreground sm:pl-6 sm:pr-32 lg:min-h-[9rem] lg:flex-row lg:items-start lg:justify-between"
          >
            <div className="flex min-w-0 items-center gap-4 lg:self-center">
              {synthesizeBanner ? (
                // Halo 5 : emblème carré recolorisé (forme d'origine, PAS rond), collé
                // visuellement au nameplate de fond.
                <RecoloredMask
                  src={`/titles/${currentTitleSlug}/spartan/emblems/${spartanEmblemId}.png`}
                  colors={spartanColors}
                  alt={`Emblème ${playerName}`}
                  className="h-20 w-20 shrink-0 object-contain drop-shadow-[0_2px_6px_rgba(8,15,28,0.5)] sm:h-24 sm:w-24"
                />
              ) : (
                <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-primary/60 bg-card/80 shadow-[0_0_0_4px_rgba(8,15,28,0.35)] sm:h-24 sm:w-24">
                  {spartanIdentity.emblem_image_url ? (
                    <img
                      data-testid="home-spartan-emblem-image"
                      src={spartanIdentity.emblem_image_url}
                      alt={`Emblème ${playerName}`}
                      className="h-full w-full object-cover"
                      loading="lazy"
                      decoding="async"
                    />
                  ) : (
                    <span
                      className="text-3xl font-semibold tracking-label-md text-primary-foreground"
                    >
                      {identityMonogram}
                    </span>
                  )}
                </div>
              )}

              <div className="min-w-0">
                <p
                  data-testid="home-spartan-gamertag"
                  className="truncate text-3xl font-semibold text-foreground sm:text-4xl"
                >
                  {playerName}
                </p>
                {spartanIdentity.spartan_id ? (
                  <p
                    data-testid="home-spartan-id-value"
                    className="mt-2 text-2xl font-medium italic tracking-label-3xl text-foreground sm:text-3xl"
                  >
                    {spartanIdentity.spartan_id}
                  </p>
                ) : (
                  <p
                    className="mt-2 text-sm text-muted-foreground"
                  >
                    {identityUnavailableLabel}
                  </p>
                )}
              </div>
            </div>

            {careerRank && (
              <div className="flex items-center gap-4 self-start">
                <div className="min-w-0 rounded-xl bg-background/40 px-3 py-2 text-right backdrop-blur-sm lg:max-w-[16rem]">
                  <p data-testid="home-career-rank-title" className="text-lg font-semibold text-foreground sm:text-xl">
                    {careerRank.rank_title}
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>

        {careerRank && (
          <div className="border-t border-border/70 bg-card px-5 py-4 sm:px-6">
            <div className="space-y-3">
              <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                <span>
                  {careerRank.is_max_rank
                    ? labels.maxRank
                    : labels.progressTowardsRank(careerRank.next_rank_title ?? '')}
                </span>
                <span>
                  {careerRank.is_max_rank
                    ? labels.maxRank
                    : `${careerRank.progress_pct.toLocaleString(numberLocale, { maximumFractionDigits: 0 })} %`}
                </span>
              </div>
              <div className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3">
                <span
                  data-testid="home-career-rank-progress-current"
                  className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 sm:text-xs"
                >
                  {`${careerRank.current_xp.toLocaleString(numberLocale)} XP`}
                </span>
                <div className="min-w-0">
                  <CompositeProgressBar
                    value={careerRank.progress_pct}
                    fillTestId="home-career-rank-progress-fill"
                  />
                </div>
                <span
                  data-testid="home-career-rank-progress-target"
                  className="shrink-0 whitespace-nowrap text-3xs font-medium text-foreground/85 sm:text-xs"
                >
                  {careerRank.is_max_rank
                    ? labels.maxRank
                    : `${careerRank.xp_for_next_rank.toLocaleString(numberLocale)} XP`}
                </span>
              </div>
            </div>
          </div>
        )}
      </div>

      <div
        data-testid="home-skill-peaks-panel"
        className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1 lg:auto-rows-fr"
      >
        {!hasRankedCap && !hasLusrCap ? null : !highestCSR &&
          !highestLUSR &&
          !hasAnySkillHistory ? (
          <div
            data-testid="home-skill-peaks-empty"
            className="rounded-lg border border-dashed border-border bg-muted/40 px-4 py-4 text-foreground shadow-[0_12px_30px_rgba(8,15,28,0.2)]"
          >
            <p className="text-sm font-semibold">{emptySkillPanelTitle}</p>
            <p className="mt-2 text-sm text-muted-foreground">{emptySkillPanelDescription}</p>
          </div>
        ) : (
          <>
            {hasRankedCap && (
              <HomeSkillPeakCard
                label={labels.highestCsr}
                peak={highestCSR}
                numberLocale={numberLocale}
                locale={locale}
                testIdPrefix="home-highest-csr"
                state={csrState.state}
                detail={csrState.detail}
              />
            )}
            {hasLusrCap && (
              <HomeSkillPeakCard
                label={labels.highestLusr}
                peak={highestLUSR}
                numberLocale={numberLocale}
                locale={locale}
                testIdPrefix="home-highest-lusr"
                state={lusrState.state}
                detail={lusrState.detail}
              />
            )}
          </>
        )}
      </div>
    </div>
  )
}
