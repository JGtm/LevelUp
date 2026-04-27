import { useState } from 'react'
import type React from 'react'
import { useParams, useSearch } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DeltaCard } from '@/components/ui/delta-card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SessionCompareEntry, SessionCompareMetricRow, SessionDetailMatchRow } from '@/lib/api/types'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'

import { outcomeScale } from '@/lib/accessibility/scales'
import { tokenCssVar } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useSessionDetailPage } from './queries'
import { formatMessage } from '@/lib/i18n/format'
import { sessionManifest, type SessionManifestKey } from '@/lib/i18n/generated/session'
import { useAppShellStore } from '@/stores/appShellStore'

function useSessionT() {
  const locale = useAppShellStore((s) => s.locale)
  return (key: SessionManifestKey, values?: Record<string, string | number>) =>
    formatMessage(sessionManifest, key, locale, values)
}

function SessionSummaryCard({
  title,
  entry,
  tone,
}: {
  title: string
  entry: SessionCompareEntry | null
  tone: 'primary' | 'compare'
}) {
  const toneClass = tone === 'primary' ? 'border-primary/20 bg-primary/5' : 'border-compare-b bg-compare-b/10'
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  const t = useSessionT()

  if (!entry) {
    return (
      <Card className={toneClass}>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice
            title={t('session.detail.summary_unavailable_title')}
            description={t('session.detail.summary_unavailable_description')}
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={toneClass}>
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base">{title}</CardTitle>
            <p className="mt-1 text-xs text-muted-foreground">{entry.session_label}</p>
          </div>
          {entry.dominant_category && <Badge variant="secondary">{entry.dominant_category}</Badge>}
        </div>
      </CardHeader>
      <CardContent className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <SessionStat label={t('session.detail.stat_matches')} value={entry.total_matches.toString()} />
        <SessionStat label={t('session.detail.stat_wins_losses')} value={`${entry.wins} / ${entry.losses}`} />
        <SessionStat label={labelOf('kda', t('session.detail.stat_kda'))} value={formatNumber(entry.kda, 2)} />
        <SessionStat label={t('session.detail.stat_perf_score')} value={formatNumber(entry.performance_score, 1)} />
      </CardContent>
    </Card>
  )
}

function SessionStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-border/60 bg-background/70 p-3">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{label}</p>
      <p className="mt-2 text-lg font-semibold text-foreground">{value}</p>
    </div>
  )
}

