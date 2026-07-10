/**
 * AdminSyncPage — onglet Sync & Jobs : snapshot du scheduler auto-sync
 * (dernier cycle, joueurs, alertes zero-insert), historique des cycles depuis
 * le boot et jobs asynchrones récents. Polling 30 s, accéléré à 5 s quand un
 * job sync est en vol.
 */
import { useQueryClient } from '@tanstack/react-query'

import { EmptyStateCard } from '@/components/ui/empty-state'
import { KpiCard } from '@/components/cards/KpiCard'
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
import { TokenHealthSection } from '../sections/TokenHealthSection'
import { SyncPlayersTable } from './SyncPlayersTable'
import { SyncCycleHistory } from './SyncCycleHistory'
import { AdminJobsTable } from './AdminJobsTable'
import { WatcherSection } from './WatcherSection'
import { ApiHaloSection } from './ApiHaloSection'
import { AdminSyncSettingsSection } from './AdminSyncSettingsSection'

export function AdminSyncPage() {
  const queryClient = useQueryClient()
  const jobsQuery = useAdminJobs(20)
  // Cadence accélérée pendant qu'un job de sync (cycle forcé ou sync delta)
  // est actif — l'état des joueurs bouge pendant le cycle.
  const syncJobActive = (jobsQuery.data?.jobs ?? []).some(
    (j) =>
      (j.status === 'running' || j.status === 'queued') &&
      (j.job_type === 'forced_sync_cycle' || j.job_type === 'delta_sync_all' || j.job_type === 'initial_sync'),
  )
  const { data, isLoading, isError } = useMonitoringScheduler({ fastPoll: syncJobActive })
  const perf = usePerfStats()
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError) {
    return <p className="text-sm text-destructive">{tA('admin.sync.scheduler_unavailable')}</p>
  }

  return (
    <div className="space-y-8">
      <section className="space-y-3">
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.sync.scheduler_section')}
        </h3>
        <SchedulerSummary data={data} tA={tA} locale={locale} />
        <AdminQuickActions
          onAnyActionSettled={() => {
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringScheduler })
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringOverview })
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringJobs })
          }}
        />
      </section>

      {data?.available && data.snapshot && (
        <>
          <section className="space-y-3">
            <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
              {tA('admin.sync.players_section')}
            </h3>
            <SyncPlayersTable
              players={data.snapshot.players ?? []}
              zeroInsertThreshold={data.zero_insert_warn_threshold}
            />
          </section>

          <section className="space-y-3">
            <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
              {tA('admin.sync.history_section')}{' '}
              <span className="normal-case font-normal">({tA('admin.sync.history_since_boot')})</span>
            </h3>
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
        <h3 className="text-sm font-medium uppercase tracking-wide text-muted-foreground">
          {tA('admin.jobs.section')}
        </h3>
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
      <SummaryCell
        label={tA('admin.sync.last_cycle')}
        value={adminRelativeTime(snap.last_cycle_at, locale)}
        title={adminAbsoluteTime(snap.last_cycle_at, locale)}
      />
      <SummaryCell label={tA('admin.sync.interval')} value={formatIntervalMinutes(snap.interval_minutes, locale)} />
      <SummaryCell label={tA('admin.sync.pool')} value={String(snap.pool_size)} />
      <SummaryCell label={tA('admin.sync.summary_synced')} value={String(res?.synced ?? 0)} good={(res?.synced ?? 0) > 0} />
      <SummaryCell label={tA('admin.sync.summary_skipped')} value={String(res?.skipped ?? 0)} />
      <SummaryCell
        label={tA('admin.sync.summary_failed')}
        value={String(res?.failed ?? 0)}
        bad={(res?.failed ?? 0) > 0}
        sub={res ? formatDurationMs(res.duration_ns / 1_000_000, locale) : undefined}
      />
    </div>
  )
}

function SummaryCell({
  label,
  value,
  sub,
  title,
  good,
  bad,
}: {
  label: string
  value: string
  sub?: string
  title?: string
  good?: boolean
  bad?: boolean
}) {
  return (
    <KpiCard accent={bad ? 'destructive' : good ? 'success' : undefined} className="h-full">
      <div className="p-3">
        <div className="text-xs text-muted-foreground">{label}</div>
        <div className="mt-0.5 text-lg font-semibold tabular-nums text-foreground" title={title}>
          {value}
        </div>
        {sub && <div className="text-xs text-muted-foreground">{sub}</div>}
      </div>
    </KpiCard>
  )
}
