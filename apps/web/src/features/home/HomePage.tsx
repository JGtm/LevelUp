/**
 * HomePage — Accueil Mission Control (Slice 5).
 *
 * P8.4 (revue 2026-04-29) : sub-components extraits dans des fichiers dédiés
 * (HomeKPICard, HomeOutcomeBar, HomeHighlightTile, HomeSessionCarousel,
 * HomeSkillPeakCard) — réduit ce fichier de ~440L.
 */
import { useState } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { MatchCard } from '@/components/ui/match-card'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { HomeBattlePassPanel } from './HomeBattlePassPanel'
import { HomeHeroBanner } from './HomeHeroBanner'
import { HomeChallengesList } from './HomeChallengesList'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'
import { RecentMediaRail } from './RecentMediaRail'
import { HomeKPICard } from './HomeKPICard'
import { HomeOutcomeBar } from './HomeOutcomeBar'
import { HomeHighlightTile } from './HomeHighlightTile'
import { HomeSessionCarousel } from './HomeSessionCarousel'
import { HomeSkillPeakCard, resolveSkillPeakState } from './HomeSkillPeakCard'
import { ChallengesCarousel } from '@/features/prestige/components/ChallengesCarousel'
import { useHomePage, useSeasonPassPreview } from './queries'
import { useSetMatchFavorite } from '@/features/match-history/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import { kdScale, accuracyScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { OutcomeSequenceTape } from '@/components/charts/OutcomeSequenceTape'
import { getHighlightText } from './highlights.i18n'
import { getKPIText } from './kpi.i18n'
import { getSpartanIdentityText } from './spartanIdentity.i18n'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { InfoTooltip } from '@/components/ui/info-tooltip'


