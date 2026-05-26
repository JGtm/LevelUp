/**
 * SessionDetailPage — page détail d'une session avec drawer compare side-by-side.
 *
 * Structure orchestrale ; les blocs métier vivent dans des sous-fichiers :
 *  - SessionSummaryCard (résumé KPI session active)
 *  - SessionOutcomeTape / KDATimeline / KillsDonut / PerfTrend (4 charts)
 *  - SessionMatchesTable (liste des matchs)
 *  - SessionCompareDrawer (panneau latéral side-by-side, contient le même
 *    layout pour la session comparée + SessionCompareMetrics)
 * Helpers partagés (i18n hook, formatters, outcome tone) dans `_shared.ts`.
 *
 * Layout : quand le drawer est ouvert, le contenu de la page reste pleine
 * largeur mais reçoit `xl:pr-[50vw]` pour libérer la moitié droite du viewport,
 * créant la vue 2 colonnes côte-à-côte sur desktop ≥ xl.
 */
import { useState } from 'react'
import { useParams, useSearch } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useLocalFilterBar } from '@/features/_shared/useLocalFilterBar'

import { useSessionDetailPage } from './queries'
import { useSessionT } from './_shared'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionKDATimeline } from './SessionKDATimeline'
import { SessionOutcomeTape } from './SessionOutcomeTape'
import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionPerfTrend } from './SessionPerfTrend'
import { SessionCompareDrawer } from './SessionCompareDrawer'

export function SessionDetailPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { session: initialSession } = useSearch({ strict: false }) as { session?: string }
  const t = useSessionT()

  const [sessionLabel, setSessionLabel] = useState(initialSession ?? '')
  const [compareSessionLabel, setCompareSessionLabel] = useState('')
  const [enableCompare, setEnableCompare] = useState(false)

  const { committedFilterContext, committedHash, bar } = useLocalFilterBar({
    playerSlug,
    labels: {
      experience: t('session.filters.experience'),
      experienceAll: t('session.filters.experience_all'),
      experienceRanked: t('session.filters.experience_ranked'),
      experienceUnranked: t('session.filters.experience_unranked'),
      playlists: t('session.filters.playlists'),
      modes: t('session.filters.modes'),
      reset: t('session.filters.reset'),
    },
  })

  const { data, isLoading, isError, refetch } = useSessionDetailPage(
    playerSlug,
    {
      filters: committedFilterContext,
      session_label: sessionLabel || undefined,
      compare_session_label: compareSessionLabel || undefined,
      enable_compare: enableCompare,
    },
    committedHash,
    sessionLabel,
    compareSessionLabel,
    enableCompare,
  )

  if (isLoading) {
    return (
      <div className="flex flex-col">
        {bar}
        <div className="flex h-full items-center justify-center p-6">
          <Spinner size="lg" label={t('session.detail.loading')} />
        </div>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex flex-col">
        {bar}
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
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex flex-col">
        {bar}
        <div className="p-6">
          <EmptyStateCard
            title={t('session.detail.empty_title')}
            description={t('session.detail.empty_description')}
            actionLabel={t('session.errors.retry')}
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
  const drawerOpen = enableCompare && hasSessions

  return (
    <div className="flex flex-col">
      {bar}
      <div className={`space-y-6 p-6 transition-[padding] duration-200 ${drawerOpen ? 'xl:pr-[calc(50vw+1.5rem)]' : ''}`}>
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

      <SessionCompareDrawer
        open={drawerOpen}
        onClose={() => setEnableCompare(false)}
        compareSession={data.compare_session}
        compareMatches={data.compare_matches ?? []}
        compareMetrics={data.compare_metrics}
        suggestedCompare={data.suggested_compare}
        previousLabel={data.previous_session_label ?? null}
        nextLabel={data.next_session_label ?? null}
        onSelectLabel={(label) => setCompareSessionLabel(label)}
      />
    </div>
  )
}
