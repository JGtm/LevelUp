/**
 * HomePage — Accueil Mission Control (Slice 5).
 *
 * P8.4 (revue 2026-04-29) : sub-components extraits dans des fichiers dédiés
 * (HomeOutcomeBar, HomeHighlightTile, HomeSessionCarousel, HomeSkillPeakCard,
 * HomeSpartanIdentityBanner, HomeHeroKPIGrid) — réduit ce fichier de ~720L.
 */
import { useState } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { DataFreshnessIndicator } from '@/components/ui/data-freshness-indicator'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { MatchCard } from '@/components/ui/match-card'
import { Carousel, CarouselItem } from '@/components/ui/carousel'
import { HomeBattlePassPanel } from './HomeBattlePassPanel'
import { HomeHeroBanner } from './HomeHeroBanner'
import { HomeChallengesList } from './HomeChallengesList'
import { HomeRecentPlaylistsCard } from './HomeRecentPlaylistsCard'
import { RecentMediaRail } from './RecentMediaRail'
import { HomeHighlightTile } from './HomeHighlightTile'
import { HomeSessionCarousel } from './HomeSessionCarousel'
import { HomeSpartanIdentityBanner } from './HomeSpartanIdentityBanner'
import { HomeHeroKPIGrid } from './HomeHeroKPIGrid'
import { SpartanCustomizerModal } from '@/features/spartan-customizer/SpartanCustomizerModal'
import { HomePrestigeSection } from './HomePrestigeSection'
import { HomeAscensionWidget } from './HomeAscensionWidget'
import { HomeCitationsNearCompletion } from './HomeCitationsNearCompletion'
import { useHomePage, useSeasonPassPreview } from './queries'
import { useSettings } from '@/features/settings/queries'
import { useSetMatchFavorite } from '@/features/match-history/queries'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { DEFAULT_GAP_MINUTES } from '@/stores/filterDefaults'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import { useCapability } from '@/lib/capabilities/capabilities'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { OutcomeSequenceTape } from '@/components/charts/OutcomeSequenceTape'
import { getHighlightText } from './highlights.i18n'
import { getKPIText } from './kpi.i18n'
import { formatMessage } from '@/lib/i18n/format'
import { homeManifest, type HomeManifestKey } from '@/lib/i18n/generated/home'
import { InfoTooltip } from '@/components/ui/info-tooltip'


