/**
 * SessionComparePage — comparaison A/B de deux sessions.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useSessionComparePage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import type { SessionCompareEntry, SessionCompareMetricRow } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { sessionManifest, type SessionManifestKey } from '@/lib/i18n/generated/session'
import { useAppShellStore } from '@/stores/appShellStore'
import { SessionOutcomesDonut } from './SessionOutcomesDonut'

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

export function SessionComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const t = useSessionT()

  const [sessionA, setSessionA] = useState('')
  const [sessionB, setSessionB] = useState('')

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
      <div className="flex h-full items-center justify-center">
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

  const hasAvailableSessions = data.available_sessions.length > 0
  const hasComparisonSelection = Boolean(sessionA && sessionB)

  return (
    <div className="flex flex-col">
      <div className="space-y-6 p-6">
        {hasAvailableSessions ? (
          <>
            {/* Sélecteurs A / B */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.compare.session_a_label')}</label>
                <select
                  className="w-full rounded-md border border-border px-3 py-2 text-sm"
                  value={sessionA}
                  onChange={(e) => setSessionA(e.target.value)}
                >
                  <option value="">{t('session.compare.placeholder_select')}</option>
                  {data.available_sessions.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.compare.session_b_label')}</label>
                <select
                  className="w-full rounded-md border border-border px-3 py-2 text-sm"
                  value={sessionB}
                  onChange={(e) => setSessionB(e.target.value)}
                >
                  <option value="">{t('session.compare.placeholder_select')}</option>
                  {data.available_sessions.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
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
                          sessionA: t('session.compare.session_card_title', {
                            side: 'A',
                            suffix: data.session_a?.session_label
                              ? ` — ${data.session_a.session_label}`
                              : '',
                          }),
                          sessionB: t('session.compare.session_card_title', {
                            side: 'B',
                            suffix: data.session_b?.session_label
                              ? ` — ${data.session_b.session_label}`
                              : '',
                          }),
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

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.radar_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.radar_chart ? (
                      <PlotlyChart figure={data.radar_chart} />
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.radar_empty_title')}
                        description={t('session.compare.radar_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.summary_title')}</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {data.metrics.length > 0 && data.session_a && data.session_b ? (
                      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                        {(['kd_ratio', 'win_rate', 'kills_per_match', 'score'] as const).flatMap((key) => {
                          const row = data.metrics.find((m) => m.key === key)
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

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.metrics_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.metrics.length > 0 ? (
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
                          {data.metrics.map((row) => (
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

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.kd_progression_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.kd_progression_chart ? (
                      <PlotlyChart figure={data.kd_progression_chart} />
                    ) : (
                      <EmptyStateNotice
                        title={t('session.compare.kd_progression_empty_title')}
                        description={t('session.compare.kd_progression_empty_description')}
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.maps_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.maps_table.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            {Object.keys(data.maps_table[0]).map((k) => (
                              <th key={k} className="py-2 pr-4">{k}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {data.maps_table.map((row, i) => (
                            <tr key={i} className="border-b last:border-0">
                              {Object.values(row).map((v, j) => (
                                <td key={j} className="py-1.5 pr-4 text-foreground">
                                  {String(v ?? '-')}
                                </td>
                              ))}
                            </tr>
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

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">{t('session.compare.modes_title')}</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.modes_table.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            {Object.keys(data.modes_table[0]).map((k) => (
                              <th key={k} className="py-2 pr-4">{k}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {data.modes_table.map((row, i) => (
                            <tr key={i} className="border-b last:border-0">
                              {Object.values(row).map((v, j) => (
                                <td key={j} className="py-1.5 pr-4 text-foreground">
                                  {String(v ?? '-')}
                                </td>
                              ))}
                            </tr>
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
    </div>
  )
}
