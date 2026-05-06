/**
 * MatchViewPage — détail d'un match (3 onglets : Résumé, Combat, Équipe).
 *
 * Refonte 2026-05-02 :
 *  - Onglet Combat : chart match_view.09 (K/D cumulés) avec annotation badges.
 *  - Médias : section dédiée en bas de l'onglet Résumé (pas d'onglet séparé).
 */
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

type TabId = 'summary' | 'combat' | 'team'

const TABS: { id: TabId; label: string }[] = [
  { id: 'summary', label: 'Résumé' },
  { id: 'combat', label: 'Combat' },
  { id: 'team', label: 'Équipe' },
]

export function MatchViewPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }
  const [activeTab, setActiveTab] = useState<TabId>('summary')
  const { data, isPending, isError, refetch } = useMatchView(playerSlug, matchId)
  const { data: settings } = useSettings()
  const friendGamertags = settings?.friend_gamertags ?? []
  const locale = useAppShellStore((s) => s.locale)

  if (isPending) return null

  if (isError || !data) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Match introuvable ou erreur de chargement.</p>
            <div className="mt-4">
              <Button variant="outline" size="sm" onClick={() => refetch()}>
                Réessayer
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  const { header, rank, summary_tab, combat_tab, team_tab, media_tab } = data
  const t = MATCH_VIEW_TEXT[locale === 'en' ? 'en' : 'fr']
  const matchLabel = buildMatchHeadingStr(header.map_ui, header.mode_ui, locale)
  // Le breadcrumb ajoute la date pour distinguer plusieurs matchs sur la même map/mode
  const breadcrumbLabel = header.start_time_label
    ? `${matchLabel} · ${header.start_time_label}`
    : matchLabel
  const meRow = team_tab.scoreboard.find((r) => r.is_me)
  const meXUID = meRow?.xuid ?? null
  const weaponData: MatchWeaponKill[] =
    combat_tab.weapon_kills.length > 0
      ? combat_tab.weapon_kills
      : killTypeFallback(meRow, t)

  return (
    <div className="flex flex-col">
      <MatchBreadcrumb playerSlug={playerSlug} matchLabel={breadcrumbLabel} />

      {/* Sprint 54-B : avertissement privacy */}
      {data.privacy_warning && (
        <div className="px-6 pt-4">
          <PrivacyBanner warning={data.privacy_warning} />
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
        ) : activeTab === 'combat' ? (
          <>
            <MatchImpactBadgesBar
              badges={combat_tab.impact_badges}
              scoreboard={team_tab.scoreboard}
            />

            <MatchKDCumulChart points={combat_tab.kd_timeline} t={t} />

            <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
              <MatchTugOfWarChart bins={combat_tab.tug_of_war} t={t} />
              <MatchCadenceChart
                cadence={combat_tab.cadence}
                scoreboard={team_tab.scoreboard}
                meXUID={meXUID}
                t={t}
              />
            </div>

            <MatchNemesisCards
              nemesis={team_tab.nemesis}
              scoreboard={team_tab.scoreboard}
              meXUID={meXUID}
              t={t}
            />

            <MatchFragDiffChart
              events={combat_tab.highlight_events}
              scoreboard={team_tab.scoreboard}
              roster={team_tab.roster}
              pairs={combat_tab.killer_victim}
              meXUID={meXUID}
              friendGamertags={friendGamertags}
            />

            <EngagementMatchSection
              playerSlug={playerSlug}
              matchId={matchId}
              granularity="intra"
            />

            <MatchAntagonistChart
              pairs={combat_tab.killer_victim}
              scoreboard={team_tab.scoreboard}
              roster={team_tab.roster}
              meXUID={meXUID}
              friendGamertags={friendGamertags}
            />
          </>
        ) : (
          <Card>
            <CardContent className="py-12 text-center">
              <p className="text-sm text-muted-foreground">
                Onglet « {TABS.find((t) => t.id === activeTab)?.label} » — contenu à refaire.
              </p>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
