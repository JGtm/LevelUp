/**
 * SessionDetailPage — page détail d'une session avec colonne compare inline.
 *
 * Quand le compare est ouvert, la page passe en deux colonnes côte-à-côte (≥ xl)
 * dans le même conteneur de scroll (`main`), donnant un scroll synchronisé naturel.
 * La colonne compare commence sous la NavL2 car elle est dans le flux DOM de l'Outlet,
 * pas en position fixed.
 */
import { useState } from 'react'
import { useParams, useSearch } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useSoloFilterStore } from '@/stores/soloFilterStore'

import { useSessionDetailPage } from './queries'
import { useSessionT } from './_shared'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionKDATimeline } from './SessionKDATimeline'
import { SessionOutcomeTape } from './SessionOutcomeTape'
import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionPerfTrend } from './SessionPerfTrend'
import { SessionCompareMetrics } from './SessionCompareMetrics'

export function SessionDetailPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { session: initialSession } = useSearch({ strict: false }) as { session?: string }
  const t = useSessionT()

  const [sessionLabel, setSessionLabel] = useState(initialSession ?? '')
  const [compareSessionLabel, setCompareSessionLabel] = useState('')
  const [enableCompare, setEnableCompare] = useState(false)

  const filterContext = useSoloFilterStore((s) => s.filterContext)
  const filterContextHash = useSoloFilterStore((s) => s.filterContextHash)

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
      <div className="flex h-full items-center justify-center p-6">
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
  const drawerOpen = enableCompare && hasSessions

  return (
    <div className={drawerOpen ? 'flex flex-col xl:flex-row' : ''}>
      {/* Colonne principale */}
      <div className={`space-y-6 p-6 ${drawerOpen ? 'xl:flex-1 xl:min-w-0 xl:border-r' : ''}`}>
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
                    className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
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
                    className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
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
                    {enableCompare ? t('session.detail.drawer_close') : t('session.detail.drawer_open')}
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

            <SessionSummaryCard
              title={t('session.detail.session_active')}
              entry={data.current_session}
              tone="primary"
            />

            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('session.detail.chart_outcomes_title')}</CardTitle>
              </CardHeader>
              <CardContent>
                <SessionOutcomeTape matches={data.matches} />
              </CardContent>
            </Card>

            <div className="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
              <SessionKDATimeline title={t('session.detail.chart_kda_title')} matches={data.matches} />
              <SessionKillsDonut title={t('session.detail.chart_kills_donut_title')} matches={data.matches} />
            </div>

            <SessionPerfTrend title={t('session.detail.chart_perf_title')} matches={data.matches} />

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

      {/* Colonne compare — inline dans le même flux de scroll */}
      {drawerOpen && (
        <div className="flex flex-col space-y-4 border-t p-4 xl:w-[50%] xl:shrink-0 xl:border-l xl:border-t-0">
          {/* En-tête navigation */}
          <div className="flex flex-col gap-2 border-b pb-3">
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold text-foreground">
                {data.compare_session
                  ? t('session.detail.drawer_title', { label: data.compare_session.session_label })
                  : t('session.detail.session_compared')}
              </h2>
              <button
                type="button"
                onClick={() => setEnableCompare(false)}
                aria-label={t('session.detail.drawer_close_aria')}
                className="rounded p-1 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              >
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={!data.previous_session_label}
                onClick={() => data.previous_session_label && setCompareSessionLabel(data.previous_session_label)}
              >
                {t('session.detail.drawer_prev_session')}
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={!data.next_session_label}
                onClick={() => data.next_session_label && setCompareSessionLabel(data.next_session_label)}
              >
                {t('session.detail.drawer_next_session')}
              </Button>
              {data.suggested_compare && (
                <Button
                  size="sm"
                  variant="secondary"
                  onClick={() => setCompareSessionLabel(data.suggested_compare!.session_label)}
                >
                  {t('session.detail.drawer_use_suggested')}
                </Button>
              )}
            </div>
          </div>

          {/* Contenu session comparée */}
          {data.compare_session ? (
            <>
              <SessionSummaryCard
                title={t('session.detail.session_compared')}
                entry={data.compare_session}
                tone="compare"
              />

              <SessionOutcomeTape matches={data.compare_matches ?? []} />

              <div className="grid gap-4 2xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
                <SessionKDATimeline
                  title={t('session.detail.chart_kda_title')}
                  matches={data.compare_matches ?? []}
                />
                <SessionKillsDonut
                  title={t('session.detail.chart_kills_donut_title')}
                  matches={data.compare_matches ?? []}
                />
              </div>

              <SessionPerfTrend
                title={t('session.detail.chart_perf_title')}
                matches={data.compare_matches ?? []}
              />

              <SessionCompareMetrics metrics={data.compare_metrics} />
            </>
          ) : (
            <EmptyStateNotice
              title={t('session.detail.no_compare_title')}
              description={t('session.detail.drawer_no_compare')}
            />
          )}
        </div>
      )}
    </div>
  )
}
