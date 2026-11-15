/**
 * ApiHaloSection — latences API Halo par type d'appel depuis le boot
 * (films, stats, skill…) + buckets d'erreurs (429/auth/5xx/réseau). Identifie
 * si le goulot de sync est côté API (343) ou côté app.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SortableTh } from '@/components/ui/sortable-th'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { useColumnSort, type SortColumn } from '@/lib/table/useColumnSort'
import type { AdminPerfStats, PerfCallStats, PerfPlayerCallStats } from '@/lib/api/types'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

type CallSortKey = 'name' | 'count' | 'avg_ms' | 'max_ms' | 'sum_ms' | 'errors'

const CALL_SORT_COLUMNS: Record<CallSortKey, SortColumn<PerfCallStats>> = {
  name: { value: (c) => c.name, firstDir: 'asc' },
  count: { value: (c) => c.count },
  avg_ms: { value: (c) => c.avg_ms },
  max_ms: { value: (c) => c.max_ms },
  sum_ms: { value: (c) => c.sum_ms },
  errors: { value: (c) => c.errors ?? 0 },
}

export function ApiHaloSection({ perf }: { perf: AdminPerfStats | undefined }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  // I16 : tri client — table entièrement chargée (perf boot), aucun tri actif par défaut (ordre serveur conservé).
  const sort = useColumnSort(perf?.api_calls ?? [], CALL_SORT_COLUMNS)
  const sortedCalls = sort.rows

  return (
    <section className="space-y-3">
      <SectionHeader title={tA('admin.api.section')} />
      {!perf || (perf.api_calls?.length ?? 0) === 0 ? (
        <EmptyStateNotice title={tA('admin.api.empty')} description="" />
      ) : (
        <>
          <div className="flex flex-wrap gap-2">
            <BucketChip label={tA('admin.api.bucket_429')} value={perf.api_buckets.rate_limited_429} bad />
            <BucketChip label={tA('admin.api.bucket_auth')} value={perf.api_buckets.auth} bad />
            <BucketChip label={tA('admin.api.bucket_5xx')} value={perf.api_buckets.server_5xx} />
            <BucketChip label={tA('admin.api.bucket_network')} value={perf.api_buckets.network} />
          </div>
          <div className="overflow-x-auto rounded-md border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
                  <SortableTh label={tA('admin.api.col_call')} {...sort.header('name')} className="px-3 py-2 font-medium" />
                  <SortableTh label={tA('admin.api.col_count')} {...sort.header('count')} className="px-3 py-2 text-right font-medium" />
                  <SortableTh label={tA('admin.api.col_avg')} {...sort.header('avg_ms')} className="px-3 py-2 text-right font-medium" tooltip={tA('admin.api.col_avg_tooltip')} />
                  <SortableTh label={tA('admin.api.col_max')} {...sort.header('max_ms')} className="px-3 py-2 text-right font-medium" tooltip={tA('admin.api.col_max_tooltip')} />
                  <SortableTh label={tA('admin.api.col_total')} {...sort.header('sum_ms')} className="px-3 py-2 text-right font-medium" tooltip={tA('admin.api.col_total_tooltip')} />
                  <SortableTh label={tA('admin.api.col_errors')} {...sort.header('errors')} className="px-3 py-2 text-right font-medium" />
                </tr>
              </thead>
              <tbody>
                {sortedCalls.map((c) => (
                  <tr key={c.name} className="border-b last:border-b-0 hover:bg-muted/30">
                    <td className="px-3 py-2 font-mono text-xs text-foreground">{c.name}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">{c.count}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-foreground">{formatDurationMs(c.avg_ms, locale)}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">{formatDurationMs(c.max_ms, locale)}</td>
                    <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">{formatDurationMs(c.sum_ms, locale)}</td>
                    <td
                      className="px-3 py-2 text-right font-mono text-xs tabular-nums"
                      style={(c.errors ?? 0) > 0 ? { color: tokenCssVar('destructive') } : undefined}
                    >
                      {c.errors ?? 0}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(perf.api_by_player?.length ?? 0) > 0 && <ApiByPlayerTable perf={perf} />}
        </>
      )}
    </section>
  )
}

type PlayerCallSortKey = 'player' | 'call' | 'count' | 'avg_ms' | 'max_ms' | 'errors'

const PLAYER_CALL_SORT_COLUMNS: Record<PlayerCallSortKey, SortColumn<PerfPlayerCallStats>> = {
  player: { value: (s) => s.player, firstDir: 'asc' },
  call: { value: (s) => s.call, firstDir: 'asc' },
  count: { value: (s) => s.count },
  avg_ms: { value: (s) => s.avg_ms },
  max_ms: { value: (s) => s.max_ms },
  errors: { value: (s) => s.errors },
}

/** Sous-tableau des appels API attribuables par joueur (erreurs desc). */
function ApiByPlayerTable({ perf }: { perf: AdminPerfStats }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  const sort = useColumnSort(perf.api_by_player ?? [], PLAYER_CALL_SORT_COLUMNS)
  const sortedStats = sort.rows

  return (
    <div className="space-y-1.5">
      <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.api.by_player_section')}
      </h4>
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <SortableTh label={tA('admin.api.col_player')} {...sort.header('player')} className="px-3 py-2 font-medium" />
              <SortableTh label={tA('admin.api.col_call')} {...sort.header('call')} className="px-3 py-2 font-medium" />
              <SortableTh label={tA('admin.api.col_count')} {...sort.header('count')} className="px-3 py-2 text-right font-medium" />
              <SortableTh label={tA('admin.api.col_avg')} {...sort.header('avg_ms')} className="px-3 py-2 text-right font-medium" tooltip={tA('admin.api.col_avg_tooltip')} />
              <SortableTh label={tA('admin.api.col_max')} {...sort.header('max_ms')} className="px-3 py-2 text-right font-medium" tooltip={tA('admin.api.col_max_tooltip')} />
              <SortableTh label={tA('admin.api.col_errors')} {...sort.header('errors')} className="px-3 py-2 text-right font-medium" />
            </tr>
          </thead>
          <tbody>
            {sortedStats.map((s, i) => (
              <tr key={`${s.player}-${s.call}-${i}`} className="border-b last:border-b-0 hover:bg-muted/30">
                <td className="px-3 py-2 font-mono text-xs text-foreground">{s.player}</td>
                <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{s.call}</td>
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">{s.count}</td>
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-foreground">{formatDurationMs(s.avg_ms, locale)}</td>
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">{formatDurationMs(s.max_ms, locale)}</td>
                <td
                  className="px-3 py-2 text-right font-mono text-xs tabular-nums"
                  style={s.errors > 0 ? { color: tokenCssVar('destructive') } : undefined}
                >
                  {s.errors}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function BucketChip({ label, value, bad }: { label: string; value: number; bad?: boolean }) {
  const color = value > 0 ? tokenCssVar(bad ? 'destructive' : 'warning') : undefined
  return (
    <span className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground">
      {label}
      <span className="font-mono tabular-nums" style={color ? { color } : undefined}>
        {value}
      </span>
    </span>
  )
}
