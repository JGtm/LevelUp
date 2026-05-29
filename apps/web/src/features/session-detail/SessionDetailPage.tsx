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
 *
 * Comparaison "côte à côte" : la session active (gauche) et la session comparée
 * (drawer, droite) affichent la MÊME pile de graphes (`SessionChartStack`), alignée.
 * Le profil de participation s'affiche en miroir (axe à droite à gauche / à gauche à
 * droite) pour un effet papillon symétrique.
 */
import { useEffect, useRef, useState } from 'react'
import { useParams, useSearch } from '@tanstack/react-router'

import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { useSoloFilterStore } from '@/stores/soloFilterStore'

import { useSessionDetailPage } from './queries'
import { useSessionT } from './_shared'
import { SessionSummaryCard } from './SessionSummaryCard'
import { SessionParamPills } from './SessionParamPills'
import { SessionMatchesTable } from './SessionMatchesTable'
import { SessionChartStack } from './SessionChartStack'
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

  // En-tête L3 sticky : il doit se coller SOUS la NavL2 (elle-même sticky top-0 dans
  // le même conteneur de scroll). On mesure la hauteur réelle de la NavL2 (sibling
  // précédent du root de la page dans PlayerLayout) et on l'applique en `top` — sinon
  // les deux se chevauchent à top-0 et le header L3 disparaît derrière la NavL2 au scroll.
  const rootRef = useRef<HTMLDivElement>(null)
  const [navHeight, setNavHeight] = useState(0)
  useEffect(() => {
    const root = rootRef.current
    if (!root) return
    const nav = root.previousElementSibling as HTMLElement | null
    if (!nav) return
    const update = () => setNavHeight(nav.offsetHeight)
    update()
    const ro = new ResizeObserver(update)
    ro.observe(nav)
    return () => ro.disconnect()
  }, [data])

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
      ref={rootRef}
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
            {/* En-tete session "L3" : sticky sous la NavL2 (top = hauteur NavL2 mesurée),
                bleed horizontal via -mx-6 pour s'aligner sur les bords de la colonne. */}
            <div
              className="sticky z-20 -mx-6 -mt-6 flex flex-col gap-3 border-b border-border bg-background px-6 py-3 sm:flex-row sm:items-center sm:justify-between"
              style={{ top: navHeight }}
            >
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

            <SessionChartStack
              entry={data.current_session}
              matches={data.matches}
              participationSide="right"
              participationColor="compare-a"
            />

            {/* Tableau "Détail des matchs" — hors bloc/Card (juste un titre + le tableau). */}
            <div className="space-y-3">
              <h2 className="text-base font-semibold text-foreground">{t('session.detail.matches_card')}</h2>
              <SessionMatchesTable
                matches={data.matches}
                playerSlug={playerSlug}
                variant={drawerOpen ? 'compact' : 'full'}
              />
            </div>
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

              {/* Contenu session comparée — même pile de graphes que la vue principale
                  (côte à côte), profil de participation en miroir (axe à gauche). */}
              {data.compare_session ? (
                <>
                  <SessionSummaryCard entry={data.compare_session} />

                  <SessionCompareMetrics metrics={data.compare_metrics} />

                  <SessionChartStack
                    entry={data.compare_session}
                    matches={data.compare_matches ?? []}
                    dense
                    participationSide="left"
                    participationColor="compare-b"
                  />

                  <div className="space-y-3">
                    <h3 className="text-sm font-semibold text-foreground">{t('session.detail.matches_card')}</h3>
                    <SessionMatchesTable
                      matches={data.compare_matches ?? []}
                      playerSlug={playerSlug}
                      variant="compact"
                    />
                  </div>
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
