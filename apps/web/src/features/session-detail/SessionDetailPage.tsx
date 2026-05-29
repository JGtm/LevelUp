/**
 * SessionDetailPage — page détail d'une session avec colonne compare inline.
 *
 * Quand le compare est ouvert, la page passe en deux colonnes côte-à-côte (≥ xl)
 * dans le même conteneur de scroll (`main`), donnant un scroll synchronisé naturel.
 * La colonne compare commence sous la NavL2 car elle est dans le flux DOM de l'Outlet,
 * pas en position fixed.
 *
 * Animation : le conteneur est une grille dont la 2e piste passe de 0fr à 1fr via
 * une transition `grid-template-columns` — le panneau glisse et pousse la colonne
 * principale vers la gauche. La query utilise `keepPreviousData` pour ne pas
 * démonter le layout pendant le fetch compare (sinon la transition ne joue pas).
 */
import { useState } from 'react'
import { useParams, useSearch } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useSoloFilterStore } from '@/stores/soloFilterStore'

import { useSessionDetailPage } from './queries'
import { useSessionT } from './_shared'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionParamPills } from './SessionParamPills'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionKDATimeline } from './SessionKDATimeline'
import { SessionOutcomeTape } from './SessionOutcomeTape'
import { SessionKillsDonut } from './SessionKillsDonut'
import { SessionPerfTrend } from './SessionPerfTrend'
import { SessionFdaBars } from './SessionFdaBars'
import { SessionFdaRadar } from './SessionFdaRadar'
import { SessionOcdrScatter } from './SessionOcdrScatter'
import { SessionEngagementChart } from './SessionEngagementChart'
import { SessionCompareMetrics } from './SessionCompareMetrics'
import { SessionCompareKillsDonut } from '../session-compare/SessionCompareKillsDonut'
import { SessionCompareOutcomeTape } from '../session-compare/SessionCompareOutcomeTape'
import { SessionComparePerfProgression } from '../session-compare/SessionComparePerfProgression'
import { SessionCompareSkillProgression } from '../session-compare/SessionCompareSkillProgression'
import { SessionCompareOCDR } from '../session-compare/SessionCompareOCDR'
import { SessionCompareEngagement } from '../session-compare/SessionCompareEngagement'