export function HomePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const locale = useAppShellStore((s) => s.locale)
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const userTimezone = useAppShellStore((s) => s.userTimezone)
  const { data: fieldMappings } = useFieldMappings()
  const { data: settings } = useSettings()
  const showProgression = settings?.show_progression ?? true
  const t = (key: HomeManifestKey, values?: Record<string, string | number>) =>
    formatMessage(homeManifest, key, locale, values)
  // Gating multi-titre (Phase 5) : la carte « Playlists récentes » (CSR) dépend de
  // `ranked`. NO-OP pour halo_infinite ; pour un titre sans `ranked`, la grille se
  // replie sur les seules « Sessions récentes » (transverses) au lieu d'une colonne vide.
  const hasRanked = useCapability('ranked')
  // Gating multi-titre : Battle Pass + Défis dépendent de `season_pass`. NO-OP
  // pour halo_infinite ; pour un titre sans cette capability (ex. Halo 5), tout
  // le bloc est masqué (au lieu d'un état « indisponible ») et la requête
  // season-pass n'est pas émise.
  const hasSeasonPass = useCapability('season_pass')
  // Halo 5 only : personnalisateur Spartan (emblème/nameplate recolorisables) accessible
  // via la bannière d'identité. Absent sur Infinite ⇒ bannière non cliquable, modale jamais montée.
  const hasSpartanCustomizer = useCapability('spartan_customizer')
  const { data, isLoading, isError, refetch } = useHomePage(playerSlug)
  const {
    data: seasonPass,
    isLoading: isSeasonPassLoading,
    error: seasonPassError,
  } = useSeasonPassPreview(playerSlug, hasSeasonPass)
  const [matchTab, setMatchTab] = useState<'recent' | 'favorites'>('recent')
  const [soloIdx, setSoloIdx] = useState(0)
  const [squadIdx, setSquadIdx] = useState(0)
  const [customizerOpen, setCustomizerOpen] = useState(false)
  const favoriteMutation = useSetMatchFavorite(playerSlug)
  const navigateToMatch = useNavigateToMatch(playerSlug)

  function goToMatch(matchId: string, source: 'home_recent' | 'home_favorites') {
    const list = source === 'home_recent' ? recentMatches : favoriteMatches
    navigateToMatch(matchId, {
      source,
      matchIds: list.map((m) => m.match_id),
      contextDescriptor:
        source === 'home_recent' ? { kind: 'recent' } : { kind: 'favorites' },
    })
  }

  // Card SOLO → Timeseries (la page de stats solo), scopée sur la session. On
  // épingle la session dans le store solo (même mécanisme que la pill Session) ;
  // Timeseries lit ce store et useFollowLatestSession ne re-snappe pas une session
  // épinglée (followLatest=false dès que picked_sessions est non vide).
  function goToSoloSession(sessionLabel: string) {
    const { filterContext, setSessions } = useSoloFilterStore.getState()
    setSessions({
      picked_sessions: [sessionLabel],
      gap_minutes: filterContext.sessions?.gap_minutes ?? DEFAULT_GAP_MINUTES,
    })
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/stats/timeseries',
      params: { titleSlug, playerSlug },
    })
  }

  // Card ESCOUADE → page /squad, scopée sur la session AVEC ses coéquipiers
  // pré-sélectionnés. SquadLayout consomme ces search params au montage.
  function goToSquadSession(sessionLabel: string, teammates: string[]) {
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/squad',
      params: { titleSlug, playerSlug },
      search: {
        session: sessionLabel,
        // Gamertags joints par virgule (aucun gamertag Xbox n'en contient).
        teammates: teammates.length > 0 ? teammates.join(',') : undefined,
      },
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

  // AXE E — première synchronisation : un titre fraîchement activé n'a pas encore de
  // matchs synchronisés. Plutôt qu'un accueil vide/cassé, on montre un écran « sync en
  // cours » qui rassure le joueur (sync en arrière-plan, on notifiera quand c'est prêt,
  // continue sur Halo Infinite en attendant). Signal AUTORITATIF = total_matches du
  // hero (≠ recent_matches qui est une fenêtre ; l'accueil agrège aussi des sources
  // indépendantes — média, identité live — non liées au sync des matchs).
  if ((hero.kpis?.total_matches ?? 0) <= 0) {
    return (
      <div className="flex min-h-[55vh] items-center justify-center px-6 py-10">
        <div className="mx-auto w-full max-w-lg">
          <EmptyStateCard
            title={t('home.first_sync.title')}
            description={t('home.first_sync.description')}
            actionLabel={t('home.first_sync.action')}
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const spartanIdentity = data.spartan_identity ?? null
  const highestCSR = spartanIdentity?.highest_csr ?? null
  const highestLUSR = spartanIdentity?.highest_lusr ?? null
  const hasRankedHistory = data.has_ranked_history ?? Boolean(highestCSR)
  const hasUnrankedHistory = data.has_unranked_history ?? Boolean(highestLUSR)
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
  const numberLocale = intlLocale(locale)
  // Séquence d'outcomes : filtrer recent_matches par le session_label de la
  // dernière session connue (solo ou squad). On prend le premier label qui
  // apparaît dans recent_matches (trié DESC) parmi les deux sessions.
  const knownLabels = new Set(
    [soloSession?.session_label, squadSession?.session_label].filter(Boolean) as string[],
  )
  const lastSessionLabel = recentMatches.find((m) => m.session_label && knownLabels.has(m.session_label))?.session_label ?? null
  const lastSessionMatches = lastSessionLabel
    ? recentMatches.filter((m) => m.session_label === lastSessionLabel)
    : recentMatches.slice(0, Math.min(20, recentMatches.length))
  const kpiText = getKPIText(locale)
  // Phase D multi-titres : résout les libellés métier via le backend TOML.
  // Fallback gracieux sur les libellés locaux de kpi.i18n.ts si l'endpoint
  // est absent (flag MULTI_TITLE_API_ENABLED off ou 404) — sinon les tuiles
  // affichent la clé canonique brute (kda, win_rate, accuracy) au lieu du
  // libellé traduit. La tuile « Matchs » utilise directement kpiText (libellé
  // fixe court, indépendant du field-mapping backend « Parties jouées »).
  const localKpiLabels: Record<string, string> = {
    kda: kpiText.labels.kda,
    win_rate: kpiText.labels.winRate,
    accuracy: kpiText.labels.accuracy,
  }
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? localKpiLabels[key] ?? key
  const hasPrivacyWarning = !!data.privacy_warning?.level && data.privacy_warning.level !== 'none'

  return (
    <div className="relative isolate flex flex-col">
      <div className="sticky top-0 z-0 px-6 pt-0" data-testid="home-hero-banner-sticky">
        <HomeHeroBanner />
      </div>

      <div className="relative z-10 space-y-6 bg-background px-6 pb-6 pt-6">
        {/* B9 : signal discret si données partielles (compte privé) */}
        <PrivacyBanner warning={data.privacy_warning} />

        {/* Hero KPIs — bannière Spartan + grille 8 tuiles. P8.4 finition (revue 2026-04-29). */}
        <div>
          <div className="space-y-4">
            <HomeSpartanIdentityBanner
              spartanIdentity={spartanIdentity ?? {}}
              playerName={hero.player_name}
              highestCSR={highestCSR}
              highestLUSR={highestLUSR}
              hasRankedHistory={hasRankedHistory}
              hasUnrankedHistory={hasUnrankedHistory}
              hasPrivacyWarning={hasPrivacyWarning}
              identityUnavailableLabel={t('home.identity.unavailable')}
              onSpartanClick={
                hasSpartanCustomizer ? () => setCustomizerOpen(true) : undefined
              }
              spartanCustomizeLabel={t('home.spartan_customizer.open_aria')}
            />
            {hasSpartanCustomizer && customizerOpen && (
              <SpartanCustomizerModal onClose={() => setCustomizerOpen(false)} />
            )}

            <HomeHeroKPIGrid
              kpis={hero.kpis}
              labelOf={labelOf}
              numberLocale={numberLocale}
              kpiText={kpiText}
            />
          </div>
        </div>


        {/* Battle Pass + Défis — masqués pour un titre sans capability season_pass. */}
        {hasSeasonPass && (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(320px,0.65fr)]">
          <HomeBattlePassPanel
            loading={isSeasonPassLoading}
            data={seasonPass}
            errorHint={seasonPassError instanceof Error ? seasonPassError.message : null}
          />

          <section className="flex flex-col gap-3">
            {/* Titre de section (type 1) + (i) freshness, SORTI de la carte (cf. demande user). */}
            <header className="flex flex-row items-center justify-between gap-3">
              <div className="flex items-center gap-1.5">
                <h3 className="text-base font-semibold text-foreground">{t('home.challenges.title')}</h3>
                <DataFreshnessIndicator
                  snapshotAt={challenges?.snapshot_at}
                  buildLabel={(date) =>
                    challenges?.from_cache
                      ? t('home.freshness.from_cache', { date })
                      : t('home.freshness.last_sync', { date })
                  }
                  locale={numberLocale}
                />
              </div>
              {challengesCompletedLabel && (
                <p data-testid="home-challenges-completed" className="shrink-0 text-sm text-foreground">
                  <strong className="text-primary">{challengesCompleted}</strong> / {challengesTotal}{' '}
                  {locale === 'en' ? 'completed' : 'complétés'}
                </p>
              )}
            </header>
            <Card data-testid="home-challenges-card" className="flex min-h-[14rem] flex-1 flex-col">
              <CardContent className="flex flex-1 flex-col pt-6">
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
          </section>
        </div>
        )}

        {/* Playlists récentes + Rang | Sessions récentes */}
        <div
          className={
            hasRanked
              ? 'grid grid-cols-1 gap-4 xl:grid-cols-[minmax(320px,0.65fr)_minmax(0,1.35fr)]'
              : 'grid grid-cols-1 gap-4'
          }
        >
          {hasRanked && (
            <HomeRecentPlaylistsCard
              recentPlaylistRanks={data.recent_playlist_ranks}
            />
          )}

          {/* Sessions récentes */}
          <section className="flex flex-col gap-3">
            {/* Titre de section (type 1 du catalogue), SORTI de la carte (cf. demande user). */}
            <header className="flex items-center gap-1.5">
              <h3 className="text-base font-semibold text-foreground">{t('home.sessions.title')}</h3>
            </header>
            <Card className="flex flex-1 flex-col">
            <CardContent className="pt-6 pb-0">
              {soloSessions.length > 0 || squadSessions.length > 0 ? (
                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  {soloSessions.length > 0 && (
                    <HomeSessionCarousel
                      sessions={soloSessions}
                      idx={soloIdx}
                      onIdxChange={setSoloIdx}
                      variant="solo"
                      playerSlug={playerSlug}
                      onNavigate={goToSoloSession}
                    />
                  )}
                  {squadSessions.length > 0 && (
                    <HomeSessionCarousel
                      sessions={squadSessions}
                      idx={squadIdx}
                      onIdxChange={setSquadIdx}
                      variant="squad"
                      playerSlug={playerSlug}
                      onNavigate={goToSquadSession}
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
          </section>
        </div>

        {/* Section Prestige (2/3) + Ascension (1/3) — masquées si show_progression=false */}
        {showProgression && (
          <div className="grid grid-cols-1 gap-4 xl:grid-cols-[2fr_1fr]">
            <HomePrestigeSection playerSlug={playerSlug} titleSlug={titleSlug} locale={locale} />
            <HomeAscensionWidget playerSlug={playerSlug} locale={locale} />
          </div>
        )}

        {/* Faits marquants — KPI sortis de leur bloc : chaque tuile est une KpiCard
            autonome, plus de carte englobante (cf. demande user). */}
        <section className="flex flex-col gap-3">
          <header className="flex items-center gap-1.5">
            <h3 className="text-base font-semibold text-foreground">{getHighlightText(locale).section.title}</h3>
            <InfoTooltip content={<p>{getHighlightText(locale).section.tooltipIntro}</p>} />
          </header>
          {highlights.length > 0 ? (
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:flex lg:flex-wrap">
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
        </section>

        {/* Résultats de la dernière session — titre + contenu sortis de la carte (cf. demande user). */}
        {lastSessionMatches.length > 0 && (
          <section className="flex flex-col gap-3">
            <header className="flex items-center gap-1.5">
              <h3 className="text-base font-semibold text-foreground">
                {locale === 'en' ? 'Last session results' : 'Résultats de la dernière session'}
              </h3>
            </header>
            <OutcomeSequenceTape
                matches={[...lastSessionMatches].reverse().map((m) => {
                  const outcome: 'win' | 'loss' | 'tie' | 'dnf' =
                    m.outcome_tone === 'win'
                      ? 'win'
                      : m.outcome_tone === 'loss'
                        ? 'loss'
                        : m.outcome_tone === 'dnf'
                          ? 'dnf'
                          : 'tie'
                  return { matchId: m.match_id, outcome, map: m.detail }
                })}
                labels={{
                  win: fieldMappings?.outcomes?.['win']?.label ?? 'win',
                  loss: fieldMappings?.outcomes?.['loss']?.label ?? 'loss',
                  tie: fieldMappings?.outcomes?.['tie']?.label ?? 'tie',
                  dnf: fieldMappings?.outcomes?.['dnf']?.label ?? 'dnf',
                }}
              />
          </section>
        )}

        {/* Citations bientôt terminées — juste au-dessus des tuiles de matchs.
            Self-hide si rien à montrer (cf. HomeCitationsNearCompletion). */}
        <HomeCitationsNearCompletion playerSlug={playerSlug} />

        {/* Matchs récents / Favoris — titre + contenu sortis de la carte ; toggles
            conservés à droite dans le header de section (cf. demande user). */}
        <section className="flex flex-col gap-3">
          <header className="flex items-center justify-between gap-3">
            <h3 className="text-base font-semibold text-foreground">
              {matchTab === 'recent' ? t('home.matches.recent_title') : t('home.matches.favorites_title')}
            </h3>
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
                    <span className="ml-1.5 rounded-full bg-warning/20 px-1.5 py-0.5 text-2xs text-warning">
                      {favoriteMatches.length}
                    </span>
                  )}
                </Button>
              </div>
          </header>
          {matchTab === 'recent' ? (
              recentMatches.length > 0 ? (
                <Carousel>
                  {recentMatches.map((m) => (
                    <CarouselItem key={m.match_id} className="w-72">
                      <MatchCard
                        match={m}
                        locale={locale}
                        timezone={userTimezone}
                        onClick={() => goToMatch(m.match_id, 'home_recent')}
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
                        onClick={() => goToMatch(m.match_id, 'home_favorites')}
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
        </section>

        <RecentMediaRail playerSlug={playerSlug} />
      </div>
    </div>
  )
}
