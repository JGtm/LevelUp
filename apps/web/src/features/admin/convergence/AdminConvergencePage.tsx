/**
 * AdminConvergencePage — onglet Convergence : backlog d'enrichissement par
 * joueur (deltas vs visite précédente, action « Converger » par joueur) +
 * compteurs post-sync du dernier cycle. Pas de polling continu (le backlog
 * résout les DBs de tous les joueurs) : staleTime 60 s + refetch au focus.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { AdminConvergenceReport, SchedulerSnapshot } from '@/lib/api/types'
import { useMonitoringConvergence, useMonitoringScheduler } from '../monitoring/queries'
import { AdminKpi } from '../components/AdminKpi'
import { counterDelta, type CountersSnapshot } from '../countersTrend'
import { useCounterSnapshot } from '../useCounterSnapshot'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import { adminRelativeTime, adminAbsoluteTime, type AdminLocale } from '../format'
import { describeLastCycle, lastCycleLabelKey } from '../actionJournalDisplay'
import { ConvergencePlayersTable } from './ConvergencePlayersTable'
import { PostSyncMatrix } from './PostSyncMatrix'
import { PostSyncTimeline } from './PostSyncTimeline'
import { SectionHeader } from '../components/SectionHeader'

const CONVERGENCE_SNAPSHOT_KEY = 'admin-convergence-snapshot'

export function AdminConvergencePage() {
  const { data, isLoading, isError } = useMonitoringConvergence()
  const scheduler = useMonitoringScheduler()
  const tA = useAdminT()
  const locale = useAdminLocale()

  // Baseline roulante (hook canonique A8.2) : delta vs run precedent.
  const previous = useCounterSnapshot(CONVERGENCE_SNAPSHOT_KEY, data?.generated_at, () =>
    buildConvergenceSnapshot(data!),
  )

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError || !data) {
    return <p className="text-sm text-destructive">{tA('admin.convergence.unavailable')}</p>
  }

  const totals = sumBacklog(data)
  const allGreen = totals.enrichment + totals.psa + totals.events + totals.weapons === 0

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <div>
          <SectionHeader title={tA('admin.convergence.backlog_section')} />
          <p className="mt-0.5 max-w-2xl text-xs text-muted-foreground">
            {tA('admin.convergence.backlog_desc')} (horizon : {data.horizon})
          </p>
        </div>
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <AdminKpi label={tA('admin.convergence.kpi_enrichment')} value={totals.enrichment} accent={totals.enrichment > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'enrichment', totals.enrichment)} />
          <AdminKpi label={tA('admin.convergence.kpi_psa')} value={totals.psa} valueSuffix={totals.psaCapped ? '+' : ''} accent={totals.psa > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'psa', totals.psa)} />
          <AdminKpi label={tA('admin.convergence.kpi_events')} value={totals.events} valueSuffix={totals.eventsCapped ? '+' : ''} accent={totals.events > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'events', totals.events)} />
          <AdminKpi label={tA('admin.convergence.kpi_weapons')} value={totals.weapons} valueSuffix={totals.weaponsCapped ? '+' : ''} accent={totals.weapons > 0 ? 'warning' : 'success'} delta={counterDelta(previous, 'weapons', totals.weapons)} />
        </div>
      </section>

      <section className="space-y-3">
        <SectionHeader title={tA('admin.convergence.players_section')} />
        {allGreen && (data.players ?? []).every((p) => !p.check_error) ? (
          <EmptyStateNotice
            title={tA('admin.convergence.all_green_title')}
            description={tA('admin.convergence.all_green_desc')}
          />
        ) : (
          <ConvergencePlayersTable report={data} />
        )}
      </section>

      <CaughtUpPanel data={data} players={scheduler.data?.snapshot?.players ?? []} tA={tA} />

      <section className="space-y-3">
        <SectionHeader title={tA('admin.convergence.timeline_section')} />
        <LastCycleBanner snapshot={scheduler.data?.snapshot} tA={tA} locale={locale} />
        <PostSyncTimeline players={scheduler.data?.snapshot?.players ?? []} />
      </section>

      <section className="space-y-3">
        <SectionHeader title={tA('admin.convergence.postsync_section')} />
        <PostSyncMatrix players={scheduler.data?.snapshot?.players ?? []} />
      </section>
    </div>
  )
}

/**
 * LastCycleBanner — horodatage du dernier cycle post-sync, réhydraté au boot
 * (C1). Distingue trois états : aucune donnée connue (jamais de cycle), cycle en
 * direct (SinceBoot), et cycle précédent daté (snapshot réhydraté d'avant le
 * redémarrage). last_cycle_at à zéro (temps Go) sérialise en « 0001-… ».
 */
