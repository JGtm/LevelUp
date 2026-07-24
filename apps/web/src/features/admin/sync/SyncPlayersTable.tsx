/**
 * SyncPlayersTable — détail par joueur du dernier cycle auto-sync : outcome,
 * raison, durée, inserts, compteur zero-insert (alerte au seuil backend).
 * Tri : échecs d'abord, puis OK, puis ignorés (à outcome égal : gamertag).
 *
 * EXCEPTION tri client par en-têtes (I16) : volontairement NON triable.
 * Le tri fixe failed→ok→skipped est INTENTIONNEL (les problèmes remontent en
 * premier, c'est la valeur diagnostique de cette table) — un tri par en-tête
 * le remplacerait par un ordre arbitraire et masquerait les échecs.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { SchedulerPlayerOutcome } from '@/lib/api/types'
import { schedulerOutcomeToAdminStatus } from '../statusDisplay'
import { formatDurationMs, adminRelativeTime, adminAbsoluteTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { StatusBadge } from '../components/StatusBadge'

const OUTCOME_ORDER: Record<string, number> = { failed: 0, ok: 1, skipped: 2 }

/** Tri : failed → ok → skipped, puis gamertag (les problèmes d'abord). */
function sortPlayersForDisplay(players: SchedulerPlayerOutcome[]): SchedulerPlayerOutcome[] {
  return [...players].sort((a, b) => {
    const oa = OUTCOME_ORDER[a.outcome] ?? 3
    const ob = OUTCOME_ORDER[b.outcome] ?? 3
    if (oa !== ob) return oa - ob
    return a.gamertag.localeCompare(b.gamertag)
  })
}

export function SyncPlayersTable({
  players,
  zeroInsertThreshold,
}: {
  players: SchedulerPlayerOutcome[]
  zeroInsertThreshold: number
}) {
  const tA = useAdminT()
  const locale = useAdminLocale()

  if (!players.length) {
    return <p className="text-sm text-muted-foreground">{tA('admin.sync.no_players')}</p>
  }

  const sorted = sortPlayersForDisplay(players)
  const outcomeLabel = (outcome: string) =>
    outcome === 'ok'
      ? tA('admin.sync.outcome_ok')
      : outcome === 'skipped'
        ? tA('admin.sync.outcome_skipped')
        : outcome === 'failed'
          ? tA('admin.sync.outcome_failed')
          : outcome

  return (
    <div className="overflow-x-auto rounded-md border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
            <th className="px-3 py-2 font-medium">{tA('admin.sync.col_player')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.sync.col_outcome')}</th>
            <th className="px-3 py-2 font-medium">{tA('admin.sync.col_reason')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.sync.col_duration')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.sync.col_inserted')}</th>
            <th className="px-3 py-2 font-medium text-right">{tA('admin.sync.col_zero_streak')}</th>
          </tr>
        </thead>
        <tbody>
          {sorted.map((p) => {
            const zeroStreak = p.consecutive_zero_inserts ?? 0
            const zeroAlert = zeroStreak >= zeroInsertThreshold
            return (
              <tr key={p.xuid || p.gamertag} className="border-b last:border-b-0 hover:bg-muted/30">
                <td className="px-3 py-2 font-medium text-foreground">
                  {p.gamertag}
                  <div
                    className="text-xs font-normal text-muted-foreground"
                    title={adminAbsoluteTime(p.attempted_at, locale)}
                  >
                    {adminRelativeTime(p.attempted_at, locale)}
                  </div>
                </td>
                <td className="px-3 py-2">
                  <StatusBadge
                    status={schedulerOutcomeToAdminStatus(p.outcome)}
                    label={outcomeLabel(p.outcome)}
                  />
                </td>
                <td className="max-w-[28rem] px-3 py-2 text-xs text-muted-foreground">
                  <span className="line-clamp-2" title={p.first_error || p.reason}>
                    {p.reason}
                  </span>
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-muted-foreground">
                  {formatDurationMs(p.duration_ms, locale)}
                </td>
                <td className="px-3 py-2 text-right font-mono text-xs tabular-nums text-foreground">
                  {p.matches_inserted ?? 0}
                </td>
                <td className="px-3 py-2 text-right">
                  <span
                    className="font-mono text-xs tabular-nums"
                    style={zeroAlert ? { color: tokenCssVar('warning') } : undefined}
                    title={zeroAlert ? `>= ${zeroInsertThreshold}` : undefined}
                  >
                    {zeroStreak}
                  </span>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
