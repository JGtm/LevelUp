/**
 * HomePage — Accueil Mission Control (Slice 5).
 */
import { useState } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { CompositeProgressBar } from '@/components/ui/composite-progress-bar'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { Spinner } from '@/components/ui/spinner'
import { MatchCard } from '@/components/ui/match-card'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { HomeBattlePassPanel } from './HomeBattlePassPanel'
import { HomeHeroBanner } from './HomeHeroBanner'
import { HomeChallengesList } from './HomeChallengesList'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'
import { RecentMediaRail } from './RecentMediaRail'
import { useHomePage, useSeasonPassPreview } from './queries'
import { useSetMatchFavorite } from '@/features/match-history/queries'
import { useAppShellStore } from '@/stores/appShellStore'
import type { HomeSkillPeakSummary } from '@/lib/api/types'

function KPICard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="rounded-lg border border-border bg-muted px-4 py-3 text-center">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="text-xl font-bold text-primary">{value}</p>
    </div>
  )
}

interface HomeSkillPeakCardProps {
  label: string
  peak: HomeSkillPeakSummary | null
  numberLocale: string
  testIdPrefix: string
  state: 'value' | 'placement' | 'neutral' | 'absent'
  detail: string
}

function HomeSkillPeakCard({ label, peak, numberLocale, testIdPrefix, state, detail }: HomeSkillPeakCardProps) {
  const isPlacement = state === 'placement'
  const hasValue = Boolean(peak)

  return (
    <div
      data-testid={`${testIdPrefix}-card`}
      className={`flex min-w-[11rem] items-center gap-3 rounded-2xl border px-4 py-3 shadow-[0_12px_30px_rgba(8,15,28,0.24)] backdrop-blur-sm ${hasValue ? 'border-cyan-100/12 bg-slate-950/35' : 'border-white/10 bg-slate-950/22'}`}
    >
      {peak?.badge_image_url ? (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 p-1.5">
          <img
            data-testid={`${testIdPrefix}-badge`}
            src={peak.badge_image_url}
            alt={label}
            className="h-full w-full object-contain"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : isPlacement ? (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 p-1.5">
          <img
            data-testid={`${testIdPrefix}-unranked`}
            src="/static/ranks/Unranked.png"
            alt="En placement"
            className="h-full w-full object-contain opacity-80"
            loading="lazy"
            decoding="async"
          />
        </div>
      ) : (
        <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-slate-950/60 text-[10px] font-semibold uppercase tracking-[0.24em] text-cyan-100/75">
          {label.replace(/[^A-Z]/gi, '').slice(0, 4) || 'MMR'}
        </div>
      )}

      <div className="min-w-0">
        <p className="text-[11px] uppercase tracking-[0.24em] text-cyan-100/68">{label}</p>
        <p data-testid={`${testIdPrefix}-value`} className="mt-1 text-xl font-semibold text-white sm:text-2xl">
          {peak ? peak.rating_value.toLocaleString(numberLocale, { maximumFractionDigits: 0 }) : '—'}
        </p>
        {(peak?.tier_label || detail) && (
          <p
            data-testid={peak?.tier_label ? `${testIdPrefix}-tier` : `${testIdPrefix}-detail`}
            className="truncate text-xs text-cyan-100/78"
          >
            {peak?.tier_label ?? detail}
          </p>
        )}
      </div>
    </div>
  )
}

function resolveSkillPeakState(
  peak: HomeSkillPeakSummary | null,
  hasHistory: boolean,
  mode: 'ranked' | 'unranked',
): Pick<HomeSkillPeakCardProps, 'state' | 'detail'> {
  if (peak) {
    return { state: 'value', detail: '' }
  }
  if (hasHistory) {
    return mode === 'ranked'
      ? { state: 'placement', detail: 'En placement' }
      : { state: 'neutral', detail: 'Sans classement' }
  }
  return {
    state: 'absent',
    detail: mode === 'ranked' ? 'Aucune partie classée' : 'Aucune partie non classée',
  }
}

export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const locale = useAppShellStore((s) => s.locale)
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const {
    data: seasonPass,
    isLoading: isSeasonPassLoading,
    error: seasonPassError,
  } = useSeasonPassPreview(playerSlug)
  const [matchTab, setMatchTab] = useState<'recent' | 'favorites'>('recent')
  const favoriteMutation = useSetMatchFavorite(playerSlug)

  function goToMatch(matchId: string) {
    void navigate({
      to: '/players/$playerSlug/matches/$matchId',
      params: { playerSlug, matchId },
    })
  }

  if (isLoading) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10 sm:min-h-[60vh]">
        <Spinner size="lg" label="Chargement de l'accueil…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title="Accueil indisponible"
            description="La page d'accueil n'a pas pu être chargée pour ce joueur. Vérifie la session ou relance la requête."
            actionLabel="Réessayer"
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
            title="Accueil vide"
            description="Aucune donnée d'accueil n'a été renvoyée pour ce joueur. Vérifie le bootstrap ou les données locales avant de continuer."
            actionLabel="Relancer"
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
  const bannerSurfaceStyle = spartanIdentity?.banner_image_url
    ? {
      backgroundImage: `url(${spartanIdentity.banner_image_url})`,
      backgroundPosition: 'center',
      backgroundSize: 'cover',
    }
    : undefined
  const backdropImageUrl = spartanIdentity?.backdrop_image_url ?? null
  const careerAdornmentUrl = careerRank?.adornment_image_url ?? null
  const identityMonogram = hero.player_name.trim().slice(0, 1).toUpperCase() || 'S'
  const soloSession = data.solo_session ?? null
  const squadSession = data.squad_session ?? null
  const challenges = seasonPass?.challenges
  const challengesCompleted = challenges?.completed ?? 0
  const challengesTotal = challenges?.total ?? 0
  const challengesCompletedLabel = challenges?.available
    ? `${challengesCompleted} / ${challengesTotal} complétés`
    : null
  const numberLocale = locale === 'en' ? 'en-US' : 'fr-FR'
  const labels = locale === 'en'
    ? {
      careerRank: 'Career rank',
      highestCsr: 'Highest CSR',
      highestLusr: 'Highest LUSR',
      currentProgress: 'Current progress',
      rankPrefix: 'Rank',
      maxRank: 'Max rank',
    }
    : {
      careerRank: 'Rang carrière',
      highestCsr: 'Meilleur CSR',
      highestLusr: 'Meilleur LUSR',
      currentProgress: 'Progression actuelle',
      rankPrefix: 'Rang',
      maxRank: 'Rang max',
    }
  const emptySkillPanelTitle = data.privacy_warning?.level && data.privacy_warning.level !== 'none'
    ? 'Classements indisponibles'
    : 'Aucun classement disponible'
  const emptySkillPanelDescription = data.privacy_warning?.level && data.privacy_warning.level !== 'none'
    ? 'Les données compétitives de ce joueur sont partielles ou indisponibles.'
    : 'Ce joueur n’a pas encore de classement CSR ni LUSR.'

  return (
    <div className="relative isolate flex flex-col">
      <div className="sticky top-0 z-0 px-6 pt-0" data-testid="home-hero-banner-sticky">
        <HomeHeroBanner />
      </div>

      <div className="relative z-10 space-y-6 bg-background px-6 pb-6 pt-6">
        {/* B9 : signal discret si données partielles (compte privé) */}
        <PrivacyBanner warning={data.privacy_warning} />

        {/* Hero KPIs */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Performance globale</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {spartanIdentity && (
              <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,19.5rem)] lg:items-start">
                <div className="overflow-hidden rounded-2xl border border-border bg-muted/60 shadow-sm">
                  <div
                    data-testid="home-spartan-identity-banner"
                    className="relative overflow-hidden bg-slate-950"
                  >
                    {careerAdornmentUrl && (
                      <div className="pointer-events-none absolute inset-x-0 top-3 z-[1] flex justify-center px-6">
                        <img
                          data-testid="home-spartan-adornment-image"
                          src={careerAdornmentUrl}
                          alt=""
                          aria-hidden="true"
                          className="h-16 w-auto max-w-[min(22rem,70vw)] object-contain opacity-85 drop-shadow-[0_14px_20px_rgba(8,15,28,0.48)] sm:h-20"
                          loading="lazy"
                          decoding="async"
                        />
                      </div>
                    )}
                    {backdropImageUrl && (
                      <div className="pointer-events-none absolute right-4 top-4 hidden h-24 w-40 overflow-hidden rounded-2xl border border-white/10 bg-slate-950/35 shadow-[0_14px_32px_rgba(8,15,28,0.4)] sm:block">
                        <img
                          data-testid="home-spartan-backdrop-image"
                          src={backdropImageUrl}
                          alt=""
                          aria-hidden="true"
                          className="h-full w-full object-cover opacity-55 saturate-[0.85]"
                          loading="lazy"
                          decoding="async"
                        />
                        <div className="absolute inset-0 bg-[linear-gradient(135deg,rgba(8,15,28,0.86),rgba(8,15,28,0.28),rgba(8,15,28,0.78))]" />
                      </div>
                    )}
                    <div className="pointer-events-none absolute inset-x-0 top-1/2 flex -translate-y-1/2 justify-center px-5 sm:px-6">
                      <div
                        data-testid="home-spartan-banner-shell"
                        className="relative h-24 w-full max-w-[34rem] overflow-hidden rounded-[28px] border border-cyan-100/12 bg-[linear-gradient(90deg,rgba(8,15,28,0.9),rgba(20,33,54,0.76),rgba(8,15,28,0.9))] shadow-[0_18px_34px_rgba(8,15,28,0.45)] sm:h-28"
                      >
                        <div className="absolute inset-[1px] rounded-[26px] border border-white/6" />
                        <div className="absolute inset-0 bg-[radial-gradient(circle_at_center,rgba(97,221,255,0.12),transparent_58%)]" />
                        <div className="absolute inset-x-10 top-1/2 h-px -translate-y-1/2 bg-cyan-100/15" />
                        {bannerSurfaceStyle && (
                          <div
                            data-testid="home-spartan-banner-surface"
                            className="absolute inset-0 bg-center bg-cover bg-no-repeat opacity-95"
                            style={bannerSurfaceStyle}
                          />
                        )}
                      </div>
                    </div>
                    <div className="absolute inset-0 bg-[linear-gradient(115deg,rgba(8,15,28,0.94),rgba(17,24,39,0.82),rgba(8,15,28,0.92))]" />
                    <div className="relative flex flex-col gap-6 px-5 py-8 text-white sm:px-6 lg:flex-row lg:items-center lg:justify-between">
                      <div className="flex min-w-0 items-center gap-4">
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
                            <p className="mt-2 text-sm text-cyan-100/70">Identité Spartan indisponible</p>
                          )}
                        </div>
                      </div>

                      {careerRank && (
                        <div className="flex items-center gap-4 self-start lg:self-center">
                          <div className="min-w-0 text-right lg:max-w-[16rem]">
                            <p className="text-[11px] uppercase tracking-[0.28em] text-cyan-100/70">{labels.careerRank}</p>
                            <p data-testid="home-career-rank-title" className="mt-2 text-lg font-semibold text-white sm:text-xl">
                              {careerRank.rank_title}
                            </p>
                            <p className="mt-2 text-sm text-cyan-100/80">{`${labels.rankPrefix} ${careerRank.rank_number}`}</p>
                          </div>

                          {careerRank.rank_image_url && (
                            <div className="rounded-2xl border border-white/15 bg-slate-950/45 p-2 backdrop-blur-sm">
                              <img
                                data-testid="home-career-rank-image"
                                src={careerRank.rank_image_url}
                                alt={`${labels.careerRank} ${careerRank.rank_title}`}
                                className="h-20 w-20 object-contain sm:h-24 sm:w-24"
                                loading="lazy"
                                decoding="async"
                              />
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>

                  {careerRank && (
                    <div className="border-t border-border/70 bg-background/80 px-5 py-4 sm:px-6">
                      <div className="space-y-3">
                        <div className="flex items-center justify-between gap-3 text-xs text-muted-foreground">
                          <span>{labels.currentProgress}</span>
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
                  className="grid content-start gap-3 sm:grid-cols-2 lg:grid-cols-1"
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

            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
              <KPICard label="Parties" value={hero.kpis.total_matches.toLocaleString('fr-FR')} />
              <KPICard label="Taux de victoire" value={`${(hero.kpis.win_rate * 100).toFixed(0)}%`} />
              <KPICard label="K/D" value={hero.kpis.global_ratio?.toFixed(2) ?? '—'} />
              <KPICard label="Victoires" value={hero.kpis.wins.toLocaleString('fr-FR')} />
              <KPICard label="Défaites" value={hero.kpis.losses.toLocaleString('fr-FR')} />
              <KPICard label="Précision" value={hero.kpis.avg_accuracy != null ? `${hero.kpis.avg_accuracy.toFixed(0)}%` : '—'} />
            </div>
          </CardContent>
        </Card>

        {/* Battle Pass + Défis */}
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <HomeBattlePassPanel
            loading={isSeasonPassLoading}
            data={seasonPass}
            errorHint={seasonPassError instanceof Error ? seasonPassError.message : null}
          />

          <Card data-testid="home-challenges-card" className="flex min-h-[14rem] self-start flex-col">
            <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
              <CardTitle className="text-base">Défis actifs</CardTitle>
              {challengesCompletedLabel && (
                <p data-testid="home-challenges-completed" className="shrink-0 text-sm text-foreground">
                  <strong className="text-primary">{challengesCompleted}</strong> / {challengesTotal} complétés
                </p>
              )}
            </CardHeader>
            <CardContent className="flex flex-1 flex-col">
              {isSeasonPassLoading && !seasonPass ? (
                <div className="flex flex-1 items-center justify-center">
                  <p className="text-sm text-muted-foreground">Chargement des défis…</p>
                </div>
              ) : challenges?.available ? (
                <div className="space-y-3">
                  {Array.isArray(challenges.items) && challenges.items.length > 0 ? (
                    <HomeChallengesList items={challenges.items} />
                  ) : (
                    <div className="flex min-h-[8.5rem] items-center justify-center">
                      <EmptyStateNotice
                        title="Aucun défi actif"
                        description="Tous les défis visibles sont terminés ou aucun défi détaillé n'est disponible pour le moment."
                        className="w-full max-w-sm"
                      />
                    </div>
                  )}
                  {challenges.xp_available != null && challenges.xp_available > 0 && (
                    <p className="text-xs text-muted-foreground">{challenges.xp_available.toLocaleString('fr-FR')} XP disponibles</p>
                  )}
                </div>
              ) : seasonPassError ? (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title="Défis indisponibles"
                    description="Les défis actifs n'ont pas pu être chargés pour le moment."
                    className="w-full max-w-sm"
                  />
                </div>
              ) : (
                <div className="flex flex-1 items-center justify-center">
                  <EmptyStateNotice
                    title="Défis indisponibles"
                    description="Aucune donnée de défis n'est disponible pour le moment."
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
            <CardHeader>
              <CardTitle className="text-base">Sessions récentes</CardTitle>
            </CardHeader>
            <CardContent>
              {soloSession || squadSession ? (
                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  {soloSession && (
                    <div className="rounded-md bg-muted p-3">
                      <Badge variant="secondary" className="mb-2">Solo</Badge>
                      <p className="text-sm font-medium">{soloSession.session_label}</p>
                      <p className="text-xs text-muted-foreground">{soloSession.match_count} parties · {(soloSession.win_rate * 100).toFixed(0)}% W</p>
                    </div>
                  )}
                  {squadSession && (
                    <div className="rounded-md bg-primary/10 p-3">
                      <Badge variant="default" className="mb-2">Escouade</Badge>
                      <p className="text-sm font-medium">{squadSession.session_label}</p>
                      <p className="text-xs text-muted-foreground">{squadSession.match_count} parties · {(squadSession.win_rate * 100).toFixed(0)}% W</p>
                    </div>
                  )}
                </div>
              ) : (
                <EmptyStateNotice
                  title="Aucune session récente disponible"
                  description="Aucune session solo ou escouade n'a été calculée pour le scope actuel."
                />
              )}
            </CardContent>
          </Card>
        </div>

        {/* Highlights */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Points saillants</CardTitle>
          </CardHeader>
          <CardContent>
            {highlights.length > 0 ? (
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {highlights.map((h, i) => (
                  <div key={i} className="rounded-md border border-border p-3">
                    <p className="text-xs font-medium text-muted-foreground">{h.title}</p>
                    <p className="text-base font-bold text-primary">{h.value}</p>
                    <p className="text-xs text-muted-foreground">{h.detail}</p>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucun point saillant disponible"
                description="Le backend n'a renvoyé aucun highlight exploitable pour cette période."
              />
            )}
          </CardContent>
        </Card>

        {/* Matchs récents / Favoris — 4 tuiles MatchCard */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle className="text-base">
                {matchTab === 'recent' ? 'Matchs récents' : 'Matchs favoris'}
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
                  Récents
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
                  Favoris
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
                  title="Aucun match récent disponible"
                  description="Les matchs récents apparaîtront ici après une synchronisation."
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
                  title="Aucun match favori"
                  description="Marquez vos meilleurs matchs avec l'icône étoile pour les retrouver ici."
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
