/**
 * PostSyncMatrix — compteurs du pipeline post-sync du dernier RunDelta par
 * joueur (capturés par le scheduler — paths watcher/HTTP non couverts, cf.
 * limite D2 du plan). Table dense, cellules à 0 en muted, erreurs fatales en
 * destructive.
 */
import { useMemo, useState } from 'react'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { SortableTh } from '@/components/ui/sortable-th'
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
  { labelKey: 'admin.convergence.ps_citations', value: (ps) => ps.citations_computed },
  { labelKey: 'admin.convergence.ps_dominance', value: (ps) => ps.dominance_flags_computed },
  { labelKey: 'admin.convergence.ps_engagement', value: (ps) => ps.engagement_scores_computed },
  { labelKey: 'admin.convergence.ps_friends', value: (ps) => Number(ps.matches_promoted_friends) },
  { labelKey: 'admin.convergence.ps_views', value: (ps) => ps.views_refreshed },
  { labelKey: 'admin.convergence.ps_career', value: (ps) => Number(ps.career_synced), boolean: true },
  { labelKey: 'admin.convergence.ps_achievements', value: (ps) => Number(ps.achievements_synced), boolean: true },
]

// I16 : clé de tri — colonne joueur (texte), une des COLUMNS dynamiques
// (index — value() est toujours numérique, y compris les colonnes booléennes
// qui renvoient Number(bool)), ou la colonne fatale (fixe).
type PostSyncSortKey = { kind: 'gamertag' } | { kind: 'col'; index: number } | { kind: 'fatal' }

function postSyncSortKeyEq(a: PostSyncSortKey | null, b: PostSyncSortKey): boolean {
  if (!a) return false
  if (a.kind !== b.kind) return false
  return a.kind === 'col' && b.kind === 'col' ? a.index === b.index : true
}

function postSyncRawValue(p: SchedulerPlayerOutcome, key: PostSyncSortKey): string | number {
  if (key.kind === 'gamertag') return p.gamertag
  if (key.kind === 'fatal') return p.post_sync?.fatal_errors?.length ?? 0
  const ps = p.post_sync
  return ps ? COLUMNS[key.index].value(ps) : 0
}

function comparePostSync(a: SchedulerPlayerOutcome, b: SchedulerPlayerOutcome, key: PostSyncSortKey, dir: 'asc' | 'desc'): number {
  const va = postSyncRawValue(a, key)
  const vb = postSyncRawValue(b, key)
  const cmp = typeof va === 'string' && typeof vb === 'string' ? va.localeCompare(vb, undefined, { numeric: true, sensitivity: 'base' }) : (va as number) - (vb as number)
  return dir === 'asc' ? cmp : -cmp
}

export function PostSyncMatrix({ players }: { players: SchedulerPlayerOutcome[] }) {
  const tA = useAdminT()
  const withPostSync = players.filter((p) => p.post_sync)
  // Aucun tri actif par défaut → l'ordre serveur (dernier RunDelta) reste
  // affiché tant qu'aucun en-tête n'a été cliqué.
  const [sortKey, setSortKey] = useState<PostSyncSortKey | null>(null)
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('desc')
  function toggleSort(key: PostSyncSortKey) {
    if (postSyncSortKeyEq(sortKey, key)) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir(key.kind === 'gamertag' ? 'asc' : 'desc')
    }
  }
  const sortedPlayers = useMemo(() => {
    if (!sortKey) return withPostSync
    return [...withPostSync].sort((a, b) => comparePostSync(a, b, sortKey, sortDir))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [withPostSync, sortKey, sortDir])

  if (!withPostSync.length) {
    return <EmptyStateNotice title={tA('admin.convergence.postsync_empty')} description="" />
  }

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <SortableTh label={tA('admin.convergence.col_player')} active={postSyncSortKeyEq(sortKey, { kind: 'gamertag' })} dir={sortDir} onClick={() => toggleSort({ kind: 'gamertag' })} className="px-3 py-2 font-medium" />
            {/* Sparkline : pas de valeur scalaire triable — jamais triable. */}
            <th className="px-2 py-2 font-medium">{tA('admin.convergence.ps_trend')}</th>
            {COLUMNS.map((c, index) => {
              const key: PostSyncSortKey = { kind: 'col', index }
              return (
                <SortableTh
                  key={c.labelKey}
                  label={tA(c.labelKey)}
                  active={postSyncSortKeyEq(sortKey, key)}
                  dir={sortDir}
                  onClick={() => toggleSort(key)}
                  className="px-2 py-2 text-right font-medium"
                />
              )
            })}
            <SortableTh label={tA('admin.convergence.ps_fatal')} active={postSyncSortKeyEq(sortKey, { kind: 'fatal' })} dir={sortDir} onClick={() => toggleSort({ kind: 'fatal' })} className="px-2 py-2 text-right font-medium" />
          </tr>
        </thead>
        <tbody>
          {sortedPlayers.map((p) => {
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
