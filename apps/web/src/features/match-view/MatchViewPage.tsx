/** MatchViewPage — détail d'un match (3 onglets : Général, Chronologie, Joueurs). */
import { useParams, useSearch, useNavigate, useRouter } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useSettings } from '@/features/settings/queries'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useMatchView, useMatchObjectiveEvents, useMatchPositions } from './queries'
import { MatchBreadcrumb, MatchNavigationBar, MatchHeaderCard } from './MatchHeader'
import { MatchSummaryCardsSection } from './MatchStatCards'
import { MatchKdaExpectedChart, MatchSpreeChart, MatchSummaryRadarChart } from './MatchSummaryCharts'
import { MatchFragCard } from './MatchFragCard'
import { MatchMediaTab } from './MatchMediaTab'
import {
  MatchMedalsSection,
  MatchCitationsSection,
  MatchNativeCommendationsSection,
} from './MatchSummaryMedalsAndCitations'
import { MatchViewTabChronology } from './MatchViewTabChronology'
import { MatchViewTabPlayers } from './MatchViewTabPlayers'
import { buildMatchHeadingStr } from './format'
import { MATCH_VIEW_TEXT } from './i18n'
import type { MatchViewTab } from './tabs'
import type { MatchViewRadarSeries } from '@/lib/api/types'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { PageUnavailable } from '@/components/ui/page-unavailable'
import { apiErrorCode } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'

/**
 * Traduit un code stable de partial_reason en impact end-user concret.
 * Décrit ce que l'utilisateur ne peut PAS voir, pas la raison technique.
 */
function translatePartialReason(code: string, locale: string): string {
  const isEN = locale === 'en'
  switch (code) {
    case 'scoreboard_empty':
      return isEN
        ? 'Scoreboard and individual player stats are unavailable'
        : "Le tableau des scores et les stats individuelles sont indisponibles"
    case 'events_empty':
      return isEN
        ? 'Combat charts cannot be displayed — Cadence, Dominance and Frag diff are empty'
        : "Les graphes de combat sont vides — Cadence, Dominance et Frags différentiel ne peuvent pas être tracés"
    case 'player_stats_empty':
      return isEN
        ? 'Some personal stats are missing — datas may be incomplete'
        : "Certaines statistiques personnelles sont absentes — les données peuvent être incomplètes"
    case 'medals_empty':
      return isEN
        ? 'Medals and commendations are unavailable'
        : "Les médailles et citations ne sont pas disponibles"
    case 'combat_narrative_unavailable':
      return isEN
        ? 'Combat charts (Cadence, Dominance, Frag diff) are not available for this match'
        : "Les graphes de combat (Cadence, Dominance, Frags différentiel) ne sont pas disponibles pour ce match"
    case 'citations_unavailable':
      return isEN
        ? 'Citations are not available for this match'
        : "Les citations ne sont pas disponibles pour ce match"
    case 'media_unavailable':
      return isEN
        ? 'Media (captures, clips) are not available for this match'
        : "Les médias (captures, clips) ne sont pas disponibles pour ce match"
    case 'accuracy_damage_taken_native_unavailable':
      return isEN
        ? 'Accuracy and damage taken are not provided for this match'
        : "La précision et les dégâts subis ne sont pas fournis pour ce match"
    default:
      return code
  }
}

