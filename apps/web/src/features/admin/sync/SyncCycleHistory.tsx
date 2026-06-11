/**
 * SyncCycleHistory — rangées compactes multi-track des derniers cycles
 * (marqueur carré coloré par état, trigger, compteurs, durée). Plus récent en
 * premier (ordre serveur). Flat hard-edge.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { SchedulerCycleRecord } from '@/lib/api/types'
import { adminAbsoluteTime, adminRelativeTime, formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'

export function SyncCycleHistory({ history }: { history: SchedulerCycleRecord[] }) {
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (!history.length) {
    return <p className="text-sm text-muted-foreground">{tA('admin.sync.never_ran_title')}</p>
  }

  return (
    <div className="rounded-md border">
      {history.map((c, i) => {
        const marker =
          c.failed > 0 ? tokenCssVar('destructive') : c.synced > 0 ? tokenCssVar('success') : tokenCssVar('divergent-neutral')
        const blockedPct = c.duration_ms > 0 ? Math.round((c.blocked_ms / c.duration_ms) * 100) : 0
        return (
          <div
            key={`${c.at}-${i}`}
            className="space-y-1 border-b px-3 py-2 text-xs last:border-b-0 hover:bg-muted/30"
          >
            <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
              <span aria-hidden className="h-2.5 w-2.5 flex-none" style={{ backgroundColor: marker }} />
              <span
                className="w-24 flex-none font-medium text-foreground"
                title={adminAbsoluteTime(c.at, locale)}
              >
                {adminRelativeTime(c.at, locale)}
              </span>
              <span className="w-14 flex-none uppercase tracking-wide text-muted-foreground">
                {c.trigger === 'tick' ? tA('admin.sync.trigger_tick') : tA('admin.sync.trigger_manual')}
              </span>
              <span className="font-mono tabular-nums text-muted-foreground">
                {tA('admin.sync.summary_total')} {c.total}
              </span>
              <span className="font-mono tabular-nums" style={{ color: c.synced > 0 ? tokenCssVar('success') : undefined }}>
                {tA('admin.sync.summary_synced')} {c.synced}
              </span>
              <span className="font-mono tabular-nums text-muted-foreground">
                {tA('admin.sync.summary_skipped')} {c.skipped}
              </span>
              <span
                className="font-mono tabular-nums"
                style={{ color: c.failed > 0 ? tokenCssVar('destructive') : undefined }}
              >
                {tA('admin.sync.summary_failed')} {c.failed}
              </span>
              <span className="ml-auto font-mono tabular-nums text-muted-foreground">
                {formatDurationMs(c.duration_ms, locale)}
              </span>
            </div>
            {/* Corrélation charge : indispo lectures (B-swap) / temps API / écritures / 503. */}
            {(c.blocked_ms > 0 || c.api_ms > 0 || c.persist_write_ms > 0 || c.reads_rejected > 0) && (
              <div className="flex flex-wrap items-center gap-x-4 gap-y-0.5 pl-6 font-mono text-[11px] text-muted-foreground">
                <span style={blockedPct >= 20 && c.blocked_ms >= 5000 ? { color: tokenCssVar('warning') } : undefined}>
                  {tA('admin.sync.col_blocked')} {formatDurationMs(c.blocked_ms, locale)} ({blockedPct}%)
                  {c.swap_count > 0 ? ` · ${c.swap_count} swaps` : ''}
                </span>
                <span>
                  {tA('admin.sync.col_api_time')} {formatDurationMs(c.api_ms, locale)}
                </span>
                <span>
                  {tA('admin.sync.col_writes')} {formatDurationMs(c.persist_write_ms, locale)}
                </span>
                {c.reads_rejected > 0 && (
                  <span style={{ color: tokenCssVar('destructive') }}>
                    {tA('admin.sync.rejected_short')} ×{c.reads_rejected}
                  </span>
                )}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
