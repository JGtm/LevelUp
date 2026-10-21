/** MatchViewPage — détail d'un match (4 onglets : Général, Chronologie, Armes et terrain, Joueurs). */
import { useParams, useSearch, useNavigate, useRouter } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { DetailSection } from '@/components/ui/detail-section'
import { useFriendGamertags } from '@/features/friends/queries'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useMatchView, useMatchObjectiveEvents, useMatchPositions } from './queries'
import { MatchBreadcrumb, MatchNavigationBar, MatchHeaderCard } from './MatchHeader'
import { MatchNotices } from './MatchNotices'
import { MatchSummaryCardsSection } from './MatchStatCards'
import { MatchKdaExpectedChart, MatchSpreeChart, MatchSummaryRadarChart } from './MatchSummaryCharts'
import { MatchMediaTab } from './MatchMediaTab'
import {
  MatchMedalsSection,
  MatchCitationsSection,
  MatchNativeCommendationsSection,
} from './MatchSummaryMedalsAndCitations'
import { MatchViewTabChronology } from './MatchViewTabChronology'
import { MatchViewTabArsenal } from './MatchViewTabArsenal'
import { MatchViewTabPlayers } from './MatchViewTabPlayers'
import { buildMatchHeadingStr } from './format'
import { MATCH_VIEW_TEXT } from './i18n'
import type { MatchViewTab } from './tabs'
import type { MatchViewRadarSeries } from '@/lib/api/types'
import { PageUnavailable } from '@/components/ui/page-unavailable'
import { apiErrorCode } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

// Libellés résolus au rendu via MATCH_VIEW_TEXT (GH2-B2 : bilingue). Les ids
// canoniques et la rétro-compat des deep-links vivent dans `./tabs`.
const TABS: {
  id: MatchViewTab
  labelKey: 'tabGeneral' | 'tabChronology' | 'tabArsenal' | 'tabPlayers'
}[] = [
  { id: 'summary', labelKey: 'tabGeneral' },
  { id: 'chronology', labelKey: 'tabChronology' },
  { id: 'arsenal', labelKey: 'tabArsenal' },
  { id: 'players', labelKey: 'tabPlayers' },
]

