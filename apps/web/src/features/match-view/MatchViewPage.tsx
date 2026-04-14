/**
 * MatchViewPage — détail d'un match (5 onglets).
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useMatchView } from './queries'
import { MatchScoreboard } from './MatchScoreboard'
import { ExpectedCardsSection, MatchRankBadge, KdIndicatorCard } from './MatchStatCards'

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

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement du match…" />
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-red-600">Match introuvable ou erreur de chargement.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-purple-600 underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  const { header, rank, summary_tab, combat_tab, team_tab, media_tab, citations_tab } = data

  return (
    <div className="flex flex-col">
      <PageHeader
        title={`${header.map_ui} — ${header.mode_ui}`}
        subtitle={header.start_time_label}
      />

      {/* Header match */}
      <div className="border-b bg-white px-6 py-4">
        <div className="flex flex-wrap items-center gap-4">
          <span
            className="text-xl font-bold"
            style={{ color: header.outcome_color }}
          >
            {header.outcome_label}
          </span>
          <span className="text-sm text-gray-600">{header.score_label}</span>
          <Badge variant="outline">{header.playlist_label}</Badge>
          {rank.rating_type !== 'none' && rank.tier_label && (
            <Badge className="bg-purple-100 text-purple-800">
              {rank.rating_type} · {rank.tier_label}
              {rank.numeric_value != null && ` (${rank.numeric_value.toFixed(0)})`}
            </Badge>
          )}
          <span className="ml-auto text-sm font-medium" style={{ color: header.performance_color ?? undefined }}>
            {header.performance_display}
          </span>
        </div>
      </div>

      {/* Onglets */}
      <div className="flex gap-0 border-b bg-white px-6">
        {TABS.map((tab) => (
          <Button
            key={tab.id}
            variant="ghost"
            size="sm"
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-none border-b-2 px-4 py-3 text-sm ${
              activeTab === tab.id
                ? 'border-purple-600 font-semibold text-purple-700'
                : 'border-transparent text-gray-500 hover:text-gray-800'
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
                { label: 'Kills', value: summary_tab.kpis.kills },
                { label: 'Deaths', value: summary_tab.kpis.deaths },
                { label: 'Assists', value: summary_tab.kpis.assists },
                { label: 'KDA', value: summary_tab.kpis.kda?.toFixed(2) },
                { label: 'Damage', value: summary_tab.kpis.damage_dealt?.toFixed(0) },
                { label: 'Vie moy.', value: summary_tab.kpis.average_life },
              ].map((kpi) => (
                <Card key={kpi.label}>
                  <CardContent className="py-3 text-center">
                    <p className="text-xs text-gray-500">{kpi.label}</p>
                    <p className="text-lg font-bold text-gray-900">{kpi.value ?? '-'}</p>
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
                  <p className="mb-3 text-sm font-semibold text-gray-700">Médailles</p>
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
                  <p className="mb-3 text-sm font-semibold text-gray-700">Citations</p>
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
                  <p className="mb-3 text-sm font-semibold text-gray-700">Kills par arme</p>
                  <div className="space-y-1">
                    {combat_tab.weapon_kills.map((w) => (
                      <div key={w.weapon_id} className="flex items-center justify-between text-sm">
                        <span className="text-gray-700">{w.weapon_label}</span>
                        <span className="font-semibold text-purple-700">{w.kill_count}</span>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
            {combat_tab.charts.map((fig, i) => (
              <Card key={i}>
                <CardContent className="py-4">
                  <PlotlyChart figure={fig} />
                </CardContent>
              </Card>
            ))}
          </div>
        )}

        {/* Onglet Équipe */}
        {activeTab === 'team' && (
          <div className="space-y-6">
            <MatchScoreboard
              rows={team_tab.scoreboard}
              weaponKills={combat_tab.weapon_kills}
              medals={summary_tab.medals}
              citations={summary_tab.citations}
            />

            {team_tab.nemesis.length > 0 && (
              <Card>
                <CardContent className="py-4">
                  <p className="mb-3 text-sm font-semibold text-gray-700">Nemesis</p>
                  <div className="space-y-1">
                    {team_tab.nemesis.map((n) => (
                      <div key={n.xuid} className="flex items-center justify-between text-sm">
                        <span className="text-gray-700">{n.gamertag}</span>
                        <span className="text-xs text-gray-500">
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

        {/* Onglet Médias */}
        {activeTab === 'media' && (
          <div className="space-y-4">
            {media_tab.media_items.length === 0 ? (
              <p className="text-sm text-gray-500">Aucun média associé à ce match.</p>
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
                        <div className="mb-2 flex h-24 items-center justify-center rounded bg-gray-100">
                          <span className="text-xs text-gray-400">Aperçu indisponible</span>
                        </div>
                      )}
                      <p className="truncate text-xs text-gray-700">{item.file_name}</p>
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
                  <p className="mb-3 text-sm font-semibold text-gray-700">Commendations</p>
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
                  <p className="mb-3 text-sm font-semibold text-gray-700">Médailles contextuelles</p>
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
              <p className="text-sm text-gray-500">Aucune citation pour ce match.</p>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
