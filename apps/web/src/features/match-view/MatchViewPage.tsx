/** MatchViewPage — détail d'un match (2 onglets : Général, Détails). */
import { useParams, useSearch, useNavigate, useRouter } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useSettings } from '@/features/settings/queries'
import { EngagementMatchSection } from '@/features/engagement/EngagementMatchSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { useMatchView } from './queries'
import { MatchBreadcrumb, MatchNavigationBar, MatchHeaderCard } from './MatchHeader'
import { MatchAntagonistChart } from './MatchAntagonistChart'
import { MatchFragDiffChart } from './MatchFragDiffChart'
import { MatchImpactBadgesBar } from './MatchImpactBadgesBar'
import { MatchKDCumulChart } from './MatchKDCumulChart'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import { MatchCadenceChart } from './MatchCadenceChart'
import { MatchNemesisCards } from './MatchNemesisCards'
import { MatchScoreboard } from './MatchScoreboard'
import { MatchEncountersTable } from './MatchEncountersTable'
import { MatchSummaryCardsSection } from './MatchStatCards'
import { MatchKdaExpectedChart, MatchSpreeChart, MatchSummaryRadarChart } from './MatchSummaryCharts'
import { MatchWeaponPieChart } from './MatchWeaponCharts'
import { MatchKillTypesDonut } from './MatchKillTypesDonut'
import { MatchMediaTab } from './MatchMediaTab'
import {
  MatchMedalsSection,
  MatchCitationsSection,
  MatchNativeCommendationsSection,
} from './MatchSummaryMedalsAndCitations'
import { buildMatchHeadingStr } from './format'
import { MATCH_VIEW_TEXT, type MatchViewText } from './i18n'
import type { MatchWeaponKill, MatchScoreboardRow, MatchViewRadarSeries } from '@/lib/api/types'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { PageUnavailable } from '@/components/ui/page-unavailable'
import { apiErrorCode } from '@/lib/api/client'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ReactNode } from 'react'

/**
 * DetailSection — titre de section type-1 (catalogue UI d'harmonisation, même
 * format que le Home : titre `text-base font-semibold`) + contenu groupé.
 * Structure l'onglet Détails (dense) en sections lisibles et titrées.
 */
function DetailSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-4">
      <h3 className="text-base font-semibold text-foreground">{title}</h3>
      {children}
    </section>
  )
}

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

function killTypeFallback(me: MatchScoreboardRow | undefined, t: MatchViewText): MatchWeaponKill[] {
  const total = me?.kills ?? 0
  if (!total) return []
  const headshots = me?.headshot_kills ?? 0
  const power = me?.power_weapon_kills ?? 0
  const melee = me?.melee_kills ?? 0
  const other = Math.max(0, total - headshots - power - melee)
  return [
    { weapon_id: 1001, weapon_label: t.labelHeadshots, effective_weapon_id: null, kill_count: headshots },
    { weapon_id: 1002, weapon_label: t.labelPowerWeapon, effective_weapon_id: null, kill_count: power },
    { weapon_id: 1003, weapon_label: t.labelMelee, effective_weapon_id: null, kill_count: melee },
    { weapon_id: 1004, weapon_label: t.labelOtherKills, effective_weapon_id: null, kill_count: other },
  ].filter((w) => w.kill_count > 0)
}

type TabId = 'summary' | 'details'

// Libellés résolus au rendu via MATCH_VIEW_TEXT (GH2-B2 : bilingue).
const TABS: { id: TabId; labelKey: 'tabGeneral' | 'tabDetails' }[] = [
  { id: 'summary', labelKey: 'tabGeneral' },
  { id: 'details', labelKey: 'tabDetails' },
]

