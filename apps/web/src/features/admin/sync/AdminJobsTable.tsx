/**
 * AdminJobsTable — jobs asynchrones récents du JobStore (type traduit, statut
 * badge, joueur, démarrage relatif, durée, étape/erreur courante).
 */
import { useMemo, useState } from 'react'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SortableTh } from '@/components/ui/sortable-th'
import type { AsyncJobStatus } from '@/lib/api/types'
import { jobStatusToAdminStatus, jobTypeLabel } from '../statusDisplay'
import { adminAbsoluteTime, adminRelativeTime, formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { StatusBadge } from '../components/StatusBadge'

type JobSortKey = 'job_type' | 'status' | 'player_slug' | 'started_at' | 'duration'

function jobDurationMs(job: AsyncJobStatus): number | null {
  if (!job.started_at || !job.finished_at) return null
  return new Date(job.finished_at).getTime() - new Date(job.started_at).getTime()
}

function jobRawValue(job: AsyncJobStatus, key: JobSortKey): string | number | null {
  switch (key) {
    case 'job_type':
      return job.job_type
    case 'status':
      return job.status
    case 'player_slug':
      return job.player_slug && job.player_slug !== '_all' ? job.player_slug : null
    case 'started_at':
      return job.started_at ? Date.parse(job.started_at) : null
    case 'duration':
      return jobDurationMs(job)
  }
}

function compareJobs(a: AsyncJobStatus, b: AsyncJobStatus, key: JobSortKey, dir: 'asc' | 'desc'): number {
  const va = jobRawValue(a, key)
  const vb = jobRawValue(b, key)
  if (va == null && vb == null) return 0
  if (va == null) return 1
  if (vb == null) return -1
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

export function AdminJobsTable({ jobs }: { jobs: AsyncJobStatus[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  // I16 : tri CLIENT — liste de jobs récents entièrement chargée, aucun tri actif
  // par défaut (ordre serveur — plus récents en tête — conservé).
  const [sortKey, setSortKey] = useState<JobSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: JobSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'job_type' || key === 'status' || key === 'player_slug' ? 'asc' : 'desc')
    }
  }
  const sortedJobs = useMemo(() => {
    if (!sortKey) return jobs
    return [...jobs].sort((a, b) => compareJobs(a, b, sortKey, sortDir))
  }, [jobs, sortKey, sortDir])

  if (!jobs.length) {
    return <EmptyStateNotice title={tA('admin.jobs.empty_title')} description={tA('admin.jobs.empty_desc')} />
  }

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <SortableTh label={tA('admin.jobs.col_type')} active={sortKey === 'job_type'} dir={sortDir} onClick={() => toggleSort('job_type')} className="px-3 py-2 font-medium" />
            <SortableTh label={tA('admin.jobs.col_status')} active={sortKey === 'status'} dir={sortDir} onClick={() => toggleSort('status')} className="px-3 py-2 font-medium" />
            <SortableTh label={tA('admin.jobs.col_player')} active={sortKey === 'player_slug'} dir={sortDir} onClick={() => toggleSort('player_slug')} className="px-3 py-2 font-medium" />
            <SortableTh label={tA('admin.jobs.col_started')} active={sortKey === 'started_at'} dir={sortDir} onClick={() => toggleSort('started_at')} className="px-3 py-2 font-medium" />
            <SortableTh label={tA('admin.jobs.col_duration')} active={sortKey === 'duration'} dir={sortDir} onClick={() => toggleSort('duration')} className="px-3 py-2 text-right font-medium" />
          </tr>
        </thead>
        <tbody>
          {sortedJobs.map((job) => (
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