// Libellés résolus au rendu via MATCH_VIEW_TEXT (GH2-B2 : bilingue). Les ids
// canoniques et la rétro-compat des deep-links vivent dans `./tabs`.
const TABS: { id: MatchViewTab; labelKey: 'tabGeneral' | 'tabChronology' | 'tabPlayers' }[] = [
  { id: 'summary', labelKey: 'tabGeneral' },
  { id: 'chronology', labelKey: 'tabChronology' },
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
  // Tirés UNIQUEMENT sur l'onglet Chronologie : leurs seuls consommateurs (frags
  // cumulés, dominance, heatmap des positions) y vivent.
  const isChronology = activeTab === 'chronology'
  const { data: objectiveEvents } = useMatchObjectiveEvents(playerSlug, matchId, isChronology)
  const { data: matchPositions } = useMatchPositions(playerSlug, matchId, isChronology)
  const { data: settings } = useSettings()
  const friendGamertags = settings?.friend_gamertags ?? []
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
              ? "You didn't take part in this match, so it can't be shown here."
              : "Tu n'as pas participé à ce match, il ne peut donc pas être affiché ici."
          }
          actions={[
            { label: isEN ? 'Home' : 'Accueil', onClick: goHome, variant: 'default' },
            { label: isEN ? 'Back' : 'Précédent', onClick: goBack },
            { label: isEN ? 'My matches' : 'Mes matchs', onClick: goMatches },
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
              ? 'This player is not associated with your account.'
              : "Ce joueur n'est pas associé à ton compte."
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
            { label: isEN ? 'My matches' : 'Mes matchs', onClick: goMatches },
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
  const weaponKills = combat_tab.weapon_kills ?? []
  const highlightEvents = combat_tab.highlight_events ?? []
  const killerVictim = combat_tab.killer_victim ?? []
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

      {/* Sprint 54-B : avertissement privacy */}
      {data.privacy_warning && (
        <div className="px-6 pt-4">
          <PrivacyBanner warning={data.privacy_warning} />
        </div>
      )}

      {/* RC6 — bandeau "sync incomplet" : le match existe mais des sections
          critiques sont vides (parser highlight expirée, sync partiel, etc.).
          Le match reste rendu normalement, on signale juste la dégradation. */}
      {data.is_partial && data.partial_reasons && data.partial_reasons.length > 0 && (
        <div className="px-6 pt-4">
          <div className="flex gap-3 rounded-md border border-warning/50 bg-warning/10 px-4 py-3">
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 20 20"
              fill="currentColor"
              className="mt-0.5 h-4 w-4 shrink-0 text-warning"
              aria-hidden="true"
            >
              <path
                fillRule="evenodd"
                d="M8.485 2.495c.673-1.167 2.357-1.167 3.03 0l6.28 10.875c.673 1.167-.17 2.625-1.516 2.625H3.72c-1.347 0-2.189-1.458-1.515-2.625L8.485 2.495zM10 5a.75.75 0 01.75.75v3.5a.75.75 0 01-1.5 0v-3.5A.75.75 0 0110 5zm0 9a1 1 0 100-2 1 1 0 000 2z"
                clipRule="evenodd"
              />
            </svg>
            <div className="min-w-0 flex-1">
              <p className="text-sm font-semibold text-foreground">
                {t.pagePartialLoad}
              </p>
              <ul className="mt-1.5 space-y-1">
                {data.partial_reasons.map((r) => (
                  <li key={r} className="flex gap-2 text-xs text-muted-foreground">
                    <span className="mt-px shrink-0 text-warning" aria-hidden="true">▸</span>
                    <span>{translatePartialReason(r, locale)}</span>
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      )}

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
          <div className="space-y-4">
            <MatchSummaryCardsSection
              kpis={summary_tab.kpis}
              expectedStats={summary_tab.expected_stats}
              offensiveConversion={meRow?.offensive_conversion ?? null}
              defensiveResistance={meRow?.defensive_resistance ?? null}
              damagePerKill={meRow?.damage_per_kill ?? null}
              damagePerDeath={meRow?.damage_per_death ?? null}
            />
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
            {/* Répartition des frags v2 sur SA PROPRE rangée : sunburst (classe→rôle,
                2/3 de largeur) + breakdown par arme (1/3). MatchFragCard porte sa
                propre grille (breakdown pleine largeur si le sunburst n'a pas de
                données) et rend null sans aucune donnée — pas de wrapper ici, sinon
                un gap fantôme resterait quand la carte est absente. Non gaté :
                Infinite = classes sans Spartan ; Halo 5 = avec (capability
                native_kill_mechanics côté backend). */}
            <MatchFragCard distribution={combat_tab.frag_distribution} weapons={weaponKills} />
            {/* Rangée suivante : Médailles À GAUCHE des Citations — grille fluide
                (auto-fit) : chaque carte garde une largeur pleine ou partagée sans
                cellule orpheline. Halo 5 : commendations NATIVES (citations_tab.
                native_commendations) affichées À LA PLACE des citations dérivées
                d'Infinite (summary_tab.citations vide pour h5). Un seul bloc
                « commendations » par titre. */}
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
            {/* Bloc Médias : TOUJOURS en dernier, seul sur sa rangée, PLEINE largeur
                (règle produit : aucun bloc seul à largeur partielle). Gaté sur
                `media` : masque l'en-tête + le bloc entier pour un titre sans
                captures/clips. */}
            <FeatureGate capability="media">
              <div className="rounded-lg border border-border bg-card">
                <div className="border-b border-border px-3 py-2 text-sm font-medium">{t.sectionMedia}</div>
                <div className="p-3">
                  <MatchMediaTab
                    items={media_tab.media_items ?? []}
                    playerSlug={playerSlug}
                    matchId={matchId}
                    locale={locale === 'en' ? 'en' : 'fr'}
                  />
                </div>
              </div>
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
            matchPositions={matchPositions}
            tugOfWar={tugOfWar}
            cadence={combat_tab.cadence}
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
