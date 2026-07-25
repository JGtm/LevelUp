/**
 * AdminSyncPage — onglet Sync & Jobs : snapshot du scheduler auto-sync
 * (dernier cycle, joueurs, alertes zero-insert), historique des cycles depuis
 * le boot et jobs asynchrones récents. Polling 30 s, accéléré à 5 s quand un
 * job sync est en vol.
 */
import { useQueryClient } from '@tanstack/react-query'

import { EmptyStateCard } from '@/components/ui/empty-state'
import { AdminKpi } from '../components/AdminKpi'
import { queryKeys } from '@/lib/query/keys'
import type { AdminSchedulerStatusResponse } from '@/lib/api/types'
import { useAdminJobs, useMonitoringScheduler, usePerfStats } from '../monitoring/queries'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import {
  adminAbsoluteTime,
  adminRelativeTime,
  formatDurationMs,
  formatIntervalMinutes,
  type AdminLocale,
} from '../format'
import { AdminQuickActions } from '../overview/AdminQuickActions'
import { AdminInitialSyncCard } from './AdminInitialSyncCard'
import { SyncActionsHelp } from './SyncActionsHelp'
import { TokenHealthSection } from '../sections/TokenHealthSection'
import { SyncPlayersTable } from './SyncPlayersTable'
import { SyncCycleHistory } from './SyncCycleHistory'
import { AdminJobsTable } from './AdminJobsTable'
import { WatcherSection } from './WatcherSection'
import { ApiHaloSection } from './ApiHaloSection'
import { AdminSyncSettingsSection } from './AdminSyncSettingsSection'
import { SectionHeader } from '../components/SectionHeader'

export function AdminSyncPage() {
  const queryClient = useQueryClient()
  const jobsQuery = useAdminJobs(20)
  // Cadence accélérée pendant qu'un job de sync (cycle forcé ou sync delta)
  // est actif — l'état des joueurs bouge pendant le cycle.
  const syncJobActive = (jobsQuery.data?.jobs ?? []).some(
    (j) =>
      (j.status === 'running' || j.status === 'queued') &&
      (j.job_type === 'forced_sync_cycle' ||
        j.job_type === 'delta_sync_all' ||
        j.job_type === 'initial_sync' ||
        j.job_type === 'admin_initial_sync'),
  )
  const { data, isLoading, isError } = useMonitoringScheduler({ fastPoll: syncJobActive })
  const perf = usePerfStats()
  const tA = useAdminT()
  const locale = useAdminLocale()

  // Invalidations partagées par les actions de sync (cycle forcé, audit, sync
  // initiale) quand elles se règlent — l'état des joueurs/jobs a bougé.
  const invalidateSyncViews = () => {
    queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringScheduler })
    queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringOverview })
    queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringJobs })
  }

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError) {
    return <p className="text-sm text-destructive">{tA('admin.sync.scheduler_unavailable')}</p>
  }

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <SectionHeader title={tA('admin.sync.scheduler_section')} />
        <SchedulerSummary data={data} tA={tA} locale={locale} />
        <AdminQuickActions onAnyActionSettled={invalidateSyncViews} />
        <SyncActionsHelp tA={tA} />
        <AdminInitialSyncCard onJobSettled={invalidateSyncViews} />
      </section>

      {data?.available && data.snapshot && (
        <>
          <section className="space-y-3">
            <SectionHeader title={tA('admin.sync.players_section')} />
            <SyncPlayersTable
              players={data.snapshot.players ?? []}
              zeroInsertThreshold={data.zero_insert_warn_threshold}
            />
          </section>

          <section className="space-y-3">
            <SectionHeader title={<>{tA('admin.sync.history_section')}{' '} <span className="normal-case font-normal">({tA('admin.sync.history_since_boot')})</span></>} />
            <SyncCycleHistory history={data.history ?? []} />
          </section>
        </>
      )}

      <WatcherSection />

      {/* Santé des tokens auth (ex-Système, A3.3) : les tokens conditionnent le
          moteur de sync — ils n'apparaissent plus qu'ici (État = verdict seul). */}
      <TokenHealthSection />

      <ApiHaloSection perf={perf.data} />

      {/* Paramètres de synchronisation (app_settings) rapatriés des Réglages
          utilisateur — auto-sync planifiée, watcher, backfill, amis par défaut. */}
      <AdminSyncSettingsSection />

      <section className="space-y-3">
        <SectionHeader title={tA('admin.jobs.section')} />
        <AdminJobsTable jobs={jobsQuery.data?.jobs ?? []} />
      </section>
    </div>
  )
}

// ─── Résumé scheduler ─────────────────────────────────────────────────────────

function SchedulerSummary({
  data,
  tA,
  locale,
}: {
  data: AdminSchedulerStatusResponse | undefined
  tA: TAdmin
  locale: AdminLocale
}) {
  if (!data?.available) {
    return (
      <EmptyStateCard
        title={tA('admin.sync.scheduler_unavailable')}
        description=""
      />
    )
  }
  const snap = data.snapshot
  if (!snap || !snap.last_cycle_at || snap.last_cycle_at.startsWith('0001-')) {
    return (
      <EmptyStateCard
        title={tA('admin.sync.never_ran_title')}
        description={tA('admin.sync.never_ran_desc')}
      />
    )
  }
  const res = snap.last_cycle_result
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-6">
      <AdminKpi
        size="sm"
        label={tA('admin.sync.last_cycle')}
        value={adminRelativeTime(snap.last_cycle_at, locale)}
        title={adminAbsoluteTime(snap.last_cycle_at, locale)}
      />
      <AdminKpi size="sm" label={tA('admin.sync.interval')} value={formatIntervalMinutes(snap.interval_minutes, locale)} />
      <AdminKpi size="sm" label={tA('admin.sync.pool')} value={String(snap.pool_size)} />
      <AdminKpi size="sm" label={tA('admin.sync.summary_synced')} value={String(res?.synced ?? 0)} accent={(res?.synced ?? 0) > 0 ? 'success' : undefined} />
      <AdminKpi size="sm" label={tA('admin.sync.summary_skipped')} value={String(res?.skipped ?? 0)} />
      <AdminKpi
        size="sm"
        label={tA('admin.sync.summary_failed')}
        value={String(res?.failed ?? 0)}
        accent={(res?.failed ?? 0) > 0 ? 'destructive' : undefined}
        sub={res ? formatDurationMs(res.duration_ns / 1_000_000, locale) : undefined}
      />
    </div>
  )
}
