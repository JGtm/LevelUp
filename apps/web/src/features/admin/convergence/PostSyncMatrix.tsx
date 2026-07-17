/**
 * PostSyncMatrix — compteurs du pipeline post-sync du dernier RunDelta par
 * joueur (capturés par le scheduler — paths watcher/HTTP non couverts, cf.
 * limite D2 du plan). Table dense, cellules à 0 en muted, erreurs fatales en
 * destructive.
 */
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { SchedulerPlayerOutcome } from '@/lib/api/types'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import { Sparkline } from '@/components/charts/Sparkline'
import { useAdminT } from '../useAdminText'

interface PostSyncColumn {
  labelKey: AdminManifestKey
  value: (p: NonNullable<SchedulerPlayerOutcome['post_sync']>) => number | string
  /** Colonne booléenne : true → libellé i18n admin.status.ok (A8.7), false → '—'. */
  boolean?: boolean
}

const COLUMNS: PostSyncColumn[] = [
  { labelKey: 'admin.convergence.ps_perf', value: (ps) => ps.perf_scores_computed },
  { labelKey: 'admin.convergence.ps_lusr', value: (ps) => ps.lusr_updated },
  { labelKey: 'admin.convergence.ps_sessions', value: (ps) => ps.sessions_assigned },
  { labelKey: 'admin.convergence.ps_weapons', value: (ps) => ps.weapon_kills_processed },
  { labelKey: 'admin.convergence.ps_no_film', value: (ps) => ps.weapon_kills_no_film },
  { labelKey: 'admin.convergence.ps_citations', value: (ps) => ps.citations_computed },
  { labelKey: 'admin.convergence.ps_dominance', value: (ps) => ps.dominance_flags_computed },
  { labelKey: 'admin.convergence.ps_engagement', value: (ps) => ps.engagement_scores_computed },
  { labelKey: 'admin.convergence.ps_friends', value: (ps) => Number(ps.matches_promoted_friends) },
  { labelKey: 'admin.convergence.ps_views', value: (ps) => ps.views_refreshed },
  { labelKey: 'admin.convergence.ps_career', value: (ps) => Number(ps.career_synced), boolean: true },
  { labelKey: 'admin.convergence.ps_achievements', value: (ps) => Number(ps.achievements_synced), boolean: true },
]

export function PostSyncMatrix({ players }: { players: SchedulerPlayerOutcome[] }) {
  const tA = useAdminT()
  const withPostSync = players.filter((p) => p.post_sync)

  if (!withPostSync.length) {
    return <EmptyStateNotice title={tA('admin.convergence.postsync_empty')} description="" />
  }

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.convergence.col_player')}</th>
            <th className="px-2 py-2 font-medium">{tA('admin.convergence.ps_trend')}</th>
            {COLUMNS.map((c) => (
              <th key={c.labelKey} className="px-2 py-2 text-right font-medium">
                {tA(c.labelKey)}
              </th>
            ))}
            <th className="px-2 py-2 text-right font-medium">{tA('admin.convergence.ps_fatal')}</th>
          </tr>
        </thead>
        <tbody>
          {withPostSync.map((p) => {
            const ps = p.post_sync
            if (!ps) return null
            const fatal = ps.fatal_errors?.length ?? 0
            return (
              <tr key={p.xuid || p.gamertag} className="border-b last:border-b-0 hover:bg-muted/30">
                <td className="px-3 py-2 font-medium text-foreground">{p.gamertag}</td>
                <td className="px-2 py-2">
                  {(p.post_sync_history_ms?.length ?? 0) >= 2 ? (
                    <Sparkline values={p.post_sync_history_ms ?? []} token="info" width={64} height={18} ariaLabel={p.gamertag} />
                  ) : (
                    <span className="text-muted-foreground/60">—</span>
                  )}
                </td>
                {COLUMNS.map((c) => {
                  const raw = c.value(ps)
                  const v = c.boolean ? (raw ? tA('admin.status.ok') : '—') : raw
                  const zero = raw === 0 || v === '—'
                  return (
                    <td
                      key={c.labelKey}
                      className={`px-2 py-2 text-right font-mono text-xs tabular-nums ${zero ? 'text-muted-foreground/60' : 'text-foreground'}`}
                    >
                      {v}
                    </td>
                  )
                })}
                <td
                  className="px-2 py-2 text-right font-mono text-xs tabular-nums"
                  style={fatal > 0 ? { color: tokenCssVar('destructive') } : undefined}
                  title={fatal > 0 ? ps.fatal_errors?.join(' | ') : undefined}
                >
                  {fatal > 0 ? fatal : <span className="text-muted-foreground/60">0</span>}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
