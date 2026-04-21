import { useState } from 'react'
import { useParams } from '@tanstack/react-router'

import { PageHeader } from '@/components/shell/PageHeader'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { DeltaCard } from '@/components/ui/delta-card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import type { SessionCompareEntry, SessionCompareMetricRow, SessionDetailMatchRow } from '@/lib/api/types'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'

import { useSessionDetailPage } from './queries'

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

  if (!entry) {
    return (
      <Card className={toneClass}>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{title}</CardTitle>
        </CardHeader>
        <CardContent>
          <EmptyStateNotice
            title="Session indisponible"
            description="La session demandée n'a pas pu être reconstruite avec les filtres actuels."
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
        <SessionStat label="Matchs" value={entry.total_matches.toString()} />
        <SessionStat label="Victoires / Défaites" value={`${entry.wins} / ${entry.losses}`} />
        <SessionStat label="KDA" value={formatNumber(entry.kda, 2)} />
        <SessionStat label="Score perf." value={formatNumber(entry.performance_score, 1)} />
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
  if (matches.length === 0) {
    return (
      <EmptyStateNotice
        title="Aucun match dans cette session"
        description="La session sélectionnée ne contient aucun match exploitable avec les filtres actuels."
      />
    )
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full min-w-[760px] text-sm">
        <thead>
          <tr className="border-b border-border text-left text-xs uppercase tracking-[0.16em] text-muted-foreground">
            <th className="px-3 py-3 font-medium">Heure</th>
            <th className="px-3 py-3 font-medium">Mode</th>
            <th className="px-3 py-3 font-medium">Playlist</th>
            <th className="px-3 py-3 text-right font-medium">K / D / A</th>
            <th className="px-3 py-3 text-right font-medium">Précision</th>
            <th className="px-3 py-3 text-right font-medium">Score perf.</th>
            <th className="px-3 py-3 text-right font-medium">Résultat</th>
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
                <span className={matchOutcomeTone(match.outcome)}>{matchOutcomeLabel(match.outcome)}</span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function SessionCompareMetrics({ metrics }: { metrics: SessionCompareMetricRow[] }) {
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
                <th className="px-3 py-3 font-medium">Métrique</th>
                <th className="px-3 py-3 text-right font-medium">Session active</th>
                <th className="px-3 py-3 text-right font-medium">Session comparée</th>
                <th className="px-3 py-3 text-right font-medium">Delta</th>
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
          title="Comparaison indisponible"
          description="Aucune métrique comparative n'a été calculée pour cette paire de sessions."
        />
      )}
    </div>
  )
}

export function SessionDetailPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((state) => state.filterContext)
  const filterContextHash = useGlobalFilterStore((state) => state.filterContextHash)

  const [sessionLabel, setSessionLabel] = useState('')
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
        <Spinner size="lg" label="Chargement de la session…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement de la session.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex flex-col">
        <PageHeader
          title="Sessions"
          subtitle="Lecture détaillée d'une session avec suggestion de comparaison intelligente."
        />
        <div className="p-6">
          <EmptyStateCard
            title="Sessions indisponibles"
            description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie les sessions calculées et les filtres actifs."
            actionLabel="Réessayer"
            onAction={() => refetch()}
          />
        </div>
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
      <PageHeader
        title="Sessions"
        subtitle="Lecture détaillée d'une session solo avec suggestion de comparaison intégrée."
        actions={
          suggestionAvailable && !enableCompare ? (
            <Button
              variant="outline"
              onClick={() => {
                setCompareSessionLabel(data.suggested_compare?.session_label ?? '')
                setEnableCompare(true)
              }}
            >
              Comparer à la session proche
            </Button>
          ) : undefined
        }
      />

      <div className="space-y-6 p-6">
        {hasSessions ? (
          <>
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Sélection</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto]">
                <div>
                  <label className="mb-1 block text-xs font-medium text-muted-foreground">Session active</label>
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
                  <label className="mb-1 block text-xs font-medium text-muted-foreground">Session comparée</label>
                  <select
                    className="w-full rounded-md border border-border px-3 py-2 text-sm"
                    value={selectedCompareSessionLabel}
                    onChange={(event) => setCompareSessionLabel(event.target.value)}
                  >
                    <option value="">Sélection intelligente</option>
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
                    {enableCompare ? 'Masquer comparaison' : 'Activer comparaison'}
                  </Button>
                </div>
              </CardContent>
            </Card>

            {data.suggested_compare ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Suggestion similaire</CardTitle>
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
              <SessionSummaryCard title="Session active" entry={data.current_session} tone="primary" />
              {enableCompare && data.compare_session ? (
                <SessionSummaryCard title="Session comparée" entry={data.compare_session} tone="compare" />
              ) : null}
            </div>

            {enableCompare ? (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Lecture comparative</CardTitle>
                </CardHeader>
                <CardContent>
                  {data.compare_session ? (
                    <SessionCompareMetrics metrics={data.compare_metrics} />
                  ) : (
                    <EmptyStateNotice
                      title="Aucune session de comparaison"
                      description="Sélectionne une session de référence ou réactive la suggestion automatique."
                    />
                  )}
                </CardContent>
              </Card>
            ) : null}

            <Card>
              <CardHeader>
                <CardTitle className="text-base">Détail des matchs</CardTitle>
              </CardHeader>
              <CardContent>
                <SessionMatchesTable matches={data.matches} />
              </CardContent>
            </Card>
          </>
        ) : (
          <EmptyStateCard
            title="Aucune session disponible"
            description="Aucune session n'a pu être reconstruite avec les filtres actuels."
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

function matchOutcomeLabel(outcome: number | null) {
  if (outcome === 2) {
    return 'Victoire'
  }
  if (outcome === 3) {
    return 'Défaite'
  }
  return '—'
}

function matchOutcomeTone(outcome: number | null) {
  if (outcome === 2) {
    return 'font-medium text-emerald-600'
  }
  if (outcome === 3) {
    return 'font-medium text-rose-600'
  }
  return 'text-muted-foreground'
}
