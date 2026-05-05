/**
 * MatchViewPage — détail d'un match (5 onglets).
 *
 * Refonte 2026-05-02 :
 *  - Bandeau "Faits marquants" (badges d'impact + timing) au-dessus des onglets.
 *  - Onglet Combat : chart match_view.09 (K/D cumulés) avec annotation badges.
 *  - Autres onglets : placeholders en attendant la refonte.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { useMatchView } from './queries'
import { MatchBreadcrumb, MatchNavigationBar, MatchHeaderCard } from './MatchHeader'
import { MatchImpactBadgesBar } from './MatchImpactBadgesBar'
import { MatchTugOfWarChart } from './MatchTugOfWarChart'
import { MatchCadenceChart } from './MatchCadenceChart'
import { MatchAntagonistChart } from './MatchAntagonistChart'
import { MatchFragDiffChart } from './MatchFragDiffChart'
import { kdTimelineSeries } from './_chartSeries'
import { buildMatchHeadingStr } from './format'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { useAppShellStore } from '@/stores/appShellStore'

type TabId = 'summary' | 'combat' | 'team' | 'media' | 'citations'

const TABS: { id: TabId; label: string }[] = [
  { id: 'summary', label: 'Résumé' },
  { id: 'combat', label: 'Combat' },
  { id: 'team', label: 'Équipe' },
  { id: 'media', label: 'Médias' },
  { id: 'citations', label: 'Citations' },
]

export function MatchViewPage() {
  const { playerSlug, matchId } = useParams({ strict: false }) as {
    playerSlug: string
    matchId: string
  }
  const [activeTab, setActiveTab] = useState<TabId>('summary')
  const { data, isPending, isError, refetch } = useMatchView(playerSlug, matchId)
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

  const { header, rank, combat_tab, team_tab } = data
  const matchLabel = buildMatchHeadingStr(header.map_ui, header.mode_ui, locale)
  // Le breadcrumb ajoute la date pour distinguer plusieurs matchs sur la même map/mode
  const breadcrumbLabel = header.start_time_label
    ? `${matchLabel} · ${header.start_time_label}`
    : matchLabel
  const labelOf = (key: string, fallback: string) => fallback ?? key
  const kdSeries = kdTimelineSeries(combat_tab.kd_timeline, labelOf)
  const meXUID = team_tab.scoreboard.find((r) => r.is_me)?.xuid ?? null

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

      {/* Faits marquants — badges d'impact + horodatage */}
      <MatchImpactBadgesBar
        badges={combat_tab.impact_badges}
        scoreboard={team_tab.scoreboard}
      />

      {/* Onglets */}
      <div className="flex gap-0 border-b bg-background px-6">
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

      <div className="p-6 space-y-6">
        {activeTab === 'combat' ? (
          <>
            {combat_tab.kd_timeline.length > 1 ? (
              <Card>
                <CardContent className="py-4">
                  <TimeseriesLineChart
                    title="K/D cumulés du match"
                    height={320}
                    xAxisType="value"
                    timeAxis={false}
                    outcomeMarkers={false}
                    series={kdSeries}
                  />
                </CardContent>
              </Card>
            ) : (
              <Card>
                <CardContent className="py-12 text-center">
                  <p className="text-sm text-muted-foreground">
                    Pas assez d'événements pour tracer la timeline K/D de ce match.
                  </p>
                </CardContent>
              </Card>
            )}

            <MatchTugOfWarChart
              bins={combat_tab.tug_of_war}
              events={combat_tab.highlight_events}
              scoreboard={team_tab.scoreboard}
              meXUID={meXUID}
            />

            <MatchFragDiffChart
              events={combat_tab.highlight_events}
              scoreboard={team_tab.scoreboard}
              meXUID={meXUID}
            />

            <MatchCadenceChart
              cadence={combat_tab.cadence}
              scoreboard={team_tab.scoreboard}
            />

            <MatchAntagonistChart pairs={combat_tab.killer_victim} />
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
