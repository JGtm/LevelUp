/**
 * ApiHaloSection — latences API Halo par type d'appel depuis le boot
 * (films, stats, skill…) + buckets d'erreurs (429/auth/5xx/réseau). Identifie
 * si le goulot de sync est côté API (343) ou côté app.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { AdminPerfStats } from '@/lib/api/types'
import { formatDurationMs } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

export function ApiHaloSection({ perf }: { perf: AdminPerfStats | undefined }) {
  const tA = useAdminT()
  const locale = useAdminLocale()

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
                  <th className="px-3 py-2 font-medium">{tA('admin.api.col_call')}</th>
                  <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_count')}</th>
                  <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_avg')}</th>
                  <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_max')}</th>
                  <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_total')}</th>
                  <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_errors')}</th>
                </tr>
              </thead>
              <tbody>
                {(perf.api_calls ?? []).map((c) => (
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

/** Sous-tableau des appels API attribuables par joueur (erreurs desc). */
function ApiByPlayerTable({ perf }: { perf: AdminPerfStats }) {
  const tA = useAdminT()
  const locale = useAdminLocale()
  return (
    <div className="space-y-1.5">
      <h4 className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {tA('admin.api.by_player_section')}
      </h4>
      <div className="overflow-x-auto rounded-md border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
              <th className="px-3 py-2 font-medium">{tA('admin.api.col_player')}</th>
              <th className="px-3 py-2 font-medium">{tA('admin.api.col_call')}</th>
              <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_count')}</th>
              <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_avg')}</th>
              <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_max')}</th>
              <th className="px-3 py-2 text-right font-medium">{tA('admin.api.col_errors')}</th>
            </tr>
          </thead>
          <tbody>
            {(perf.api_by_player ?? []).map((s, i) => (
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