export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const locale = useAppShellStore((s) => s.locale)
  const userTimezone = useAppShellStore((s) => s.userTimezone)
  const { data: fieldMappings } = useFieldMappings()
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const {
    data: seasonPass,
    isLoading: isSeasonPassLoading,
    error: seasonPassError,
  } = useSeasonPassPreview(playerSlug)
  const [matchTab, setMatchTab] = useState<'recent' | 'favorites'>('recent')
  const [soloIdx, setSoloIdx] = useState(0)
  const [squadIdx, setSquadIdx] = useState(0)
  const favoriteMutation = useSetMatchFavorite(playerSlug)

  function goToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  function goToSession(sessionLabel: string) {
    void navigate({
      to: '/players/$playerSlug/stats/sessions',
      params: { playerSlug },
      search: { session: sessionLabel },
    })
  }

  if (isLoading) return null

  if (isError) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title={t('home.empty.error_title')}
            description={t('home.empty.error_description')}
            actionLabel={t('home.errors.retry')}
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title={t('home.empty.no_data_title')}
            description={t('home.empty.no_data_description')}
            actionLabel={t('home.errors.reload')}
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const { hero } = data
  const highlights = data.highlights ?? []
  const recentMatches = data.recent_matches ?? []
  const favoriteMatches = data.favorite_matches ?? []
  const spartanIdentity = data.spartan_identity ?? null
  const highestCSR = spartanIdentity?.highest_csr ?? null
  const highestLUSR = spartanIdentity?.highest_lusr ?? null
  const hasRankedHistory = data.has_ranked_history ?? Boolean(highestCSR)
  const hasUnrankedHistory = data.has_unranked_history ?? Boolean(highestLUSR)
  const hasAnySkillHistory = hasRankedHistory || hasUnrankedHistory
  const csrState = resolveSkillPeakState(highestCSR, hasRankedHistory, 'ranked')
  const lusrState = resolveSkillPeakState(highestLUSR, hasUnrankedHistory, 'unranked')
  const careerRank = spartanIdentity?.career_rank ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const identityMonogram = hero.player_name.trim().slice(0, 1).toUpperCase() || 'S'
  const soloSession = data.solo_session ?? null
  const squadSession = data.squad_session ?? null
  const soloSessions = data.solo_sessions ?? (soloSession ? [soloSession] : [])
  const squadSessions = data.squad_sessions ?? (squadSession ? [squadSession] : [])
  const challenges = seasonPass?.challenges
  const challengesCompleted = challenges?.completed ?? 0
  const challengesTotal = challenges?.total ?? 0
  const challengesCompletedLabel = challenges?.available
    ? `${challengesCompleted} / ${challengesTotal} complétés`
    : null
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const kpiText = getKPIText(locale)
  // Phase D multi-titres : résout les libellés métier via le backend TOML.
  // Fallback gracieux sur les libellés locaux de kpi.i18n.ts si l'endpoint
  // est absent (flag MULTI_TITLE_API_ENABLED off ou 404).
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  const spartanText = getSpartanIdentityText(locale)
  const labels = spartanText.labels
  const hasPrivacyWarning = !!data.privacy_warning?.level && data.privacy_warning.level !== 'none'
  const emptySkillPanelTitle = hasPrivacyWarning
    ? spartanText.emptyPanel.titleUnavailable
    : spartanText.emptyPanel.titleNone
  const emptySkillPanelDescription = hasPrivacyWarning
    ? spartanText.emptyPanel.descriptionUnavailable
    : spartanText.emptyPanel.descriptionNone

  return (
    <div className="relative isolate flex flex-col">
      <div className="sticky top-0 z-0 px-6 pt-0" data-testid="home-hero-banner-sticky">
        <HomeHeroBanner />
      </div>

      <div className="relative z-10 space-y-6 bg-background px-6 pb-6 pt-6">
        {/* B9 : signal discret si données partielles (compte privé) */}
        <PrivacyBanner warning={data.privacy_warning} />

        {/* Hero KPIs */}
        <div>
          <div className="space-y-4">
            {spartanIdentity && (
              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,19.5rem)] lg:items-stretch">
                <div className="overflow-hidden rounded-2xl border border-border bg-muted/60 shadow-sm">
                  <div
                    data-testid="home-spartan-identity-banner"
                    className="relative overflow-hidden bg-slate-950"
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
                        <div className="flex h-20 w-20 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-cyan-300/60 bg-slate-950/60 shadow-[0_0_0_4px_rgba(8,15,28,0.35)] sm:h-24 sm:w-24">
                          {spartanIdentity.emblem_image_url ? (
                            <img
                              data-testid="home-spartan-emblem-image"
                              src={spartanIdentity.emblem_image_url}
                              alt={`Emblème ${hero.player_name}`}
                              className="h-full w-full object-cover"
                              loading="lazy"
                              decoding="async"
                            />
                          ) : (
                            <span className="text-3xl font-semibold tracking-[0.18em] text-cyan-100">{identityMonogram}</span>
                          )}
                        </div>

                        <div className="min-w-0">
                          <p
                            data-testid="home-spartan-gamertag"
                            className="truncate text-3xl font-semibold text-white sm:text-4xl"
                          >
                            {hero.player_name}
                          </p>
                          {spartanIdentity.spartan_id ? (
                            <p
                              data-testid="home-spartan-id-value"
                              className="mt-2 text-2xl font-medium italic tracking-[0.34em] text-cyan-50 sm:text-3xl"
                            >
                              {spartanIdentity.spartan_id}
                            </p>
                          ) : (
                            <p className="mt-2 text-sm text-cyan-100/70">{t('home.identity.unavailable')}</p>
                          )}
                        </div>
                      </div>

                      {careerRank && (
                        <div className="flex items-center gap-4 self-start">
                          <div className="min-w-0 rounded-xl bg-slate-950/15 px-3 py-2 text-right backdrop-blur-sm lg:max-w-[16rem]">
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
                      className="rounded-2xl border border-dashed border-white/10 bg-slate-950/22 px-4 py-4 text-white shadow-[0_12px_30px_rgba(8,15,28,0.2)]"
                    >
                      <p className="text-sm font-semibold">{emptySkillPanelTitle}</p>
                      <p className="mt-2 text-sm text-cyan-100/72">{emptySkillPanelDescription}</p>
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
            )}

            <div className="kpi-stats-grid items-stretch">
              {/* 1 — Parties */}
              <HomeKPICard label={labelOf('total_matches_played')} value={hero.kpis.total_matches.toLocaleString(numberLocale)} compact />

              {/* 2 — KDA/FDA coloré comme les tuiles match */}
              {(() => {
                const kda = hero.kpis.avg_kda
                const kdaStyle = kda != null ? { color: tokenCssVar(kdScale(kda)) } : undefined
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('kda')}</p>
                    <p className="text-xl font-bold text-muted-foreground" style={kdaStyle}>{kda != null ? kda.toFixed(2) : '—'}</p>
                  </div>
                )
              })()}

              {/* 3 — Taux de victoire + barre composite outcomes */}
              {(() => {
                const wins = hero.kpis.wins
                const losses = hero.kpis.losses
                const draws = hero.kpis.draws ?? 0
                const dnfs = hero.kpis.dnfs ?? 0
                const neutral = draws + dnfs
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('win_rate')}</p>
                    <p className="text-xl font-bold text-primary">{`${(hero.kpis.win_rate * 100).toFixed(0)}%`}</p>
                    <div className="mt-2 w-full">
                      <HomeOutcomeBar wins={wins} draws={draws} losses={losses} dnfs={dnfs} />
                    </div>
                    <div className="mt-1.5 flex justify-center gap-3 text-xs font-semibold tabular-nums">
                      <span style={{ color: tokenCssVar('outcome-win') }}>{wins}</span>
                      {neutral > 0 && <span style={{ color: tokenCssVar('outcome-draw') }}>{neutral}</span>}
                      <span style={{ color: tokenCssVar('outcome-loss') }}>{losses}</span>
                    </div>
                  </div>
                )
              })()}

              {/* 4 — Durée totale */}
              {(() => {
                const secs = hero.kpis.total_playtime_secs ?? 0
                let formatted: string
                if (secs <= 0) {
                  formatted = '—'
                } else {
                  const totalMin = Math.floor(secs / 60)
                  const h = Math.floor(totalMin / 60)
                  const totalDays = Math.floor(h / 24)
                  if (totalDays >= 365) {
                    const years = Math.floor(totalDays / 365)
                    const remDays = totalDays % 365
                    const months = Math.floor(remDays / 30)
                    const days = remDays % 30
                    const parts = [`${years}${kpiText.units.year}`]
                    if (months > 0) parts.push(`${months}${kpiText.units.month}`)
                    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
                    formatted = parts.join(' ')
                  } else if (totalDays >= 30) {
                    const months = Math.floor(totalDays / 30)
                    const days = totalDays % 30
                    const parts = [`${months}${kpiText.units.month}`]
                    if (days > 0) parts.push(`${days}${kpiText.units.day}`)
                    formatted = parts.join(' ')
                  } else if (totalDays >= 1) {
                    const remH = h % 24
                    formatted = remH > 0 ? `${totalDays}${kpiText.units.day} ${remH}${kpiText.units.hour}` : `${totalDays}${kpiText.units.day}`
                  } else {
                    const m = totalMin % 60
                    formatted = h === 0 ? `${m}${kpiText.units.minute}` : `${h}${kpiText.units.hour}${m > 0 ? String(m).padStart(2, '0') : ''}`
                  }
                }
                return <HomeKPICard label={kpiText.labels.totalTime} value={formatted} />
              })()}

              {/* 5 — Playlist favorite */}
              <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                <p className="text-xs text-muted-foreground">{kpiText.labels.favoritePlaylist}</p>
                <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
                  {hero.kpis.favorite_playlist_name || '—'}
                </p>
                {hero.kpis.favorite_playlist_count > 0 && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {hero.kpis.favorite_playlist_count.toLocaleString(numberLocale)} {kpiText.matches(hero.kpis.favorite_playlist_count)}
                  </p>
                )}
              </div>

              {/* 7 — Rendement / Résistance (barre composite) */}
              {(() => {
                const offConv = hero.kpis.avg_offensive_conversion
                const defRes = hero.kpis.avg_defensive_resistance
                const hasData = offConv != null || defRes != null
                const off = offConv ?? 0
                const def = defRes ?? 0
                const total = off + def
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                    <p className="text-xs text-muted-foreground mb-1.5">{kpiText.labels.offDef}</p>
                    {hasData ? (
                      <div className="w-full">
                        <div className="h-2 w-full rounded-full overflow-hidden flex">
                          {off > 0 && <div className="h-full" style={{ width: total > 0 ? `${(off / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-pos') }} />}
                          {def > 0 && <div className="h-full" style={{ width: total > 0 ? `${(def / total) * 100}%` : '50%', backgroundColor: tokenCssVar('divergent-neutral') }} />}
                        </div>
                        <div className="flex justify-center gap-3 mt-2">
                          <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-pos') }}>{off.toFixed(2)}</span>
                          <span className="text-sm font-bold leading-none" style={{ color: tokenCssVar('divergent-neutral') }}>{def.toFixed(2)}</span>
                        </div>
                      </div>
                    ) : (
                      <p className="text-xl font-bold text-muted-foreground">—</p>
                    )}
                  </div>
                )
              })()}

              {/* 8 — Précision avec code couleur */}
              {(() => {
                const acc = hero.kpis.avg_accuracy
                const accStyle = acc != null ? { color: tokenCssVar(accuracyScale(acc)) } : undefined
                return (
                  <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-2 py-3 text-center">
                    <p className="text-xs text-muted-foreground">{labelOf('accuracy')}</p>
                    <p className="text-xl font-bold text-primary" style={accStyle}>{acc != null ? `${acc.toFixed(0)}%` : '—'}</p>
                  </div>
                )
              })()}

              {/* 9 — Arme favorite */}
              <div className="flex h-full flex-col items-center justify-center rounded-lg border border-border bg-muted px-4 py-3 text-center">
                <p className="text-xs text-muted-foreground">{kpiText.labels.favoriteWeapon}</p>
                <p className="w-full truncate text-sm font-bold text-primary leading-tight mt-1">
                  {hero.kpis.favorite_weapon_name || '—'}
                </p>
                {hero.kpis.favorite_weapon_kills > 0 && (
                  <p className="text-xs text-muted-foreground mt-0.5">
                    {hero.kpis.favorite_weapon_kills.toLocaleString(numberLocale)} {kpiText.kills(hero.kpis.favorite_weapon_kills)}
                  </p>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* Battle Pass + Défis */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <HomeBattlePassPanel
            loading={isSeasonPassLoading}
            data={seasonPass}
            errorHint={seasonPassError instanceof Error ? seasonPassError.message : null}
          />

          <Card data-testid="home-challenges-card" className="flex min-h-[14rem] self-start flex-col">
            <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
              <CardTitle className="text-base">{t('home.challenges.title')}</CardTitle>
              {challengesCompletedLabel && (
                <p data-testid="home-challenges-completed" className="shrink-0 text-sm text-foreground">
                  <strong className="text-primary">{challengesCompleted}</strong> / {challengesTotal}{' '}
                  {locale === 'en' ? 'completed' : 'complétés'}
                </p>
              )}
            </CardHeader>
            <CardContent className="flex flex-1 flex-col">
              {isSeasonPassLoading && !seasonPass ? (
                <div className="flex flex-1 items-center justify-center">
                  <p className="text-sm text-muted-foreground">{t('home.challenges.loading')}</p>
                </div>
              ) : challenges?.available ? (
                <div className="space-y-3">
                  {Array.isArray(challenges.items) && challenges.items.length > 0 ? (
                    <HomeChallengesList items={challenges.items} />
                  ) : (
                    <div className="flex min-h-[8.5rem] items-center justify-center">
                      <EmptyStateNotice
                        title={t('home.challenges.empty_title')}
                        description={t('home.challenges.empty_description')}
                        className="w-full max-w-sm"
                      />
                    </div>
                  )}
                  {challenges.xp_available != null && challenges.xp_available > 0 && (
                    <p className="text-xs text-muted-foreground">
                      {t('home.challenges.xp_available', {
                        xp: challenges.xp_available.toLocaleString(numberLocale),
                      })}
                    </p>
                  )}
                </div>
              ) : seasonPassError ? (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title={t('home.challenges.unavailable_title')}
                    description={t('home.challenges.unavailable_error')}
                    className="w-full max-w-sm"
                  />
                </div>
              ) : (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title={t('home.challenges.unavailable_title')}
                    description={t('home.challenges.unavailable_no_data')}
                    className="w-full max-w-sm"
                  />
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Playlists récentes + Rang | Sessions récentes */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(320px,0.65fr)_minmax(0,1.35fr)]">
          <HomeRecentPlaylistsCard
            recentPlaylistRanks={data.recent_playlist_ranks}
          />

          {/* Sessions récentes */}
          <Card>
            <CardHeader className="space-y-0 pb-3">
              <CardTitle className="text-base">{t('home.sessions.title')}</CardTitle>
            </CardHeader>
            <CardContent>
              {soloSessions.length > 0 || squadSessions.length > 0 ? (
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  {soloSessions.length > 0 && (
                    <HomeSessionCarousel
                      sessions={soloSessions}
                      idx={soloIdx}
                      onIdxChange={setSoloIdx}
                      variant="solo"
                      playerSlug={playerSlug}
                      onNavigate={goToSession}
                    />
                  )}
                  {squadSessions.length > 0 && (
                    <HomeSessionCarousel
                      sessions={squadSessions}
                      idx={squadIdx}
                      onIdxChange={setSquadIdx}
                      variant="squad"
                      playerSlug={playerSlug}
                      onNavigate={goToSession}
                    />
                  )}
                </div>
              ) : (
                <EmptyStateNotice
                  title={t('home.sessions.empty_title')}
                  description={t('home.sessions.empty_description')}
                />
              )}
            </CardContent>
          </Card>
        </div>

        {/* Carousel défis Prestige (au-dessus des Faits Marquants) */}
        <Card>
          <CardContent className="pt-4">
            <ChallengesCarousel
              userId={playerSlug}
              titleSlug="halo_infinite"
              playerSlug={playerSlug}
            />
          </CardContent>
        </Card>

        {/* Highlights */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-1.5 text-base">
              {getHighlightText(locale).section.title}
              <InfoTooltip content={<p>{getHighlightText(locale).section.tooltipIntro}</p>} />
            </CardTitle>
          </CardHeader>
          <CardContent>
            {highlights.length > 0 ? (
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:[grid-template-columns:repeat(20,minmax(0,1fr))]">
                {highlights.map((h, i) => (
                  <HomeHighlightTile key={i} h={h} locale={locale} />
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title={t('home.highlights.empty_title')}
                description={t('home.highlights.empty_description')}
              />
            )}
          </CardContent>
        </Card>

        {/* Séquence des outcomes — bande compacte avant les tuiles */}
        {recentMatches.length > 0 && (
          <OutcomeSequenceTape
            matches={[...recentMatches].reverse().map((m) => ({
              matchId: m.match_id,
              outcome:
                m.outcome_tone === 'positive'
                  ? ('win' as const)
                  : m.outcome_tone === 'negative'
                    ? ('loss' as const)
                    : ('tie' as const),
              map: m.detail,
            }))}
            labels={{
              win: fieldMappings?.outcomes?.['win']?.label ?? 'win',
              loss: fieldMappings?.outcomes?.['loss']?.label ?? 'loss',
              tie: fieldMappings?.outcomes?.['tie']?.label ?? 'tie',
              dnf: fieldMappings?.outcomes?.['dnf']?.label ?? 'dnf',
            }}
          />
        )}

        {/* Matchs récents / Favoris — 4 tuiles MatchCard */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">
                {matchTab === 'recent' ? t('home.matches.recent_title') : t('home.matches.favorites_title')}
              </CardTitle>
              <div className="flex items-center gap-1 border-b border-transparent">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setMatchTab('recent')}
                  className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                    matchTab === 'recent'
                      ? 'border-primary text-primary font-semibold'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {t('home.matches.tab_recent')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setMatchTab('favorites')}
                  className={`rounded-none border-b-2 px-3 py-1.5 text-xs ${
                    matchTab === 'favorites'
                      ? 'border-primary text-primary font-semibold'
                      : 'border-transparent text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {t('home.matches.tab_favorites')}
                  {favoriteMatches.length > 0 && (
                    <span className="ml-1.5 rounded-full bg-amber-500/20 px-1.5 py-0.5 text-[10px] text-amber-400">
                      {favoriteMatches.length}
                    </span>
                  )}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {matchTab === 'recent' ? (
              recentMatches.length > 0 ? (
                <Carousel>
                  {recentMatches.map((m) => (
                    <CarouselItem key={m.match_id} className="w-72">
                      <MatchCard
                        match={m}
                        locale={locale}
                        timezone={userTimezone}
                        onClick={() => goToMatch(m.match_id)}
                        onToggleFavorite={() =>
                          favoriteMutation.mutate({ matchId: m.match_id, favorite: !m.is_favorite })
                        }
                        favoriteDisabled={favoriteMutation.isPending && favoriteMutation.variables?.matchId === m.match_id}
                      />
                    </CarouselItem>
                  ))}
                </Carousel>
              ) : (
                <EmptyStateNotice
                  title={t('home.matches.recent_empty_title')}
                  description={t('home.matches.recent_empty_description')}
                />
              )
            ) : (
              favoriteMatches.length > 0 ? (
                <Carousel>
                  {favoriteMatches.map((m) => (
                    <CarouselItem key={m.match_id} className="w-72">
                      <MatchCard
                        match={m}
                        locale={locale}
                        timezone={userTimezone}
                        onClick={() => goToMatch(m.match_id)}
                        onToggleFavorite={() =>
                          favoriteMutation.mutate({ matchId: m.match_id, favorite: false })
                        }
                        favoriteDisabled={favoriteMutation.isPending && favoriteMutation.variables?.matchId === m.match_id}
                      />
                    </CarouselItem>
                  ))}
                </Carousel>
              ) : (
                <EmptyStateNotice
                  title={t('home.matches.favorites_empty_title')}
                  description={t('home.matches.favorites_empty_description')}
                />
              )
            )}
          </CardContent>
        </Card>

        <RecentMediaRail playerSlug={playerSlug} />
      </div>
    </div>
  )
}
