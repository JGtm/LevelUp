/**
 * ApiHaloSection — latences API Halo par type d'appel depuis le boot
 * (films, stats, skill…) + buckets d'erreurs (429/auth/5xx/réseau). Identifie
 * si le goulot de sync est côté API (343) ou côté app.
 */
import { useMemo, useState } from 'react'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SortableTh } from '@/components/ui/sortable-th'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminPerfStats, PerfCallStats, PerfPlayerCallStats } from '@/lib/api/types'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

type CallSortKey = 'name' | 'count' | 'avg_ms' | 'max_ms' | 'sum_ms' | 'errors'

function callRawValue(c: PerfCallStats, key: CallSortKey): string | number {
  if (key === 'name') return c.name
  if (key === 'errors') return c.errors ?? 0
  return c[key]
}

function compareCallStats(a: PerfCallStats, b: PerfCallStats, key: CallSortKey, dir: 'asc' | 'desc'): number {
  const va = callRawValue(a, key)
  const vb = callRawValue(b, key)
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

/** Hook de tri client hand-rolled (I16) — table entièrement chargée (perf boot),
 *  aucun tri actif par défaut (ordre serveur conservé). */
function useCallSort() {
  const [sortKey, setSortKey] = useState<CallSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: CallSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'name' ? 'asc' : 'desc')
    }
  }
  return { sortKey, sortDir, toggleSort }
}

export function ApiHaloSection({ perf }: { perf: AdminPerfStats | undefined }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  const { sortKey, sortDir, toggleSort } = useCallSort()
  const sortedCalls = useMemo(() => {
    const calls = perf?.api_calls ?? []
    if (!sortKey) return calls
    return [...calls].sort((a, b) => compareCallStats(a, b, sortKey, sortDir))
  }, [perf?.api_calls, sortKey, sortDir])

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
                  <SortableTh label={tA('admin.api.col_call')} active={sortKey === 'name'} dir={sortDir} onClick={() => toggleSort('name')} className="px-3 py-2 font-medium" />
                  <SortableTh label={tA('admin.api.col_count')} active={sortKey === 'count'} dir={sortDir} onClick={() => toggleSort('count')} className="px-3 py-2 text-right font-medium" />
                  <SortableTh label={tA('admin.api.col_avg')} active={sortKey === 'avg_ms'} dir={sortDir} onClick={() => toggleSort('avg_ms')} className="px-3 py-2 text-right font-medium" />
                  <SortableTh label={tA('admin.api.col_max')} active={sortKey === 'max_ms'} dir={sortDir} onClick={() => toggleSort('max_ms')} className="px-3 py-2 text-right font-medium" />
                  <SortableTh label={tA('admin.api.col_total')} active={sortKey === 'sum_ms'} dir={sortDir} onClick={() => toggleSort('sum_ms')} className="px-3 py-2 text-right font-medium" />
                  <SortableTh label={tA('admin.api.col_errors')} active={sortKey === 'errors'} dir={sortDir} onClick={() => toggleSort('errors')} className="px-3 py-2 text-right font-medium" />
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

function playerCallRawValue(s: PerfPlayerCallStats, key: PlayerCallSortKey): string | number {
  return s[key]
}

function comparePlayerCallStats(a: PerfPlayerCallStats, b: PerfPlayerCallStats, key: PlayerCallSortKey, dir: 'asc' | 'desc'): number {
  const va = playerCallRawValue(a, key)
  const vb = playerCallRawValue(b, key)
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

/** Sous-tableau des appels API attribuables par joueur (erreurs desc). */
function ApiByPlayerTable({ perf }: { perf: AdminPerfStats }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  const [sortKey, setSortKey] = useState<PlayerCallSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: PlayerCallSortKey) {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key === 'player' || key === 'call' ? 'asc' : 'desc')
    }
  }
  const sortedStats = useMemo(() => {
    const stats = perf.api_by_player ?? []
    if (!sortKey) return stats
    return [...stats].sort((a, b) => comparePlayerCallStats(a, b, sortKey, sortDir))
  }, [perf.api_by_player, sortKey, sortDir])

  return (
    <div className="space-y-1.5">
      <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.api.by_player_section')}
      </h4>
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <SortableTh label={tA('admin.api.col_player')} active={sortKey === 'player'} dir={sortDir} onClick={() => toggleSort('player')} className="px-3 py-2 font-medium" />
              <SortableTh label={tA('admin.api.col_call')} active={sortKey === 'call'} dir={sortDir} onClick={() => toggleSort('call')} className="px-3 py-2 font-medium" />
              <SortableTh label={tA('admin.api.col_count')} active={sortKey === 'count'} dir={sortDir} onClick={() => toggleSort('count')} className="px-3 py-2 text-right font-medium" />
              <SortableTh label={tA('admin.api.col_avg')} active={sortKey === 'avg_ms'} dir={sortDir} onClick={() => toggleSort('avg_ms')} className="px-3 py-2 text-right font-medium" />
              <SortableTh label={tA('admin.api.col_max')} active={sortKey === 'max_ms'} dir={sortDir} onClick={() => toggleSort('max_ms')} className="px-3 py-2 text-right font-medium" />
              <SortableTh label={tA('admin.api.col_errors')} active={sortKey === 'errors'} dir={sortDir} onClick={() => toggleSort('errors')} className="px-3 py-2 text-right font-medium" />
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
