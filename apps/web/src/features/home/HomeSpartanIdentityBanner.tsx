/**
 * HomeSpartanIdentityBanner — bannière hero (banner image + emblem + gamertag
 * + Spartan ID + rang carrière) avec panneau skill peaks adjacent.
 *
 * P8.4 finition (revue 2026-04-29) : extrait de HomePage.tsx (~160L).
 */
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSkillPeakSummary, HomeSpartanIdentity } from '@/lib/api/types'
import { getSpartanIdentityText } from './spartanIdentity.i18n'
import { HomeSkillPeakCard, resolveSkillPeakState } from './HomeSkillPeakCard'

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
}: HomeSpartanIdentityBannerProps) {
  const locale = useAppShellStore((s) => s.locale)
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const spartanText = getSpartanIdentityText(locale)
  const labels = spartanText.labels

  const careerRank = spartanIdentity.career_rank ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const identityMonogram = playerName.trim().slice(0, 1).toUpperCase() || 'S'

  const hasAnySkillHistory = hasRankedHistory || hasUnrankedHistory
  const csrState = resolveSkillPeakState(highestCSR, hasRankedHistory, 'ranked')
  const lusrState = resolveSkillPeakState(highestLUSR, hasUnrankedHistory, 'unranked')

  const emptySkillPanelTitle = hasPrivacyWarning
    ? spartanText.emptyPanel.titleUnavailable
    : spartanText.emptyPanel.titleNone
  const emptySkillPanelDescription = hasPrivacyWarning
    ? spartanText.emptyPanel.descriptionUnavailable
    : spartanText.emptyPanel.descriptionNone

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,19.5rem)] lg:items-stretch">
      <div className="overflow-hidden rounded-2xl border border-border bg-muted/60 shadow-sm">
        <div
          data-testid="home-spartan-identity-banner"
          className="relative overflow-hidden bg-slate-950" // color-allow: thématique Spartan UI (banner hero distinctif Halo)
        >
          {spartanIdentity.banner_image_url && (
            <img
              data-testid="home-spartan-banner-surface"
              src={spartanIdentity.banner_image_url}
              alt=""
              aria-hidden="true"
              className="absolute inset-0 h-full w-full object-cover"
              loading="lazy"
              decoding="async"
            />
          )}
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
            className="relative flex flex-col gap-6 pt-1 pb-5 pl-5 pr-28 text-white sm:pl-6 sm:pr-32 lg:min-h-[9rem] lg:flex-row lg:items-start lg:justify-between"
            style={{ textShadow: '0 1px 6px rgba(0,0,0,0.85)' }}
          >
            <div className="flex min-w-0 items-center gap-4 lg:self-center">
              <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-cyan-300/60 bg-slate-950/60 shadow-[0_0_0_4px_rgba(8,15,28,0.35)] sm:h-24 sm:w-24"> {/* color-allow: thématique Spartan UI (emblem holder) */}
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
                    className="text-3xl font-semibold tracking-[0.18em] text-cyan-100" // color-allow: thématique Spartan UI
                  >
                    {identityMonogram}
                  </span>
                )}
              </div>

              <div className="min-w-0">
                <p
                  data-testid="home-spartan-gamertag"
                  className="truncate text-3xl font-semibold text-white sm:text-4xl"
                >
                  {playerName}
                </p>
                {spartanIdentity.spartan_id ? (
                  <p
                    data-testid="home-spartan-id-value"
                    className="mt-2 text-2xl font-medium italic tracking-[0.34em] text-cyan-50 sm:text-3xl" // color-allow: thématique Spartan UI (Spartan ID)
                  >
                    {spartanIdentity.spartan_id}
                  </p>
                ) : (
                  <p
                    className="mt-2 text-sm text-cyan-100/70" // color-allow: thématique Spartan UI
                  >
                    {identityUnavailableLabel}
                  </p>
                )}
              </div>
            </div>

            {careerRank && (
              <div className="flex items-center gap-4 self-start">
                <div className="min-w-0 rounded-xl bg-slate-950/15 px-3 py-2 text-right backdrop-blur-sm lg:max-w-[16rem]"> {/* color-allow: thématique Spartan UI (career rank panel) */}
                  <p data-testid="home-career-rank-title" className="text-lg font-semibold text-white sm:text-xl">
                    {careerRank.rank_title}
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>

        {careerRank && (
          <div className="border-t border-border/70 bg-background/80 px-5 py-4 sm:px-6">
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
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
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
                  className="shrink-0 whitespace-nowrap text-[11px] font-medium text-foreground/85 sm:text-xs"
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
        {!highestCSR && !highestLUSR && !hasAnySkillHistory ? (
          <div
            data-testid="home-skill-peaks-empty"
            className="rounded-2xl border border-dashed border-white/10 bg-slate-950/22 px-4 py-4 text-white shadow-[0_12px_30px_rgba(8,15,28,0.2)]" // color-allow: thématique Spartan UI (empty skill peaks panel)
          >
            <p className="text-sm font-semibold">{emptySkillPanelTitle}</p>
            <p className="mt-2 text-sm text-cyan-100/72">{emptySkillPanelDescription}</p> {/* color-allow: thématique Spartan UI */}
          </div>
        ) : (
          <>
            <HomeSkillPeakCard
              label={labels.highestCsr}
              peak={highestCSR}
              numberLocale={numberLocale}
              testIdPrefix="home-highest-csr"
              state={csrState.state}
              detail={csrState.detail}
            />
            <HomeSkillPeakCard
              label={labels.highestLusr}
              peak={highestLUSR}
              numberLocale={numberLocale}
              testIdPrefix="home-highest-lusr"
              state={lusrState.state}
              detail={lusrState.detail}
            />
          </>
        )}
      </div>
    </div>
  )
}
