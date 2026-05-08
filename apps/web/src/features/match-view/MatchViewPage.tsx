/** MatchViewPage — détail d'un match (2 onglets : Général, Détails). */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { useSettings } from '@/features/settings/queries'
import { EngagementMatchSection } from '@/features/engagement/EngagementMatchSection'
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
import { MatchMediaTab } from './MatchMediaTab'
import { MatchMedalsSection, MatchCitationsSection } from './MatchSummaryMedalsAndCitations'
import { buildMatchHeadingStr } from './format'
import { MATCH_VIEW_TEXT, type MatchViewText } from './i18n'
import type { MatchWeaponKill, MatchScoreboardRow } from '@/lib/api/types'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
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

const TABS: { id: TabId; label: string }[] = [
  { id: 'summary', label: 'Général' },
  { id: 'details', label: 'Détails' },
]

export function MatchViewPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }
  const [activeTab, setActiveTab] = useState<TabId>('summary')
  const { data, isPending, isError, error, refetch } = useMatchView(playerSlug, matchId)
  const { data: settings } = useSettings()
  const friendGamertags = settings?.friend_gamertags ?? []
  const locale = useAppShellStore((s) => s.locale)
  const t = MATCH_VIEW_TEXT[locale === 'en' ? 'en' : 'fr']

  if (isPending) return null

  if (isError || !data) {
    // 404 strict ou erreur réseau. On garde la barre de navigation pour
    // permettre à l'utilisateur de continuer à naviguer entre les matchs.
    return (
      <div className="flex flex-col">
        <MatchBreadcrumb playerSlug={playerSlug} matchLabel={t.mapUnknown} />
        <MatchNavigationBar playerSlug={playerSlug} matchId={matchId} locale={locale === 'en' ? 'en' : 'fr'} />
        <div className="p-6">
          <Card>
            <CardContent className="py-8 text-center">
              <p className="font-medium text-destructive">
                {locale === 'en'
                  ? 'Match not found or load error.'
                  : 'Match introuvable ou erreur de chargement.'}
              </p>
              {error && (error as { message?: string }).message && (
                <p className="mt-1 text-xs text-muted-foreground font-mono">
                  {(error as { message: string }).message}
                </p>
              )}
              <div className="mt-4">
                <Button variant="outline" size="sm" onClick={() => refetch()}>
                  {locale === 'en' ? 'Retry' : 'Réessayer'}
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }

  const { header, rank, summary_tab, combat_tab, team_tab, media_tab } = data
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

  return (
    <div className="flex flex-col">
      <MatchBreadcrumb playerSlug={playerSlug} matchLabel={breadcrumbLabel} />

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
                {locale === 'en'
                  ? 'This match could not be fully loaded.'
                  : 'Ce match n\'a pas pu être chargé en totalité.'}
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
              {tab.label}
            </Button>
          ))}
        </div>
      </div>

      <div className="p-6 space-y-6">
        {activeTab === 'summary' ? (
          <div className="space-y-4">
            <MatchSummaryCardsSection kpis={summary_tab.kpis} expectedStats={summary_tab.expected_stats} />
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
                radar={data.radar}
                meXUID={meXUID}
                t={t}
              />
            </div>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <MatchWeaponPieChart weaponKills={weaponData} t={t} />
              <MatchMedalsSection medals={summary_tab.medals ?? []} t={t} />
              <MatchCitationsSection citations={summary_tab.citations ?? []} t={t} />
            </div>
            <div className="rounded-lg border border-border bg-card">
              <div className="border-b border-border px-3 py-2 text-sm font-medium">{t.sectionMedia}</div>
              <div className="p-3">
                <MatchMediaTab
                  items={media_tab.media_items}
                  playerSlug={playerSlug}
                  matchId={matchId}
                  locale={locale === 'en' ? 'en' : 'fr'}
                />
              </div>
            </div>
          </div>
        ) : (
          <>
            {/* Faits marquants | Frags cumulés */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-[180px_1fr]">
              <MatchImpactBadgesBar badges={impactBadges} scoreboard={scoreboard} />
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

            <MatchFragDiffChart
              events={highlightEvents}
              scoreboard={scoreboard}
              roster={roster}
              pairs={killerVictim}
              meXUID={meXUID}
              t={t}
              friendGamertags={friendGamertags}
            />

            {/* Antagonistes | Némésis + Souffre-douleur */}
            <div className="grid grid-cols-1 gap-4 lg:grid-cols-[2fr_1fr] items-start">
              <MatchAntagonistChart
                pairs={killerVictim}
                scoreboard={scoreboard}
                roster={roster}
                meXUID={meXUID}
                t={t}
                friendGamertags={friendGamertags}
              />
              <MatchNemesisCards
                nemesis={nemesis}
                scoreboard={scoreboard}
                meXUID={meXUID}
                t={t}
              />
            </div>

            <EngagementMatchSection
              playerSlug={playerSlug}
              matchId={matchId}
              granularity="intra"
            />

            <MatchScoreboard
              rows={scoreboard}
              killerVictim={killerVictim}
              citations={summary_tab.citations ?? []}
              header={header}
              rank={rank}
              t={t}
            />

            <MatchEncountersTable
              rows={team_tab.encounters ?? []}
              locale={locale === 'en' ? 'en' : 'fr'}
            />
          </>
        )}
      </div>
    </div>
  )
}