export function MatchViewPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }
  const { tab } = useSearch({
    from: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
  })
  const activeTab: MatchViewTab = tab ?? 'summary'
  const navigate = useNavigate({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId' })
  const router = useRouter()
  const setActiveTab = (next: MatchViewTab) => {
    navigate({ search: (prev) => ({ ...prev, tab: next }), replace: true }).catch(() => {})
  }
  const { data, isPending, isError, error, refetch } = useMatchView(playerSlug, matchId)
  // Deux calques décodés du film, best-effort : un titre sans film répond 503 et
  // `data` reste undefined — la page s'affiche entière, sans placeholder mort.
  // Chacun n'est tiré QUE sur l'onglet qui le consomme : les événements d'objectif
  // sur Chronologie (frags cumulés, dominance), les positions sur « Armes et terrain »
  // (occupation du terrain, 2026-09-19 ; onglet renommé le 2026-09-22).
  const isChronology = activeTab === 'chronology'
  const isArsenal = activeTab === 'arsenal'
  const { data: objectiveEvents } = useMatchObjectiveEvents(playerSlug, matchId, isChronology)
  const { data: matchPositions } = useMatchPositions(playerSlug, matchId, isArsenal)
  const friendGamertags = useFriendGamertags(playerSlug)
  const locale = useAppShellStore((s) => s.locale)
  const t = MATCH_VIEW_TEXT[locale === 'en' ? 'en' : 'fr']

  if (isPending) return null

  const isEN = locale === 'en'
  const goHome = () => {
    navigate({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home', params: { playerSlug } }).catch(() => {})
  }
  const goMatches = () => {
    navigate({ to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer', params: { playerSlug } }).catch(() => {})
  }
  const goBack = () => {
    if (router.history.length > 1) router.history.back()
    else goMatches()
  }

  if (isError || !data) {
    // ADR 0029 Couche B : match existant mais joueur non-participant (404
    // match_not_participant) ou accès refusé (403 player_forbidden) → page
    // "Indisponible" claire avec navigation, plutôt qu'une page mal renseignée.
    const code = apiErrorCode(error)
    if (code === 'match_not_participant') {
      return (
        <PageUnavailable
          title={isEN ? 'Match unavailable' : 'Match indisponible'}
          description={
            isEN
              ? 'The player of this page did not take part in this match, so it cannot be shown here.'
              : "Le joueur de la page n'a pas participé à ce match : il ne peut pas être affiché ici."
          }
          actions={[
            { label: isEN ? 'Home' : 'Accueil', onClick: goHome, variant: 'default' },
            { label: isEN ? 'Back' : 'Précédent', onClick: goBack },
            { label: t.navMatches, onClick: goMatches },
          ]}
        />
      )
    }
    if (code === 'player_forbidden') {
      return (
        <PageUnavailable
          title={isEN ? 'Access denied' : 'Accès non autorisé'}
          description={
            isEN
              ? 'This player is not associated with the signed-in account.'
              : "Ce joueur n'est pas associé au compte connecté."
          }
          actions={[{ label: isEN ? 'Home' : 'Accueil', onClick: goHome, variant: 'default' }]}
        />
      )
    }
    if (code === 'match_not_found') {
      // Match absent du substrat local (jamais synchronisé, ou pas encore) : le
      // backend ne fait plus AUCUN fetch live vers l'API du titre depuis cette
      // page (fallback LIVE retiré le 2026-07-25, cf. BACKLOG "Retirer le
      // fallback LIVE du Match view"). État dédié plutôt qu'un écran d'erreur
      // générique — même famille que match_not_participant ci-dessus.
      return (
        <PageUnavailable
          title={t.notSyncedTitle}
          description={t.notSyncedDescription}
          actions={[
            { label: isEN ? 'Home' : 'Accueil', onClick: goHome, variant: 'default' },
            { label: isEN ? 'Back' : 'Précédent', onClick: goBack },
            { label: t.navMatches, onClick: goMatches },
          ]}
        />
      )
    }
    // 404 strict ou erreur réseau. On garde la barre de navigation pour
    // permettre à l'utilisateur de continuer à naviguer entre les matchs.
    return (
      <div className="flex flex-col">
        <MatchBreadcrumb playerSlug={playerSlug} matchLabel={t.mapUnknown} locale={locale === 'en' ? 'en' : 'fr'} />
        <MatchNavigationBar playerSlug={playerSlug} matchId={matchId} locale={locale === 'en' ? 'en' : 'fr'} />
        <div className="p-6">
          <Card>
            <CardContent className="py-8 text-center">
              <p className="font-medium text-destructive">
                {t.pageErrorTitle}
              </p>
              {error && (error as { message?: string }).message && (
                <p className="mt-1 text-xs text-muted-foreground font-mono">
                  {(error as { message: string }).message}
                </p>
              )}
              <div className="mt-4">
                <Button variant="outline" size="sm" onClick={() => refetch()}>
                  {t.pageRetry}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  const { header, rank, summary_tab, combat_tab, team_tab, media_tab, citations_tab } = data
  const matchLabel = buildMatchHeadingStr(header.map_ui, header.mode_ui, locale)
  // Le breadcrumb ajoute la date pour distinguer plusieurs matchs sur la même map/mode
  const breadcrumbLabel = header.start_time_label
    ? `${matchLabel} · ${header.start_time_label}`
    : matchLabel
  // Sécurité : le backend Go peut renvoyer `null` pour les slices non
  // initialisés (`var x []T` non assigné → JSON null). On normalise une fois.
  const scoreboard = team_tab.scoreboard ?? []
  const roster = team_tab.roster ?? []
  const nemesis = team_tab.nemesis ?? []
  const highlightEvents = combat_tab.highlight_events ?? []
  const killerVictim = combat_tab.killer_victim ?? []
  // assist_pairs n'est PAS normalisé en objet vide : son absence est un ÉTAT
  // (aucune ligne de film pour ce match) que le graphe distingue de « mesuré, zéro
  // assistance ». Le combler ici effacerait la distinction avant l'écran.
  const assistPairs = combat_tab.assist_pairs
  const impactBadges = combat_tab.impact_badges ?? []
  const tugOfWar = combat_tab.tug_of_war ?? []

  const meRow = scoreboard.find((r) => r.is_me)
  const meXUID = meRow?.xuid ?? null
  // `radar` est typé `unknown[]` dans le contrat (le générateur OpenAPI a perdu
  // le type d'élément du slice Go `[]MatchViewRadarSeries`). On rétablit le type
  // réel renvoyé par l'API au point de consommation, sans changer la donnée.
  const radarSeries = (data.radar ?? undefined) as MatchViewRadarSeries[] | undefined

  return (
    <div className="flex flex-col">
      <MatchBreadcrumb playerSlug={playerSlug} matchLabel={breadcrumbLabel} locale={locale === 'en' ? 'en' : 'fr'} />

      {/* Barre nav full-width — Match précédent · Match X/Y · Match suivant */}
      <MatchNavigationBar playerSlug={playerSlug} matchId={matchId} locale={locale} />

      {/* Header match — image map + outcome + actions + perf/rang (mock C) */}
      <MatchHeaderCard
        header={header}
        rank={rank}
        matchId={matchId}
        playerSlug={playerSlug}
        matchTitle={matchLabel}
        locale={locale}
      />

      {/* La SEULE zone de messages de la page : lecture en échec, données absentes, analyse du
          film, confidentialité (ordre et libellés : matchNotices.logic.ts). */}
      <MatchNotices data={data} t={t} locale={locale} />

      {/* Onglets — mx-6 aligne la bordure sur le header card (même inset px-6) */}
      <div className="mx-6 mt-4 border-b border-border">
        <div className="flex gap-0">
          {TABS.map((tab) => (
            <Button
              key={tab.id}
              variant="ghost"
              size="sm"
              onClick={() => setActiveTab(tab.id)}
              className={`rounded-none border-b-2 px-4 py-3 text-sm ${
                activeTab === tab.id
                  ? 'border-primary font-semibold text-primary'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
            >
              {t[tab.labelKey]}
            </Button>
          ))}
        </div>
      </div>

      <div className="p-6 space-y-6">
        {activeTab === 'summary' && (
          /* Onglet Général — LE BILAN DU MATCH. Trois sections titrées depuis le
             2026-09-22 (gabarit unique `DetailSection`), plus la bande de KPI qui, elle,
             reste SANS titre : comme sur l'accueil, elle se lit d'elle-même et un titre
             au-dessus n'y ajouterait rien. La répartition des frags et la distance des
             frags ont quitté cet onglet pour « Armes et terrain » : elles disent avec quoi
             on a tué, pas comment le match s'est soldé. */
          <div className="space-y-6">
            <MatchSummaryCardsSection
              kpis={summary_tab.kpis}
              expectedStats={summary_tab.expected_stats}
              offensiveConversion={meRow?.offensive_conversion ?? null}
              defensiveResistance={meRow?.defensive_resistance ?? null}
              damagePerKill={meRow?.damage_per_kill ?? null}
              damagePerDeath={meRow?.damage_per_death ?? null}
            />
            <DetailSection title={t.sectionCombat}>
              <div className="grid grid-cols-1 gap-4 sm:[grid-template-columns:repeat(auto-fit,minmax(280px,1fr))]">
                <MatchKdaExpectedChart
                  kpis={summary_tab.kpis}
                  expectedStats={summary_tab.expected_stats}
                  t={t}
                />
                <MatchSpreeChart
                  kpis={summary_tab.kpis}
                  expectedStats={summary_tab.expected_stats}
                  t={t}
                />
                <MatchSummaryRadarChart
                  radar={radarSeries}
                  meXUID={meXUID}
                  t={t}
                />
              </div>
            </DetailSection>
            {/* Récompenses : Médailles À GAUCHE des Citations — grille fluide
                (auto-fit) : chaque carte garde une largeur pleine ou partagée sans
                cellule orpheline. Halo 5 : commendations NATIVES (citations_tab.
                native_commendations) affichées À LA PLACE des citations dérivées
                d'Infinite (summary_tab.citations vide pour h5). Un seul bloc
                « commendations » par titre. */}
            <DetailSection title={t.sectionRewards}>
              <div className="grid grid-cols-1 gap-4 sm:[grid-template-columns:repeat(auto-fit,minmax(320px,1fr))]">
                <MatchMedalsSection medals={summary_tab.medals ?? []} t={t} />
                {(citations_tab?.native_commendations?.length ?? 0) > 0 ? (
                  <MatchNativeCommendationsSection
                    commendations={citations_tab.native_commendations ?? []}
                    t={t}
                  />
                ) : (
                  <MatchCitationsSection citations={summary_tab.citations ?? []} t={t} />
                )}
              </div>
            </DetailSection>
            {/* Bloc Médias : TOUJOURS en dernier, seul sur sa rangée, PLEINE largeur
                (règle produit : aucun bloc seul à largeur partielle). C'est un BLOC à titre
                (SectionCard, comme Médailles et Citations) : le titre « Médias » est DANS
                la carte, qui contient toutes les vignettes ou l'état vide (2026-10-21 ; en
                v7.3.2 titre et contenu vivaient dans une même carte). Une seule porte : la
                capability `media` (le TITRE mesure des captures et des clips). */}
            <FeatureGate capability="media">
              <MatchMediaTab
                items={media_tab.media_items ?? []}
                playerSlug={playerSlug}
                matchId={matchId}
                t={t}
              />
            </FeatureGate>
          </div>
        )}

        {activeTab === 'chronology' && (
          <MatchViewTabChronology
            playerSlug={playerSlug}
            matchId={matchId}
            replayAvailable={header.replay_available === true}
            impactBadges={impactBadges}
            highlightEvents={highlightEvents}
            scoreboard={scoreboard}
            meXUID={meXUID}
            objectiveEvents={objectiveEvents}
            tugOfWar={tugOfWar}
            cadence={combat_tab.cadence}
            scoreTimelineKind={header.score_timeline_kind}
            t0Ms={header.t0_ms}
            t={t}
          />
        )}

        {activeTab === 'arsenal' && (
          <MatchViewTabArsenal
            playerSlug={playerSlug}
            matchId={matchId}
            replayAvailable={header.replay_available === true}
            scoreboard={scoreboard}
            roster={roster}
            fragDistribution={combat_tab.frag_distribution}
            weaponTools={combat_tab.weapon_tools}
            killDistance={combat_tab.kill_distance_by_weapon}
            emprise={data.emprise}
            livesNearTeammate={data.lives_near_teammate}
            analysisCauses={header}
            meXUID={meXUID}
            friendGamertags={friendGamertags}
            matchPositions={matchPositions}
            locale={locale}
            t={t}
          />
        )}

        {activeTab === 'players' && (
          <MatchViewTabPlayers
            header={header}
            rank={rank}
            scoreboard={scoreboard}
            roster={roster}
            nemesis={nemesis}
            killerVictim={killerVictim}
            assistPairs={assistPairs}
            highlightEvents={highlightEvents}
            citations={summary_tab.citations ?? []}
            encounters={team_tab.encounters ?? []}
            meXUID={meXUID}
            friendGamertags={friendGamertags}
            locale={locale}
            t={t}
          />
        )}
      </div>
    </div>
  )
}
