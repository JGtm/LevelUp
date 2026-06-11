/**
 * AdminQuickActions — les deux actions rapides du dashboard : audit data
 * health (synchrone) et cycle auto-sync forcé (job suivi inline).
 * Réutilisé par la Vue d'ensemble et la page Sync & Jobs.
 */
import { toast } from 'sonner'

import { apiErrorMessage } from '@/lib/api/client'
import { conflictJobId, useRunDataHealthCheck, useRunSyncCycle } from '../monitoring/mutations'
import { useAdminT } from '../useAdminText'
import { AdminActionButton } from '../components/AdminActionButton'

interface AdminQuickActionsProps {
  /** Invalidations à la fin de chaque action (sync ou terminal de job). */
  onAnyActionSettled: () => void
}

export function AdminQuickActions({ onAnyActionSettled }: AdminQuickActionsProps) {
  const tA = useAdminT()
  const runHealthCheck = useRunDataHealthCheck()
  const runSyncCycle = useRunSyncCycle()

  async function handleRunHealthCheck(): Promise<string | null> {
    try {
      const res = await runHealthCheck.mutateAsync()
      toast.success(`${tA('admin.actions.done')} — ${tA('admin.dh.warnings_total')} : ${res.warnings_total}`)
      onAnyActionSettled()
    } catch (err) {
      toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed'))
    }
    return null
  }

  async function handleRunSyncCycle(): Promise<string | null> {
    try {
      const job = await runSyncCycle.mutateAsync()
      toast.success(tA('admin.actions.started'))
      return job.job_id
    } catch (err) {
      const existing = conflictJobId(err)
      if (existing) {
        toast.info(tA('admin.actions.conflict'))
        return existing
      }
      toast.error(apiErrorMessage(err) ?? tA('admin.actions.failed'))
      return null
    }
  }

  return (
    <div className="flex flex-wrap items-start gap-3">
      <AdminActionButton
        label={tA('admin.actions.run_health_check')}
        confirmMessage={tA('admin.actions.run_health_check_confirm')}
        busyLabel={tA('admin.job.in_progress')}
        onRun={handleRunHealthCheck}
      />
      <AdminActionButton
        label={tA('admin.actions.run_sync_cycle')}
        confirmMessage={tA('admin.actions.run_sync_cycle_confirm')}
        busyLabel={tA('admin.job.in_progress')}
        onRun={handleRunSyncCycle}
        onJobTerminal={(job) => {
          if (job.status === 'succeeded') toast.success(tA('admin.actions.done'))
          else toast.error(tA('admin.actions.failed'))
          onAnyActionSettled()
        }}
      />
    </div>
  )
}