function LastCycleBanner({
  snapshot,
  tA,
  locale,
}: {
  snapshot: SchedulerSnapshot | undefined
  tA: TAdmin
  locale: AdminLocale
}) {
  const display = describeLastCycle(snapshot)
  if (display.kind === 'none') {
    return <p className="text-xs text-muted-foreground/70">{tA('admin.convergence.no_cycle_ever')}</p>
  }
  return (
    <p className="text-xs text-muted-foreground" title={adminAbsoluteTime(display.at, locale)}>
      {tA(lastCycleLabelKey(display.kind))} : {adminRelativeTime(display.at, locale)}
    </p>
  )
}

/**
 * CaughtUpPanel — « rattrapé par la convergence » : dernier passage (somme
 * des compteurs post-sync par joueur) + cumuls depuis le boot (expvar).
 */
function CaughtUpPanel({
  data,
  players,
  tA,
}: {
  data: AdminConvergenceReport
  players: import('@/lib/api/types').SchedulerPlayerOutcome[]
  tA: TAdmin
}) {
  const lastEvents = players.reduce((acc, p) => acc + (p.post_sync?.converged_events ?? 0), 0)
  const lastPSA = players.reduce((acc, p) => acc + (p.post_sync?.converged_psa ?? 0), 0)
  const lastWeapons = players.reduce((acc, p) => acc + (p.post_sync?.weapon_kills_processed ?? 0), 0)
  const boot = data.totals_since_boot

  const rows: Array<{ label: string; last: number; sinceBoot: number }> = [
    { label: tA('admin.convergence.caught_events'), last: lastEvents, sinceBoot: boot.events_processed },
    { label: tA('admin.convergence.caught_weapons'), last: lastWeapons, sinceBoot: boot.weapons_processed },
    { label: tA('admin.convergence.caught_psa'), last: lastPSA, sinceBoot: boot.psa_processed },
    { label: tA('admin.convergence.caught_aliases'), last: -1, sinceBoot: boot.aliases_upserted },
  ]

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.convergence.caught_up_section')} />
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {rows.map((r) => (
          <div key={r.label} className="rounded-md border px-3 py-2">
            <div className="text-xs text-muted-foreground">{r.label}</div>
            <div className="text-lg font-semibold tabular-nums text-foreground">
              {r.sinceBoot}
              <span className="ml-1 text-xs font-normal text-muted-foreground">
                {tA('admin.convergence.caught_since_boot')}
              </span>
            </div>
            {r.last >= 0 && (
              <div className="font-mono text-[11px] text-muted-foreground">
                +{r.last} {tA('admin.convergence.caught_last_cycle')}
              </div>
            )}
          </div>
        ))}
      </div>
    </section>
  )
}

function sumBacklog(data: AdminConvergenceReport) {
  const totals = {
    enrichment: 0,
    psa: 0,
    events: 0,
    weapons: 0,
    psaCapped: false,
    eventsCapped: false,
    weaponsCapped: false,
  }
  for (const p of data.players ?? []) {
    totals.enrichment += p.missing_enrichment
    totals.psa += p.missing_psa
    totals.events += p.missing_events
    totals.weapons += p.missing_weapons
    if (p.missing_psa >= data.horizon) totals.psaCapped = true
    if (p.missing_events >= data.horizon) totals.eventsCapped = true
    if (p.missing_weapons >= data.horizon) totals.weaponsCapped = true
  }
  return totals
}

function buildConvergenceSnapshot(data: AdminConvergenceReport): CountersSnapshot {
  const t = sumBacklog(data)
  return { enrichment: t.enrichment, psa: t.psa, events: t.events, weapons: t.weapons }
}
