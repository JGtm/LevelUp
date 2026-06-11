/**
 * AdminJobsTable — jobs asynchrones récents du JobStore (type traduit, statut
 * badge, joueur, démarrage relatif, durée, étape/erreur courante).
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { AsyncJobStatus } from '@/lib/api/types'
import { jobStatusToAdminStatus, jobTypeLabel } from '../statusDisplay'
import { adminAbsoluteTime, adminRelativeTime, formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { StatusBadge } from '../components/StatusBadge'

export function AdminJobsTable({ jobs }: { jobs: AsyncJobStatus[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (!jobs.length) {
    return <EmptyStateNotice title={tA('admin.jobs.empty_title')} description={tA('admin.jobs.empty_desc')} />
  }

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.jobs.col_type')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.jobs.col_status')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.jobs.col_player')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.jobs.col_started')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.jobs.col_duration')}</th>
          </tr>
        </thead>
        <tbody>
          {jobs.map((job) => (
            <tr key={job.job_id} className="border-b last:border-b-0 hover:bg-muted/30">
              <td className="px-3 py-2">
                <span className="font-medium text-foreground">{jobTypeLabel(job.job_type, locale)}</span>
                {(job.current_step || job.error?.message) && (
                  <div
                    className="max-w-[24rem] truncate text-xs text-muted-foreground"
                    title={job.error?.message ?? job.current_step ?? undefined}
                  >
                    {job.error?.message ?? job.current_step}
                  </div>
                )}
              </td>
              <td className="px-3 py-2">
                <StatusBadge status={jobStatusToAdminStatus(job.status)} />
              </td>
              <td className="px-3 py-2 font-mono text-xs text-muted-foreground">
                {job.player_slug && job.player_slug !== '_all' ? job.player_slug : '—'}
              </td>
              <td
                className="px-3 py-2 text-xs text-muted-foreground"
                title={adminAbsoluteTime(job.started_at ?? undefined, locale)}
              >
                {adminRelativeTime(job.started_at ?? undefined, locale)}
              </td>
              <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">
                {job.started_at && job.finished_at
                  ? formatDurationMs(
                      new Date(job.finished_at).getTime() - new Date(job.started_at).getTime(),
                      locale,
                    )
                  : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
