/**
 * MatchViewPage — détail d'un match (5 onglets).
 *
 * P8.4 (revue 2026-04-29) : MatchBreadcrumb + MatchNavigation extraits dans
 * MatchHeader.tsx ; helpers ChartSeries dans _chartSeries.ts. Ce fichier ne
 * porte plus que l'orchestrateur d'onglets.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { useMatchView } from './queries'
import { EngagementMatchSection } from '@/features/engagement/EngagementMatchSection'

import { MatchScoreboard } from './MatchScoreboard'
import { ExpectedCardsSection, MatchRankBadge, KdIndicatorCard } from './MatchStatCards'
import { MatchBreadcrumb, MatchNavigation } from './MatchHeader'
import { kdTimelineSeries, tugOfWarSeries } from './_chartSeries'
import { useSetMatchExclusion } from '@/features/match-history/queries'
import { queryKeys } from '@/lib/query/keys'
import { PrivacyBanner } from '@/components/ui/privacy-banner'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

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
  const { data, isLoading, isError, refetch } = useMatchView(playerSlug, matchId)
  const queryClient = useQueryClient()
  const excludeMutation = useSetMatchExclusion(playerSlug)
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key

  if (isLoading) return null

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

  const { header, rank, summary_tab, combat_tab, team_tab, media_tab, citations_tab } = data
  const matchLabel = `${header.map_ui} — ${header.mode_ui}`

  return (
    <div className="flex flex-col">
      <MatchBreadcrumb playerSlug={playerSlug} matchLabel={matchLabel} />
      <div className="flex items-center justify-between px-6 pt-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
            {header.map_ui} — {header.mode_ui}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">{header.start_time_label}</p>
        </div>
        <MatchNavigation playerSlug={playerSlug} matchId={matchId} />
      </div>

      {/* Sprint 54-B : avertissement privacy */}
      {data.privacy_warning && (
        <div className="px-6 pt-4">
          <PrivacyBanner warning={data.privacy_warning} />
        </div>
      )}

      {/* Header match */}
      <div className="border-b bg-background px-6 py-4">
        <div className="flex flex-wrap items-center gap-4">
          <span
            className="text-xl font-bold"
            style={{ color: header.outcome_color }}
          >
            {header.outcome_label}
          </span>
          <span className="text-sm text-muted-foreground">{header.score_label}</span>
          <Badge variant="outline">{header.playlist_label}</Badge>
          {rank.rating_type !== 'none' && rank.tier_label && (
            <Badge className="bg-primary/10 text-primary">
              {rank.rating_type} · {rank.tier_label}
              {rank.numeric_value != null && ` (${rank.numeric_value.toFixed(0)})`}
            </Badge>
          )}
          <span className="ml-auto text-sm font-medium" style={{ color: header.performance_color ?? undefined }}>
            {header.performance_display}
          </span>
          <Button
            variant="ghost"
            size="sm"
            className={header.is_excluded ? 'text-muted-foreground' : 'text-muted-foreground hover:text-destructive'}
            loading={excludeMutation.isPending}
            onClick={() => {
              excludeMutation.mutate(
                { matchId, excluded: !header.is_excluded },
                {
                  onSuccess: () => {
                    void queryClient.invalidateQueries({
                      queryKey: queryKeys.matchView(playerSlug, matchId),
                    })
                  },
                },
              )
            }}
          >
            {header.is_excluded ? '↩ Réactiver' : 'Marquer comme non pertinent ⊘'}
          </Button>
        </div>
      </div>

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
        {/* Onglet Résumé */}
        {activeTab === 'summary' && (
          <div className="space-y-6">
            {/* KPI grid principale */}
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
              {[
                { label: labelOf('kills'), value: summary_tab.kpis.kills },
                { label: labelOf('deaths'), value: summary_tab.kpis.deaths },
                { label: labelOf('assists'), value: summary_tab.kpis.assists },
                { label: labelOf('kda'), value: summary_tab.kpis.kda?.toFixed(2) },
                { label: labelOf('damage_dealt'), value: summary_tab.kpis.damage_dealt?.toFixed(0) },
                { label: 'Vie moy.', value: summary_tab.kpis.average_life },
              ].map((kpi) => (
                <Card key={kpi.label}>
                  <CardContent className="py-3 text-center">
                    <p className="text-xs text-muted-foreground">{kpi.label}</p>
                    <p className="text-lg font-bold text-foreground">{kpi.value ?? '-'}</p>
                  </CardContent>
                </Card>
              ))}
            </div>

            {/* C3 — Expected vs Actual */}
            <ExpectedCardsSection
              kpis={summary_tab.kpis}
              expectedStats={summary_tab.expected_stats}
            />

            {/* C4 — Rang après match + C5 — K/D vs nemesis */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <MatchRankBadge rank={rank} hadBotTeammate={header.had_bot_teammate} />
              <KdIndicatorCard nemesis={team_tab.nemesis[0] ?? null} />
            </div>

            {summary_tab.medals.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Médailles</p>
                  <div className="flex flex-wrap gap-2">
                    {summary_tab.medals.map((m) => (
                      <Badge key={m.medal_name_id} variant="secondary" title={m.description ?? undefined}>
                        {m.name} × {m.count}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {summary_tab.citations.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Citations</p>
                  <div className="flex flex-wrap gap-2">
                    {summary_tab.citations.map((c) => (
                      <Badge
                        key={c.key}
                        style={{ backgroundColor: c.color ?? undefined, color: '#fff' }}
                      >
                        {c.label}
                        {c.value != null && ` · ${c.value}`}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {/* Onglet Combat */}
        {activeTab === 'combat' && (
          <div className="space-y-6">
            {combat_tab.weapon_kills.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Kills par arme</p>
                  <div className="space-y-1">
                    {combat_tab.weapon_kills.map((w) => (
                      <div key={w.weapon_id} className="flex items-center justify-between text-sm">
                        <span className="text-foreground">{w.weapon_label}</span>
                        <span className="font-semibold text-primary">{w.kill_count}</span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Impact badges */}
            {combat_tab.impact_badges.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Moments clés</p>
                  <div className="flex flex-wrap gap-2">
                    {combat_tab.impact_badges.map((b) => (
                      <Badge key={b.key} variant="outline" className="text-xs">
                        {b.label}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* K/D Timeline (chunk MV5 — migration Recharts -> ECharts wrapper S10) */}
            {combat_tab.kd_timeline.length > 1 && (
              <Card>
                <CardContent className="py-4">
                  <TimeseriesLineChart
                    title="Timeline K/D"
                    height={200}
                    xAxisType="value"
                    timeAxis={false}
                    outcomeMarkers={false}
                    series={kdTimelineSeries(combat_tab.kd_timeline, labelOf)}
                  />
                </CardContent>
              </Card>
            )}

            {/* Tug-of-War (chunk MV5) */}
            {combat_tab.tug_of_war.length > 1 && (
              <Card>
                <CardContent className="py-4">
                  <TimeseriesLineChart
                    title="Tir à la corde (kills nets)"
                    height={160}
                    xAxisType="value"
                    timeAxis={false}
                    outcomeMarkers={false}
                    series={tugOfWarSeries(combat_tab.tug_of_war)}
                  />
                </CardContent>
              </Card>
            )}

            {/* Nemesis duels */}
            {combat_tab.nemesis_duels.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Duels nemesis</p>
                  <div className="space-y-1">
                    {combat_tab.nemesis_duels.map((n) => (
                      <div key={n.xuid} className="flex items-center justify-between text-sm">
                        <span className="text-foreground">{n.gamertag}</span>
                        <span className="text-xs text-muted-foreground">
                          {n.killed_me} kills reçus · {n.i_killed} kills rendus
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {/* Onglet Équipe */}
        {activeTab === 'team' && (
          <div className="space-y-6">
            <EngagementMatchSection
              playerSlug={playerSlug}
              matchId={matchId}
              granularity="intra"
            />
            <MatchScoreboard
              rows={team_tab.scoreboard}
              weaponKills={combat_tab.weapon_kills}
              medals={summary_tab.medals}
              citations={summary_tab.citations}
            />

            {team_tab.nemesis.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Nemesis</p>
                  <div className="space-y-1">
                    {team_tab.nemesis.map((n) => (
                      <div key={n.xuid} className="flex items-center justify-between text-sm">
                        <span className="text-foreground">{n.gamertag}</span>
                        <span className="text-xs text-muted-foreground">
                          {n.killed_me} kills reçus · {n.i_killed} kills rendus
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {team_tab.encounters && team_tab.encounters.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Joueurs fréquents</p>
                  <div className="space-y-1">
                    {team_tab.encounters.map((e) => (
                      <div key={e.xuid} className="flex items-center justify-between text-sm">
                        <span className="text-foreground">{e.gamertag}</span>
                        <span className="text-xs text-muted-foreground">
                          {e.count_together} matchs ensemble
                          {e.is_ally ? ' · Coéquipier' : ' · Adversaire'}
                        </span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}

        {/* Onglet Médias */}
        {activeTab === 'media' && (
          <div className="space-y-4">
            {media_tab.media_items.length === 0 ? (
              <p className="text-sm text-muted-foreground">Aucun média associé à ce match.</p>
            ) : (
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
                {media_tab.media_items.map((item) => (
                  <Card key={item.file_id}>
                    <CardContent className="p-3">
                      {item.thumbnail_url ? (
                        <img
                          src={item.thumbnail_url}
                          alt={item.file_name}
                          className="mb-2 h-24 w-full rounded object-cover"
                        />
                      ) : (
                        <div className="mb-2 flex h-24 items-center justify-center rounded bg-muted">
                          <span className="text-xs text-muted-foreground">Aperçu indisponible</span>
                        </div>
                      )}
                      <p className="truncate text-xs text-foreground">{item.file_name}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Onglet Citations */}
        {activeTab === 'citations' && (
          <div className="space-y-6">
            {citations_tab.commendations.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Commendations</p>
                  <div className="flex flex-wrap gap-2">
                    {citations_tab.commendations.map((c) => (
                      <Badge
                        key={c.key}
                        style={{ backgroundColor: c.color ?? undefined, color: '#fff' }}
                      >
                        {c.label}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
            {citations_tab.medals.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-foreground">Médailles contextuelles</p>
                  <div className="flex flex-wrap gap-2">
                    {citations_tab.medals.map((m) => (
                      <Badge key={m.medal_name_id} variant="secondary">
                        {m.name} × {m.count}
                      </Badge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
            {citations_tab.commendations.length === 0 && citations_tab.medals.length === 0 && (
              <p className="text-sm text-muted-foreground">Aucune citation pour ce match.</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
