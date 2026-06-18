/**
 * SessionComparePage — comparaison A/B de deux sessions.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { useSessionComparePage } from './queries'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import type {
  SessionCompareEntry,
  SessionCompareMapRow,
  SessionCompareModeRow,
  SessionCompareMetricRow,
} from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { sessionManifest, type SessionManifestKey } from '@/lib/i18n/generated/session'
import { useAppShellStore } from '@/stores/appShellStore'
import { SessionOutcomesDonut } from './SessionOutcomesDonut'
import { CombatYieldBar } from '@/components/ui/combat-yield-bar'
import { SessionCompareRadar } from './SessionCompareRadar'
import { SessionCompareBarMetrics } from './SessionCompareBarMetrics'
import { SessionCompareCumulative } from './SessionCompareCumulative'
import { SessionCompareKDProgression } from './SessionCompareKDProgression'
import { SessionCompareSkillHeader } from './SessionCompareSkillHeader'
import { SessionCompareMMR } from './SessionCompareMMR'
import { SessionCompareParticipation } from './SessionCompareParticipation'
import { SessionCompareMatchHistory } from './SessionCompareMatchHistory'
import { SessionCompareKillsDonut } from './SessionCompareKillsDonut'
import { SessionCompareOutcomeTape } from './SessionCompareOutcomeTape'
import { SessionComparePerfProgression } from './SessionComparePerfProgression'
import { SessionCompareSkillProgression } from './SessionCompareSkillProgression'
import { SessionCompareOCDR } from './SessionCompareOCDR'
import { SessionCompareEngagement } from './SessionCompareEngagement'

function useSessionT() {
  const locale = useAppShellStore((s) => s.locale)
  return (key: SessionManifestKey, values?: Record<string, string | number>) =>
    formatMessage(sessionManifest, key, locale, values)
}

function SessionCard({
  label,
  entry,
  side,
}: {
  label: string
  entry: SessionCompareEntry | null
  side: 'A' | 'B'
}) {
  const color = side === 'A' ? 'text-compare-a' : 'text-compare-b'
  const bg = side === 'A' ? 'bg-compare-a/10 border-compare-a' : 'bg-compare-b/10 border-compare-b'
  const t = useSessionT()

  return (
    <Card className={`border ${bg}`}>
      <CardHeader className="pb-2">
        <CardTitle className={`text-sm ${color}`}>
          {t('session.compare.session_card_title', { side, suffix: label ? ` — ${label}` : '' })}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 text-xs text-foreground">
        {entry ? (
          <>
            <p>
              <span className="font-medium">{t('session.compare.label_matches')}</span> {entry.total_matches}
            </p>
            <p>
              <span className="font-medium">{t('session.compare.label_wins')}</span> {entry.wins} · {t('session.compare.label_losses')} {entry.losses}
            </p>
            {entry.kda != null && (
              <p>
                <span className="font-medium">{t('session.compare.label_kda')}</span> {entry.kda.toFixed(2)}
              </p>
            )}
            {entry.performance_score != null && (
              <p>
                <span className="font-medium">{t('session.compare.label_perf_score')}</span>{' '}
                {entry.performance_score.toFixed(1)}
              </p>
            )}
            {entry.dominant_category && (
              <Badge variant="secondary" className="mt-1">
                {entry.dominant_category}
              </Badge>
            )}
            {(entry.avg_oc != null || entry.avg_dr != null) && (
              <div className="mt-2 flex items-center gap-2">
                <span className="text-muted-foreground">OC/DR</span>
                <CombatYieldBar
                  offensiveConversion={entry.avg_oc}
                  defensiveResistance={entry.avg_dr}
                />
              </div>
            )}
          </>
        ) : (
          <p className="text-muted-foreground italic">{t('session.compare.not_selected')}</p>
        )}
      </CardContent>
    </Card>
  )
}

function MetricRow({ row }: { row: SessionCompareMetricRow }) {
  const winnerColor =
    row.winner === 'a'
      ? 'text-compare-a font-semibold'
      : row.winner === 'b'
        ? 'text-compare-b font-semibold'
        : 'text-foreground'

  return (
    <tr className="border-b last:border-0 text-sm">
      <td className="py-2 pr-4 text-muted-foreground">{row.label}</td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'a' ? 'text-compare-a font-semibold' : 'text-foreground'}`}>
        {row.value_a}
      </td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'b' ? 'text-compare-b font-semibold' : 'text-foreground'}`}>
        {row.value_b}
      </td>
      <td className={`py-2 text-right text-xs ${winnerColor}`}>
        {row.delta ?? '—'}
      </td>
    </tr>
  )
}

function MapTableRow({ row }: { row: SessionCompareMapRow }) {
  return (
    <tr className="border-b last:border-0 text-sm hover:bg-muted/30">
      <td className="py-1.5 pr-4 text-foreground font-medium">{row.map_name}</td>
      <td className="py-1.5 pr-4 text-center text-compare-a">{row.a_matches}</td>
      <td className="py-1.5 pr-4 text-center text-compare-a">{row.a_wins}V {row.a_losses}D</td>
      <td className="py-1.5 pr-4 text-center text-compare-b">{row.b_matches}</td>
      <td className="py-1.5 text-center text-compare-b">{row.b_wins}V {row.b_losses}D</td>
    </tr>
  )
}

function ModeTableRow({ row }: { row: SessionCompareModeRow }) {
  return (
    <tr className="border-b last:border-0 text-sm hover:bg-muted/30">
      <td className="py-1.5 pr-4 text-foreground font-medium">{row.mode_name}</td>
      <td className="py-1.5 pr-4 text-center text-compare-a">{row.a_matches}</td>
      <td className="py-1.5 pr-4 text-center text-compare-a">{row.a_wins}</td>
      <td className="py-1.5 pr-4 text-center text-compare-b">{row.b_matches}</td>
      <td className="py-1.5 text-center text-compare-b">{row.b_wins}</td>
    </tr>
  )
}

export function SessionComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const t = useSessionT()

  const [sessionA, setSessionA] = useState('')
  const [sessionB, setSessionB] = useState('')

  const filterContext = useSoloFilterStore((s) => s.filterContext)
  const filterContextHash = useSoloFilterStore((s) => s.filterContextHash)

  const { data, isLoading, isError, refetch } = useSessionComparePage(
    playerSlug,
    {
      filters: filterContext,
      session_a: sessionA || null,
      session_b: sessionB || null,
    },
    filterContextHash,
    sessionA,
    sessionB,
  )

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <Spinner size="lg" label={t('session.compare.loading')} />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('session.compare.load_error')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('session.errors.retry')}
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title={t('session.compare.empty_title')}
          description={t('session.compare.empty_description')}
          actionLabel={t('session.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  // Le contrat OpenAPI déclare ces collections nullable (le Go peut renvoyer null) ;
  // on les normalise en tableaux pour les itérations et les passages aux sous-composants.
  const availableSessions = data.available_sessions ?? []
  const metrics = data.metrics ?? []
  const mapsTable = data.maps_table ?? []
  const modesTable = data.modes_table ?? []
  const hasAvailableSessions = availableSessions.length > 0
  const hasComparisonSelection = Boolean(sessionA && sessionB)

  const labelA = t('session.compare.session_card_title', {
    side: 'A',
    suffix: data.session_a?.session_label ? ` — ${data.session_a.session_label}` : '',
  })
  const labelB = t('session.compare.session_card_title', {
    side: 'B',
    suffix: data.session_b?.session_label ? ` — ${data.session_b.session_label}` : '',
  })
  const chartEmpty = t('session.compare.chart_empty')

  return (
    <div className="space-y-6 p-6">
        {hasAvailableSessions ? (
          <>
            {/* Sélecteurs A / B */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.compare.session_a_label')}</label>
                <select
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                  value={sessionA}
                  onChange={(e) => setSessionA(e.target.value)}
                >
                  <option value="">{t('session.compare.placeholder_select')}</option>
                  {availableSessions.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.compare.session_b_label')}</label>
                <select
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
                  value={sessionB}
                  onChange={(e) => setSessionB(e.target.value)}
                >
                  <option value="">{t('session.compare.placeholder_select')}</option>
                  {availableSessions.map((s) => (
                    <option key={s} value={s}>{s}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Cards résumé */}
            <div className="grid gap-4 sm:grid-cols-2">
              <SessionCard label={data.session_a?.session_label ?? ''} entry={data.session_a} side="A" />
              <SessionCard label={data.session_b?.session_label ?? ''} entry={data.session_b} side="B" />
            </div>

            {hasComparisonSelection ? (
              <>
                {/* Skill Rating (chart 01) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.skill_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    <SessionCompareSkillHeader
                      sessionA={data.session_a}
                      sessionB={data.session_b}
                      labels={{
                        title: t('session.compare.skill_title'),
                        deltaLabel: t('session.compare.skill_delta_label'),
                        empty: t('session.compare.skill_empty'),
                      }}
                    />
                  </CardContent>
                </Card>

                {/* Outcomes distribution (chart 03) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.outcomes_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.session_a || data.session_b ? (
                      <SessionOutcomesDonut
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          sessionA: labelA,
                          sessionB: labelB,
                          wins: t('session.compare.donut.wins'),
                          losses: t('session.compare.donut.losses'),
                          other: t('session.compare.donut.other'),
                        }}
                      />
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.outcomes_empty_title')}
                        description={t('session.compare.outcomes_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                {/* Répartition K/D/A (côte à côte) + enchaînement résultats */}
                <div className="grid gap-6 sm:grid-cols-2">
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">{t('session.compare.kills_donut_title')}</CardTitle>
                    </CardHeader>
                    <CardContent className="pb-4">
                      <SessionCompareKillsDonut
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.kills_donut_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: t('session.compare.kills_donut_empty'),
                        }}
                      />
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">{t('session.compare.outcome_tape_title')}</CardTitle>
                    </CardHeader>
                    <CardContent className="pb-4">
                      <SessionCompareOutcomeTape
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.outcome_tape_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: t('session.compare.outcome_tape_empty'),
                        }}
                      />
                    </CardContent>
                  </Card>
                </div>

                {/* Match highlights (chart 04) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.highlight_title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {data.session_a?.best_match || data.session_b?.best_match ? (
                      <div className="grid gap-4 sm:grid-cols-2">
                        {[
                          { label: t('session.compare.highlight_best'), entries: [data.session_a?.best_match, data.session_b?.best_match] as const },
                          { label: t('session.compare.highlight_worst'), entries: [data.session_a?.worst_match, data.session_b?.worst_match] as const },
                        ].map(({ label, entries }) => (
                          <div key={label}>
                            <p className="text-xs font-medium text-muted-foreground mb-2">{label}</p>
                            <div className="grid grid-cols-2 gap-2">
                              {entries.map((match, i) => {
                                const side = i === 0 ? 'a' : 'b'
                                const color = side === 'a' ? 'text-compare-a border-compare-a/30' : 'text-compare-b border-compare-b/30'
                                return match ? (
                                  <div key={side} className={`rounded border p-2 text-xs space-y-0.5 ${color}`}>
                                    <p className="font-medium truncate">{match.pair_name || match.playlist_name || '—'}</p>
                                    <p className="text-muted-foreground">
                                      {t('session.compare.highlight_kills')} {match.kills}/{match.deaths}/{match.assists}
                                    </p>
                                    {match.performance_score != null && (
                                      <p>{t('session.compare.highlight_score')} {match.performance_score.toFixed(1)}</p>
                                    )}
                                  </div>
                                ) : (
                                  <div key={side} className="rounded border p-2 text-xs text-muted-foreground italic">—</div>
                                )
                              })}
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.highlight_empty')}
                        description={t('session.compare.chart_empty')}
                      />
                    )}
                  </CardContent>
                </Card>

                {/* Profil global radar (chart 07) */}
                <div className="grid gap-6 sm:grid-cols-2">
                  <Card>
                    <CardContent className="pt-4">
                      <SessionCompareRadar
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        metrics={metrics}
                        labels={{
                          title: t('session.compare.radar_title'),
                          axisKD: t('session.compare.radar_axis_kd'),
                          axisWinRate: t('session.compare.radar_axis_winrate'),
                          axisAccuracy: t('session.compare.radar_axis_accuracy'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: chartEmpty,
                        }}
                        height={300}
                      />
                    </CardContent>
                  </Card>

                  {/* Métriques normalisées bar (chart 08) */}
                  <Card>
                    <CardContent className="pt-4">
                      <SessionCompareBarMetrics
                        metrics={metrics}
                        labels={{
                          title: t('session.compare.bar_metrics_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: chartEmpty,
                          catKD: t('session.compare.radar_axis_kd'),
                          catWinRate: t('session.compare.radar_axis_winrate'),
                          catAccuracy: t('session.compare.radar_axis_accuracy'),
                        }}
                        height={300}
                      />
                    </CardContent>
                  </Card>
                </div>

                {/* Résumé delta cards */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.summary_title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {metrics.length > 0 && data.session_a && data.session_b ? (
                      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                        {(['kd_ratio', 'win_rate', 'kills_per_match', 'score'] as const).flatMap((key) => {
                          const row = metrics.find((m) => m.key === key)
                          if (!row) return []
                          const delta = row.delta ? parseFloat(row.delta) : null
                          return [
                            <DeltaCard
                              key={key}
                              label={row.label}
                              value={row.value_a}
                              delta={delta}
                              lowerIsBetter={false}
                            />,
                          ]
                        })}
                      </div>
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.summary_empty_title')}
                        description={t('session.compare.summary_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                {/* Métriques détaillées (chart 05) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.metrics_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {metrics.length > 0 ? (
                      <table className="w-full">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            <th className="py-2 pr-4">{t('session.compare.metric_col_metric')}</th>
                            <th className="py-2 pr-4 text-right text-compare-a">{t('session.compare.metric_col_session_a')}</th>
                            <th className="py-2 pr-4 text-right text-compare-b">{t('session.compare.metric_col_session_b')}</th>
                            <th className="py-2 text-right">{t('session.compare.metric_col_delta')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {metrics.map((row) => (
                            <MetricRow key={row.key} row={row} />
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.metrics_empty_title')}
                        description={t('session.compare.metrics_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                {/* MMR moyen (chart 06) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.mmr_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    <SessionCompareMMR
                      sessionA={data.session_a}
                      sessionB={data.session_b}
                      labels={{
                        title: t('session.compare.mmr_title'),
                        teamMMR: t('session.compare.mmr_team'),
                        enemyMMR: t('session.compare.mmr_enemy'),
                        empty: t('session.compare.mmr_empty'),
                      }}
                    />
                  </CardContent>
                </Card>

                {/* Cumulative + K/D progression (charts 09 et 10) */}
                <div className="grid gap-6 sm:grid-cols-2">
                  <Card>
                    <CardContent className="pt-4">
                      <SessionCompareCumulative
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.cumulative_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: chartEmpty,
                        }}
                        height={280}
                      />
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pt-4">
                      <SessionCompareKDProgression
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.kd_progression_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: chartEmpty,
                        }}
                        height={280}
                      />
                    </CardContent>
                  </Card>
                </div>

                {/* Progression perf score + LUSR/CSR */}
                <div className="grid gap-6 sm:grid-cols-2">
                  <Card>
                    <CardContent className="pt-4">
                      <SessionComparePerfProgression
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.perf_progression_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: t('session.compare.perf_progression_empty'),
                        }}
                        height={280}
                      />
                    </CardContent>
                  </Card>
                  <Card>
                    <CardContent className="pt-4">
                      <SessionCompareSkillProgression
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.skill_progression_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: t('session.compare.skill_progression_empty'),
                        }}
                        height={280}
                      />
                    </CardContent>
                  </Card>
                </div>

                {/* OC/DR + Engagement */}
                <div className="grid gap-6 sm:grid-cols-2">
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">{t('session.compare.ocdr_title')}</CardTitle>
                    </CardHeader>
                    <CardContent className="pb-4">
                      <SessionCompareOCDR
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.ocdr_title'),
                          empty: t('session.compare.ocdr_empty'),
                        }}
                      />
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-base">{t('session.compare.engagement_title')}</CardTitle>
                    </CardHeader>
                    <CardContent className="pb-4">
                      <SessionCompareEngagement
                        sessionA={data.session_a}
                        sessionB={data.session_b}
                        labels={{
                          title: t('session.compare.engagement_title'),
                          progressionTitle: t('session.compare.engagement_progression_title'),
                          sessionA: labelA,
                          sessionB: labelB,
                          empty: t('session.compare.engagement_empty'),
                        }}
                        height={240}
                      />
                    </CardContent>
                  </Card>
                </div>

                {/* Profil de participation 6 axes (chart 13) */}
                <Card>
                  <CardContent className="pt-4">
                    <SessionCompareParticipation
                      sessionA={data.session_a}
                      sessionB={data.session_b}
                      labels={{
                        title: t('session.compare.participation_title'),
                        sessionA: labelA,
                        sessionB: labelB,
                        empty: t('session.compare.participation_empty'),
                        combat: t('session.compare.participation_combat'),
                        survival: t('session.compare.participation_survival'),
                        support: t('session.compare.participation_support'),
                        score: t('session.compare.participation_score'),
                        objective: t('session.compare.participation_objective'),
                        impact: t('session.compare.participation_impact'),
                      }}
                      height={320}
                    />
                  </CardContent>
                </Card>

                {/* Historique des matchs (chart 14) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.history_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    <SessionCompareMatchHistory
                      matchesA={data.session_a?.matches ?? []}
                      matchesB={data.session_b?.matches ?? []}
                      labels={{
                        title: t('session.compare.history_title'),
                        tabA: t('session.compare.history_tab_a'),
                        tabB: t('session.compare.history_tab_b'),
                        colDate: t('session.compare.history_col_date'),
                        colKDA: t('session.compare.history_col_kda'),
                        colMode: t('session.compare.history_col_mode'),
                        colPerf: t('session.compare.history_col_perf'),
                        win: t('session.compare.history_win'),
                        loss: t('session.compare.history_loss'),
                        other: t('session.compare.history_other'),
                        empty: t('session.compare.history_empty'),
                      }}
                    />
                  </CardContent>
                </Card>

                {/* Par carte (chart 12) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.maps_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {mapsTable.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            <th className="py-2 pr-4">{t('session.compare.map_col_map')}</th>
                            <th className="py-2 pr-4 text-center text-compare-a">{t('session.compare.map_col_a_matches')}</th>
                            <th className="py-2 pr-4 text-center text-compare-a">{t('session.compare.map_col_a_wl')}</th>
                            <th className="py-2 pr-4 text-center text-compare-b">{t('session.compare.map_col_b_matches')}</th>
                            <th className="py-2 text-center text-compare-b">{t('session.compare.map_col_b_wl')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {mapsTable.map((row) => (
                            <MapTableRow key={row.map_name} row={row} />
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.maps_empty_title')}
                        description={t('session.compare.maps_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                {/* Par mode (chart 11) */}
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.modes_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {modesTable.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            <th className="py-2 pr-4">{t('session.compare.mode_col_mode')}</th>
                            <th className="py-2 pr-4 text-center text-compare-a">{t('session.compare.mode_col_a_matches')}</th>
                            <th className="py-2 pr-4 text-center text-compare-a">{t('session.compare.mode_col_a_wins')}</th>
                            <th className="py-2 pr-4 text-center text-compare-b">{t('session.compare.mode_col_b_matches')}</th>
                            <th className="py-2 text-center text-compare-b">{t('session.compare.mode_col_b_wins')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {modesTable.map((row) => (
                            <ModeTableRow key={row.mode_name} row={row} />
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.modes_empty_title')}
                        description={t('session.compare.modes_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>
              </>
            ) : (
              <EmptyStateCard
                title={t('session.compare.incomplete_title')}
                description={t('session.compare.incomplete_description')}
              />
            )}
          </>
        ) : (
          <EmptyStateCard
            title={t('session.empty.no_sessions_title')}
            description={t('session.empty.no_sessions_description')}
          />
        )}
    </div>
  )
}
