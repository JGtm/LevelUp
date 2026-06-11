/**
 * JobProgressInline — suivi inline d'un job asynchrone : badge d'état,
 * étape courante, barre de progression (si progress_pct), erreur détaillée.
 * Polling 3 s via useJobStatus (s'arrête seul aux statuts terminaux) ;
 * onTerminal est déclenché UNE fois à l'arrivée en état terminal.
 */
import { useEffect, useRef } from 'react'

import { useJobStatus } from '@/features/setup/queries'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AsyncJobStatus } from '@/lib/api/types'
import { jobStatusToAdminStatus, jobTypeLabel } from '../statusDisplay'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { StatusBadge } from './StatusBadge'

const TERMINAL = new Set(['succeeded', 'failed', 'cancelled', 'interrupted'])

interface JobProgressInlineProps {
  jobId: string
  onTerminal?: (job: AsyncJobStatus) => void
}

export function JobProgressInline({ jobId, onTerminal }: JobProgressInlineProps) {
  const { data: job } = useJobStatus(jobId, !!jobId)
  const tA = useAdminT()
  const locale = useAdminLocale()

  // Déclenche onTerminal une seule fois par jobId (invalidations + toast côté caller).
  const firedForRef = useRef<string | null>(null)
  useEffect(() => {
    if (!job || firedForRef.current === jobId) return
    if (TERMINAL.has(job.status)) {
      firedForRef.current = jobId
      onTerminal?.(job)
    }
  }, [job, jobId, onTerminal])

  if (!job) return null

  const durationMs =
    job.started_at && job.finished_at
      ? new Date(job.finished_at).getTime() - new Date(job.started_at).getTime()
      : undefined

  return (
    <div className="mt-2 space-y-1.5 rounded-sm border border-border bg-muted/40 px-3 py-2">
      <div className="flex flex-wrap items-center gap-2 text-xs">
        <StatusBadge status={jobStatusToAdminStatus(job.status)} />
        <span className="text-muted-foreground">{jobTypeLabel(job.job_type, locale)}</span>
        {job.current_step && !TERMINAL.has(job.status) && (
          <span className="text-muted-foreground">
            {tA('admin.job.step')} : {job.current_step}
          </span>
        )}
        {durationMs !== undefined && (
          <span className="font-mono text-muted-foreground">{formatDurationMs(durationMs, locale)}</span>
        )}
      </div>
      {typeof job.progress_pct === 'number' && !TERMINAL.has(job.status) && (
        <div className="h-1 w-full bg-muted" role="progressbar" aria-valuenow={job.progress_pct}>
          <div
            className="h-1 transition-all"
            style={{
              width: `${Math.min(100, Math.max(0, job.progress_pct))}%`,
              backgroundColor: tokenCssVar('info'),
            }}
          />
        </div>
      )}
      {job.error?.message && (
        <p className="text-xs" style={{ color: tokenCssVar('destructive') }}>
          {job.error.message}
        </p>
      )}
    </div>
  )
}
