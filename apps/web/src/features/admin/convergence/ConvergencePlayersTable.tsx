/**
 * ConvergencePlayersTable — backlog par joueur avec action « Converger »
 * (job player_convergence suivi inline). Compteurs plafonnés affichés « N+ ».
 */
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'

import { queryKeys } from '@/lib/query/keys'
import { apiErrorMessage } from '@/lib/api/client'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminConvergenceReport, AsyncJobStatus, PlayerConvergenceReport } from '@/lib/api/types'
import { conflictJobId } from '../monitoring/mutations'
import { useRunPlayerConvergence } from '../data-quality/mutations'
import { AdminActionButton } from '../components/AdminActionButton'
import { useAdminT } from '../useAdminText'

export function ConvergencePlayersTable({ report }: { report: AdminConvergenceReport }) {
  const tA = useAdminT()
  const queryClient = useQueryClient()
  const runConvergence = useRunPlayerConvergence()

  function invalidateAfterConvergence() {
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringConvergence })
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringScheduler })
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminMonitoringOverview })
    void queryClient.invalidateQueries({ queryKey: queryKeys.adminDataQuality })
  }

  async function launch(p: PlayerConvergenceReport): Promise<string | null> {
    try {
      const job = (await runConvergence.mutateAsync(p.player_slug || p.gamertag)) as AsyncJobStatus
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

  const cap = (n: number) => `${n}${n >= report.horizon ? '+' : ''}`

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.convergence.col_player')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.convergence.kpi_enrichment')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.convergence.kpi_psa')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.convergence.kpi_events')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.convergence.kpi_weapons')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.convergence.col_total')}</th>
            <th className="px-3 py-2" />
          </tr>
        </thead>
        <tbody>
          {(report.players ?? []).map((p) => {
            const total = p.missing_enrichment + p.missing_psa + p.missing_events + p.missing_weapons
            return (
              <tr key={p.xuid || p.gamertag} className="border-b align-top last:border-b-0 hover:bg-muted/30">
                <td className="px-3 py-2 font-medium text-foreground">
                  {p.gamertag}
                  {p.check_error && (
                    <div className="text-xs font-normal" style={{ color: tokenCssVar('destructive') }}>
                      {p.check_error}
                    </div>
                  )}
                </td>
                <BacklogCell value={p.missing_enrichment} display={String(p.missing_enrichment)} />
                <BacklogCell value={p.missing_psa} display={cap(p.missing_psa)} />
                <BacklogCell value={p.missing_events} display={cap(p.missing_events)} />
                <BacklogCell value={p.missing_weapons} display={cap(p.missing_weapons)} />
                <td className="px-3 py-2 text-right font-mono text-sm font-semibold tabular-nums text-foreground">
                  {total}
                </td>
                <td className="px-3 py-2 text-right">
                  {!p.check_error && (
                    <AdminActionButton
                      label={tA('admin.convergence.run_player')}
                      confirmMessage={tA('admin.convergence.run_player_confirm')}
                      busyLabel={tA('admin.job.in_progress')}
                      onRun={() => launch(p)}
                      onJobTerminal={(job) => {
                        if (job.status === 'succeeded') toast.success(tA('admin.actions.done'))
                        else toast.error(job.error?.message ?? tA('admin.actions.failed'))
                        invalidateAfterConvergence()
                      }}
                    />
                  )}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function BacklogCell({ value, display }: { value: number; display: string }) {
  return (
    <td
      className="px-3 py-2 text-right font-mono text-xs tabular-nums"
      style={value > 0 ? { color: tokenCssVar('warning') } : undefined}
    >
      {value > 0 ? display : <span className="text-muted-foreground">0</span>}
    </td>
  )
}
