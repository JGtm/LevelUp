/**
 * AdminOverviewPage — Vue d'ensemble du dashboard monitoring : KPIs santé
 * pipeline (drill-down vers les onglets), actions rapides, dernier audit
 * data health. Polling 30 s (l'endpoint est zéro I/O DuckDB côté Go).
 */
import { useQueryClient } from '@tanstack/react-query'

import type { AdminMonitoringOverview } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useMonitoringOverview, useMonitoringResources } from '../monitoring/queries'
import { diskToken } from '../system/ResourcesSection'
import { useAdminT, useAdminLocale, type TAdmin } from '../useAdminText'
import { adminAbsoluteTime, adminRelativeTime, formatBytes, formatDurationMs, type AdminLocale } from '../format'
import { AdminKpi } from '../components/AdminKpi'
import { SectionHeader } from '../components/SectionHeader'
import { AdminQuickActions } from './AdminQuickActions'
import { CronsPanel } from './CronsPanel'
import { DataHealthPanel } from './DataHealthPanel'
import { DiagnosticPanel } from './DiagnosticPanel'
import { FreshnessPanel } from './FreshnessPanel'
import { WeaponCoveragePanel } from './WeaponCoveragePanel'

export function AdminOverviewPage() {
  const { data, isLoading, isError } = useMonitoringOverview()
  const resources = useMonitoringResources()
  const queryClient = useQueryClient()
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">…</p>
  }
  if (isError || !data) {
    return <p className="text-sm text-destructive">{tA('admin.overview.unavailable')}</p>
  }

  return (
    <div className="space-y-8">
      <DiagnosticPanel overview={data} />

      <OverviewKpiGrid data={data} tA={tA} locale={locale} disk={resources.data?.disk} />

      <section className="space-y-3">
        <SectionHeader title={tA('admin.overview.quick_actions')} />
        <AdminQuickActions
          onAnyActionSettled={() => {
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringOverview })
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringScheduler })
            queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringJobs })
          }}
        />
      </section>

      <FreshnessPanel />

      <CronsPanel />

      <section className="space-y-3">
        <SectionHeader title={tA('admin.overview.data_health_section')} />
        <DataHealthPanel dataHealth={data.data_health} tA={tA} locale={locale} />
      </section>

      <WeaponCoveragePanel />
    </div>
  )
}

// ─── Grille KPI ───────────────────────────────────────────────────────────────

function OverviewKpiGrid({
  data,
  tA,
  locale,
  disk,
}: {
  data: AdminMonitoringOverview
  tA: TAdmin
  locale: AdminLocale
  /** Verdict disque compact (A5.2) — undefined tant que /resources charge. */
  disk?: import('@/lib/api/types').AdminResourcesResponse['disk']
}) {
  const sched = data.scheduler
  const tokensNeedAction = data.tokens
    ? data.tokens.expired + data.tokens.absent + data.tokens.reauth
    : undefined
  const invariantsRan = data.invariants.runs_total > 0
  const dhRan = !!data.data_health

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <AdminKpi
        label={tA('admin.overview.kpi_last_cycle')}
        value={sched.available ? adminRelativeTime(sched.last_cycle_at, locale) : '—'}
        sub={sched.available ? undefined : tA('admin.sync.scheduler_unavailable')}
        title={adminAbsoluteTime(sched.last_cycle_at, locale)}
        accent={
          !sched.available || !sched.last_cycle_at
            ? undefined
            : sched.last_failed > 0
              ? 'destructive'
              : 'success'
        }
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_sync_failures')}
        value={String(sched.last_failed)}
        accent={sched.last_failed > 0 ? 'destructive' : 'success'}
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_jobs_running')}
        value={String(data.jobs.active_count)}
        accent={data.jobs.active_count > 0 ? 'info' : undefined}
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_in_flight')}
        value={String(sched.in_flight_claims)}
        accent={sched.in_flight_claims > 0 ? 'info' : undefined}
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_zero_insert')}
        value={String(sched.zero_insert_alerts)}
        accent={sched.zero_insert_alerts > 0 ? 'warning' : 'success'}
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_invariants')}
        value={
          invariantsRan
            ? `${data.invariants.fail_last} ${tA('admin.status_label.fail')} · ${data.invariants.warn_last} ${tA('admin.status_label.warn')}`
            : tA('admin.overview.invariants_never_ran')
        }
        compactValue={!invariantsRan}
        accent={
          !invariantsRan
            ? undefined
            : data.invariants.fail_last > 0
              ? 'destructive'
              : data.invariants.warn_last > 0
                ? 'warning'
                : 'success'
        }
        to="/admin/data"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_tokens')}
        value={tokensNeedAction === undefined ? '—' : String(tokensNeedAction)}
        sub={data.tokens_error}
        accent={
          tokensNeedAction === undefined ? undefined : tokensNeedAction > 0 ? 'destructive' : 'success'
        }
        to="/admin/sync"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_data_health')}
        value={dhRan ? String(data.data_health?.warnings_total ?? 0) : tA('admin.overview.never_ran')}
        compactValue={!dhRan}
        sub={dhRan ? adminRelativeTime(data.data_health?.ran_at, locale) : undefined}
        accent={
          !dhRan ? undefined : (data.data_health?.warnings_total ?? 0) > 0 ? 'warning' : 'success'
        }
      />
      <AdminKpi
        label={tA('admin.overview.kpi_freshness')}
        value={String(data.freshness_critical)}
        accent={data.freshness_critical > 0 ? 'destructive' : 'success'}
      />
      <AdminKpi
        label={tA('admin.overview.kpi_disk')}
        value={disk && disk.status !== 'unknown' ? formatBytes(disk.free_bytes, locale) : '—'}
        accent={disk ? diskToken(disk.status) : undefined}
        to="/admin/system"
      />
      <AdminKpi
        label={tA('admin.overview.kpi_http_5xx')}
        value={String(data.http.status_5xx)}
        sub={`4xx ${data.http.status_4xx} · 2xx ${data.http.status_2xx}`}
        accent={data.http.status_5xx > 0 ? 'destructive' : 'success'}
      />
      <AdminKpi
        label={tA('admin.overview.kpi_uptime')}
        value={formatDurationMs(data.server.uptime_s * 1000, locale)}
        sub={data.server.version}
        title={adminAbsoluteTime(data.server.started_at, locale)}
        accent="info"
      />
    </div>
  )
}