export function MatchViewPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }
  const { tab } = useSearch({
    from: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
  })
  const activeTab: TabId = tab ?? 'summary'
  const navigate = useNavigate({ from: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId' })
  const router = useRouter()
  const setActiveTab = (next: TabId) => {
    navigate({ search: (prev) => ({ ...prev, tab: next }), replace: true }).catch(() => {})
  }
  const { data, isPending, isError, error, refetch } = useMatchView(playerSlug, matchId)
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
  const weaponData: MatchWeaponKill[] =
    weaponKills.length > 0 ? weaponKills : killTypeFallback(meRow, t)
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
        {activeTab === 'summary' ? (
          <div className="space-y-4">
            <MatchSummaryCardsSection
              kpis={summary_tab.kpis}
              expectedStats={summary_tab.expected_stats}
              offensiveConversion={meRow?.offensive_conversion ?? null}
              defensiveResistance={meRow?.defensive_resistance ?? null}
              damagePerKill={meRow?.damage_per_kill ?? null}
              damagePerDeath={meRow?.damage_per_death ?? null}
            />
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
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
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <MatchWeaponPieChart weaponKills={weaponData} t={t} />
              {/* Halo 5 : donut « Frags par technique » du viewer (mécaniques natives
                  incl. assassinats + compétences spartiate). Null hors h5 (capability). */}
              <MatchKillTypesDonut me={meRow} t={t} />
              <MatchMedalsSection medals={summary_tab.medals ?? []} t={t} />
              {/* Halo 5 : commendations NATIVES (citations_tab.native_commendations)
                  affichées À LA PLACE des citations dérivées d'Infinite
                  (summary_tab.citations vide pour h5). Un seul bloc « commendations »
                  par titre. */}
              {(citations_tab?.native_commendations?.length ?? 0) > 0 ? (
                <MatchNativeCommendationsSection
                  commendations={citations_tab.native_commendations ?? []}
                  t={t}
                />
              ) : (
                <MatchCitationsSection citations={summary_tab.citations ?? []} t={t} />
              )}
            </div>
            {/* Section Médias gatée sur `media` : masque l'en-tête + le bloc entier
                (pas seulement le contenu) pour un titre sans captures/clips. */}
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
        ) : (
          <>
            {/* §1 — Déroulé du match (lecture chronologique) */}
            <DetailSection title={t.sectionFlow}>
              {/* Faits marquants | Frags cumulés */}
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-[180px_1fr]">
                <MatchImpactBadgesBar badges={impactBadges} scoreboard={scoreboard} t={t} />
                <MatchKDCumulChart
                  events={highlightEvents}
                  badges={impactBadges}
                  scoreboard={scoreboard}
                  meXUID={meXUID}
                  t={t}
                />
              </div>

              {/* Dominance | Cadence des frags */}
              <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
                <MatchTugOfWarChart
                  bins={tugOfWar}
                  events={highlightEvents}
                  scoreboard={scoreboard}
                  meXUID={meXUID}
                  t={t}
                />
                <MatchCadenceChart
                  cadence={combat_tab.cadence}
                  scoreboard={scoreboard}
                  meXUID={meXUID}
                  t={t}
                />
              </div>

              {/* Engagement — remonté ici (avant Frags différentiel cumulé).
                  Gaté sur `engagement` : évite le fetch + la carte placeholder
                  pour un titre sans score d'engagement intra-match. */}
              <FeatureGate capability="engagement">
                <EngagementMatchSection
                  playerSlug={playerSlug}
                  matchId={matchId}
                  granularity="intra"
                  emptyBehavior="placeholder"
                />
              </FeatureGate>
            </DetailSection>

            {/* §2 — Duels & confrontations (face-à-face) */}
            <DetailSection title={t.sectionDuels}>
              {/* Némésis + Souffre-douleur | Antagonistes */}
              <div className="flex flex-col gap-4">
                <MatchNemesisCards
                  nemesis={nemesis}
                  scoreboard={scoreboard}
                  meXUID={meXUID}
                  t={t}
                />
                <MatchAntagonistChart
                  pairs={killerVictim}
                  scoreboard={scoreboard}
                  meXUID={meXUID}
                  t={t}
                />
              </div>

              {/* Frags différentiel cumulé — descendu ici (après Antagonistes) */}
              <MatchFragDiffChart
                events={highlightEvents}
                scoreboard={scoreboard}
                roster={roster}
                pairs={killerVictim}
                meXUID={meXUID}
                t={t}
                friendGamertags={friendGamertags}
              />
            </DetailSection>

            {/* §3 — Tableau des scores (table sortie de son bloc) */}
            <DetailSection title={t.scoreboardTitle}>
              <MatchScoreboard
                rows={scoreboard}
                killerVictim={killerVictim}
                citations={summary_tab.citations ?? []}
                header={header}
                rank={rank}
                t={t}
              />
            </DetailSection>

            {/* §4 — Historique des rencontres (table sortie de son bloc) */}
            <DetailSection title={t.sectionEncounters}>
              <MatchEncountersTable
                rows={team_tab.encounters ?? []}
                locale={locale === 'en' ? 'en' : 'fr'}
                hideCardWrapper
              />
            </DetailSection>
          </>
        )}
      </div>
    </div>
  )
}