function SessionMatchesTable({ matches }: { matches: SessionDetailMatchRow[] }) {
  // Phase 4 plan finition multi-titres : libellés outcome via outcomes.toml.
  // Si MULTI_TITLE_API_ENABLED=false, le hook retourne undefined et on tombe
  // sur la clé canonique brute (UX dégradée mais lisible).
  const { data: fieldMappings } = useFieldMappings()
  const t = useSessionT()
  const outcomeLabel = (outcome: number | null) => {
    const key =
      outcome === 2 ? 'win' : outcome === 3 ? 'loss' : outcome === 1 ? 'tie' : outcome === 4 ? 'dnf' : null
    if (!key) return '—'
    return fieldMappings?.outcomes?.[key]?.label ?? key
  }

  if (matches.length === 0) {
    return (
      <EmptyStateNotice
        title={t('session.detail.matches_empty_title')}
        description={t('session.detail.matches_empty_description')}
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs uppercase tracking-[0.16em] text-muted-foreground">
            <th className="px-3 py-3 font-medium">{t('session.detail.col_time')}</th>
            <th className="px-3 py-3 font-medium">{t('session.detail.col_mode')}</th>
            <th className="px-3 py-3 font-medium">{t('session.detail.col_playlist')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_kda')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_accuracy')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_perf_score')}</th>
            <th className="px-3 py-3 text-right font-medium">{t('session.detail.col_outcome')}</th>
          </tr>
        </thead>
        <tbody>
          {matches.map((match) => (
            <tr key={match.match_id} className="border-b border-border/60 text-foreground last:border-0">
              <td className="px-3 py-3 text-muted-foreground">{formatShortDateTime(match.start_time)}</td>
              <td className="px-3 py-3 font-medium">{match.pair_name || '—'}</td>
              <td className="px-3 py-3 text-muted-foreground">{match.playlist_name || '—'}</td>
              <td className="px-3 py-3 text-right tabular-nums">{`${match.kills}/${match.deaths}/${match.assists}`}</td>
              <td className="px-3 py-3 text-right tabular-nums">{formatPercent(match.accuracy)}</td>
              <td className="px-3 py-3 text-right tabular-nums">{formatNumber(match.performance_score, 1)}</td>
              <td className="px-3 py-3 text-right">
                {(() => { const tone = matchOutcomeTone(match.outcome); return <span className={tone.className} style={tone.style}>{outcomeLabel(match.outcome)}</span> })()}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SessionCompareMetrics({ metrics }: { metrics: SessionCompareMetricRow[] }) {
  const t = useSessionT()
  const summaryKeys = ['kd_ratio', 'win_rate', 'kills_per_match', 'score']
  const summaryRows = summaryKeys
    .map((key) => metrics.find((row) => row.key === key))
    .filter((row): row is SessionCompareMetricRow => Boolean(row))

  return (
    <div className="space-y-4">
      {summaryRows.length > 0 ? (
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          {summaryRows.map((row) => (
            <DeltaCard
              key={row.key}
              label={row.label}
              value={row.value_a}
              delta={parseDelta(row.delta)}
              lowerIsBetter={false}
            />
          ))}
        </div>
      ) : null}

      {metrics.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[680px] text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs uppercase tracking-[0.16em] text-muted-foreground">
                <th className="px-3 py-3 font-medium">{t('session.detail.compare_col_metric')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_session_active')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_session_compared')}</th>
                <th className="px-3 py-3 text-right font-medium">{t('session.detail.compare_col_delta')}</th>
              </tr>
            </thead>
            <tbody>
              {metrics.map((row) => (
                <tr key={row.key} className="border-b border-border/60 last:border-0">
                  <td className="px-3 py-3 text-muted-foreground">{row.label}</td>
                  <td className="px-3 py-3 text-right font-medium text-foreground">{row.value_a}</td>
                  <td className="px-3 py-3 text-right font-medium text-compare-b">{row.value_b}</td>
                  <td className="px-3 py-3 text-right text-xs text-muted-foreground">{row.delta ?? '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyStateNotice
          title={t('session.detail.compare_metrics_empty_title')}
          description={t('session.detail.compare_metrics_empty_description')}
        />
      )}
    </div>
  )
}

export function SessionDetailPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { session: initialSession } = useSearch({ strict: false }) as { session?: string }
  const filterContext = useGlobalFilterStore((state) => state.filterContext)
  const filterContextHash = useGlobalFilterStore((state) => state.filterContextHash)
  const t = useSessionT()

  const [sessionLabel, setSessionLabel] = useState(initialSession ?? '')
  const [compareSessionLabel, setCompareSessionLabel] = useState('')
  const [enableCompare, setEnableCompare] = useState(false)

  const { data, isLoading, isError, refetch } = useSessionDetailPage(
    playerSlug,
    {
      filters: filterContext,
      session_label: sessionLabel || undefined,
      compare_session_label: compareSessionLabel || undefined,
      enable_compare: enableCompare,
    },
    filterContextHash,
    sessionLabel,
    compareSessionLabel,
    enableCompare,
  )

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label={t('session.detail.loading')} />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('session.detail.load_error')}</p>
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
          title={t('session.detail.empty_title')}
          description={t('session.detail.empty_description')}
          actionLabel={t('session.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const selectedSessionLabel = sessionLabel || data.current_session?.session_label || ''
  const selectedCompareSessionLabel =
    compareSessionLabel || data.compare_session?.session_label || data.suggested_compare?.session_label || ''
  const hasSessions = data.available_sessions.length > 0
  const suggestionAvailable = Boolean(data.suggested_compare)

  return (
    <div className="flex flex-col">
      <div className="space-y-6 p-6">
        {suggestionAvailable && !enableCompare && (
          <div className="flex justify-end">
            <Button
              variant="outline"
              onClick={() => {
                setCompareSessionLabel(data.suggested_compare?.session_label ?? '')
                setEnableCompare(true)
              }}
            >
              {t('session.detail.suggested_compare_button')}
            </Button>
          </div>
        )}
        {hasSessions ? (
          <>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('session.detail.selection_card')}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
                <div>
                  <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.detail.session_active')}</label>
                  <select
                    className="w-full rounded-md border border-border px-3 py-2 text-sm"
                    value={selectedSessionLabel}
                    onChange={(event) => setSessionLabel(event.target.value)}
                  >
                    {data.available_sessions.map((session) => (
                      <option key={session} value={session}>
                        {session}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="mb-1 block text-xs font-medium text-muted-foreground">{t('session.detail.session_compared')}</label>
                  <select
                    className="w-full rounded-md border border-border px-3 py-2 text-sm"
                    value={selectedCompareSessionLabel}
                    onChange={(event) => setCompareSessionLabel(event.target.value)}
                  >
                    <option value="">{t('session.detail.smart_selection')}</option>
                    {data.available_sessions
                      .filter((session) => session !== selectedSessionLabel)
                      .map((session) => (
                        <option key={session} value={session}>
                          {session}
                        </option>
                      ))}
                  </select>
                </div>

                <div className="flex items-end gap-2">
                  <Button
                    variant={enableCompare ? 'secondary' : 'default'}
                    onClick={() => setEnableCompare((value) => !value)}
                  >
                    {enableCompare ? t('session.detail.compare_hide') : t('session.detail.compare_show')}
                  </Button>
                </div>
              </CardContent>
            </Card>

            {data.suggested_compare ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t('session.detail.suggestion_title')}</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-foreground">{data.suggested_compare.session_label}</p>
                    <p className="text-sm text-muted-foreground">{data.suggested_compare.reason}</p>
                  </div>
                  <Badge variant="secondary">{data.suggested_compare.strategy}</Badge>
                </CardContent>
              </Card>
            ) : null}

            <div className={enableCompare && data.compare_session ? 'grid gap-6 xl:grid-cols-2' : 'grid gap-6'}>
              <SessionSummaryCard title={t('session.detail.session_active')} entry={data.current_session} tone="primary" />
              {enableCompare && data.compare_session ? (
                <SessionSummaryCard title={t('session.detail.session_compared')} entry={data.compare_session} tone="compare" />
              ) : null}
            </div>

            {enableCompare ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">{t('session.detail.compare_view_title')}</CardTitle>
                </CardHeader>
                <CardContent>
                  {data.compare_session ? (
                    <SessionCompareMetrics metrics={data.compare_metrics} />
                  ) : (
                    <EmptyStateNotice
                      title={t('session.detail.no_compare_title')}
                      description={t('session.detail.no_compare_description')}
                    />
                  )}
                </CardContent>
              </Card>
            ) : null}

            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('session.detail.matches_card')}</CardTitle>
              </CardHeader>
              <CardContent>
                <SessionMatchesTable matches={data.matches} />
              </CardContent>
            </Card>
          </>
        ) : (
          <EmptyStateCard
            title={t('session.detail.no_session_in_scope_title')}
            description={t('session.detail.no_session_in_scope_description')}
          />
        )}
      </div>
    </div>
  )
}

function formatNumber(value: number | null, digits: number) {
  if (value == null) {
    return '—'
  }
  return value.toFixed(digits)
}

function formatPercent(value: number | null) {
  if (value == null) {
    return '—'
  }
  return `${value.toFixed(1)}%`
}

function parseDelta(delta: string | null) {
  if (!delta) {
    return null
  }
  const parsed = Number.parseFloat(delta)
  return Number.isNaN(parsed) ? null : parsed
}

function formatShortDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat('fr-FR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

// La résolution du libellé d'outcome est faite directement dans le composant
// SessionMatchesTable via useFieldMappings + outcomes.toml. Le fallback
// minimal en cas d'API absente est défini inline avec la clé canonique pour
// rester scannable par le lint anti-hardcode (les libellés FR concrets
// viennent du TOML).

const OUTCOME_INT_KEY: Record<number, string> = { 2: 'win', 1: 'draw', 3: 'loss', 4: 'dnf' }

function matchOutcomeTone(outcome: number | null): { className: string; style?: React.CSSProperties } {
  if (outcome == null) return { className: 'text-muted-foreground' }
  const key = OUTCOME_INT_KEY[outcome]
  const token = key ? outcomeScale(key) : null
  if (!token) return { className: 'text-muted-foreground' }
  return { className: 'font-medium', style: { color: tokenCssVar(token) } }
}