export function SessionDetailPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { session: initialSession } = useSearch({ strict: false }) as { session?: string }
  const t = useSessionT()

  const [sessionLabel, setSessionLabel] = useState(initialSession ?? '')
  const [compareSessionLabel, setCompareSessionLabel] = useState('')
  const [enableCompare, setEnableCompare] = useState(false)

  const filterContext = useSoloFilterStore((s) => s.filterContext)
  const filterContextHash = useSoloFilterStore((s) => s.filterContextHash)

  const { data, isLoading, isError, isFetching, refetch } = useSessionDetailPage(
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
  const drawerOpen = enableCompare && hasSessions
  // Compare demande mais donnees pas encore arrivees (placeholder de la requete
  // precedente) : on affiche un spinner dans le panneau pendant qu'il glisse.
  const isCompareLoading = drawerOpen && !data.compare_session && isFetching

  return (
    // Layout en grille a deux colonnes anime via `grid-template-columns` : la
    // 2e colonne passe de 0fr a 1fr, ce qui fait glisser le panneau compare ET
    // pousse la colonne principale vers la gauche (un seul flux de scroll). Le
    // conteneur reste monte en permanence pour que la transition CSS se declenche.
    <div
      className={`xl:grid xl:transition-[grid-template-columns] xl:duration-300 xl:ease-out ${
        drawerOpen
          ? 'xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'
          : 'xl:grid-cols-[minmax(0,1fr)_minmax(0,0fr)]'
      }`}
    >
      {/* Colonne principale */}
      <div className={`min-w-0 space-y-6 p-6 ${drawerOpen ? 'xl:border-r' : ''}`}>
        {hasSessions ? (
          <>
            {/* En-tete session "L3" : sticky sous la NavL2 (le <main> scrolle), bleed
                horizontal via -mx-6 pour s'aligner sur les bords de la colonne. */}
            <div className="sticky top-0 z-20 -mx-6 -mt-6 flex flex-col gap-3 border-b border-border bg-background px-6 py-3 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  disabled={!data.previous_session_label}
                  onClick={() => data.previous_session_label && setSessionLabel(data.previous_session_label)}
                  aria-label={t('session.detail.drawer_prev_session')}
                  className="rounded-md border border-border p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <ChevronIcon direction="left" />
                </button>
                <div className="min-w-0">
                  <p className="text-3xs font-semibold uppercase tracking-label-md text-muted-foreground">
                    {t('session.detail.header_session')}
                  </p>
                  <div className="flex flex-wrap items-center gap-2">
                    <h1 className="truncate text-lg font-semibold text-foreground">{selectedSessionLabel}</h1>
                    <SessionParamPills entry={data.current_session} />
                  </div>
                </div>
                <button
                  type="button"
                  disabled={!data.next_session_label}
                  onClick={() => data.next_session_label && setSessionLabel(data.next_session_label)}
                  aria-label={t('session.detail.drawer_next_session')}
                  className="rounded-md border border-border p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground disabled:cursor-not-allowed disabled:opacity-40"
                >
                  <ChevronIcon direction="right" />
                </button>
              </div>

              {!drawerOpen && data.available_sessions.length >= 2 && (
                <div className="flex flex-col items-start gap-1 sm:items-end">
                  <Button
                    onClick={() => {
                      setCompareSessionLabel(data.suggested_compare?.session_label ?? '')
                      setEnableCompare(true)
                    }}
                    aria-label={t('session.detail.header_compare_aria')}
                  >
                    {t('session.detail.header_compare')}
                  </Button>
                  {data.suggested_compare && (
                    <p className="text-xs text-muted-foreground">
                      {t('session.detail.header_compare_hint', { label: data.suggested_compare.session_label })}
                      {' · '}
                      {data.suggested_compare.reason}
                    </p>
                  )}
                </div>
              )}
            </div>

            <SessionSummaryCard entry={data.current_session} />

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

            <div className="grid gap-6 xl:grid-cols-2">
              <SessionFdaRadar title={t('session.detail.chart_fda_per_game_title')} matches={data.matches} />
              <SessionFdaBars
                title={t('session.detail.chart_fda_per_minute_title')}
                matches={data.matches}
                mode="minute"
              />
            </div>

            <SessionPerfTrend title={t('session.detail.chart_perf_title')} matches={data.matches} />

            {/* Graphes remontes du drawer compare en vue single (sessionB=null). */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('session.compare.skill_progression_title')}</CardTitle>
              </CardHeader>
              <CardContent>
                <SessionCompareSkillProgression
                  sessionA={data.current_session}
                  sessionB={null}
                  labels={{
                    title: '',
                    sessionA: selectedSessionLabel,
                    sessionB: '',
                    empty: t('session.compare.skill_progression_empty'),
                  }}
                />
              </CardContent>
            </Card>

            <SessionOcdrScatter title={t('session.compare.ocdr_title')} matches={data.matches} />

            <SessionEngagementChart
              title={t('session.detail.chart_engagement_title')}
              matches={data.matches}
              entry={data.current_session}
            />

            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('session.detail.matches_card')}</CardTitle>
              </CardHeader>
              <CardContent>
                <SessionMatchesTable
                  matches={data.matches}
                  playerSlug={playerSlug}
                  variant={drawerOpen ? 'compact' : 'full'}
                />
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

      {/* Colonne compare — 2e piste de la grille. `overflow-hidden` masque le
          contenu tant que la piste est a 0fr ; il se revele au fur et a mesure
          qu'elle grandit (effet glisse + pousse). Sur mobile (< xl) elle se
          place sous la colonne principale. */}
      <div
        className={`overflow-hidden ${drawerOpen ? '' : 'hidden xl:block'}`}
        aria-hidden={!drawerOpen}
      >
        <div
          className={`flex flex-col space-y-4 border-t p-4 transition-opacity duration-300 xl:border-l xl:border-t-0 ${
            drawerOpen ? 'opacity-100' : 'opacity-0'
          }`}
        >
          {(drawerOpen || data.compare_session) && (
            <>
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
                <SessionParamPills entry={data.compare_session} />
                <select
                  value={selectedCompareSessionLabel}
                  onChange={(event) => setCompareSessionLabel(event.target.value)}
                  aria-label={t('session.detail.session_compared')}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground"
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

              {/* Contenu session comparée */}
              {data.compare_session ? (
                <>
                  <SessionSummaryCard entry={data.compare_session} />

                  <SessionCompareMetrics metrics={data.compare_metrics} />

                  <SessionCompareOutcomeTape
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.outcome_tape_title'),
                      sessionA: data.current_session?.session_label ?? t('session.detail.session_active'),
                      sessionB: data.compare_session.session_label,
                      empty: t('session.compare.outcome_tape_empty'),
                    }}
                  />

                  <SessionCompareKillsDonut
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.kills_donut_title'),
                      sessionA: data.current_session?.session_label ?? t('session.detail.session_active'),
                      sessionB: data.compare_session.session_label,
                      empty: t('session.compare.kills_donut_empty'),
                    }}
                  />

                  <SessionComparePerfProgression
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.perf_progression_title'),
                      sessionA: data.current_session?.session_label ?? t('session.detail.session_active'),
                      sessionB: data.compare_session.session_label,
                      empty: t('session.compare.perf_progression_empty'),
                    }}
                    height={240}
                  />

                  <SessionCompareSkillProgression
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.skill_progression_title'),
                      sessionA: data.current_session?.session_label ?? t('session.detail.session_active'),
                      sessionB: data.compare_session.session_label,
                      empty: t('session.compare.skill_progression_empty'),
                    }}
                    height={240}
                  />

                  <SessionCompareOCDR
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.ocdr_title'),
                      empty: t('session.compare.ocdr_empty'),
                    }}
                  />

                  <SessionCompareEngagement
                    sessionA={data.current_session}
                    sessionB={data.compare_session}
                    labels={{
                      title: t('session.compare.engagement_title'),
                      progressionTitle: t('session.compare.engagement_progression_title'),
                      sessionA: data.current_session?.session_label ?? t('session.detail.session_active'),
                      sessionB: data.compare_session.session_label,
                      empty: t('session.compare.engagement_empty'),
                    }}
                    height={200}
                  />

                  <SessionMatchesTable
                    matches={data.compare_matches ?? []}
                    playerSlug={playerSlug}
                    variant="compact"
                  />
                </>
              ) : isCompareLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Spinner size="md" label={t('session.detail.loading')} />
                </div>
              ) : (
                <EmptyStateNotice
                  title={t('session.detail.no_compare_title')}
                  description={t('session.detail.drawer_no_compare')}
                />
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function ChevronIcon({ direction }: { direction: 'left' | 'right' }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {direction === 'left' ? (
        <polyline points="15 18 9 12 15 6" />
      ) : (
        <polyline points="9 18 15 12 9 6" />
      )}
    </svg>
  )
}
