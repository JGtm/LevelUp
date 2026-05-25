/**
 * SquadVsSoloCard — Phase 1 : comparaison perfs solo vs squad.
 *
 * Affiché uniquement si le PatternReport contient des patterns bySquad.
 */
import type { ContextualPattern } from './types'
import type { AscensionText } from './i18n'

interface SquadVsSoloCardProps {
  patterns: ContextualPattern[]
  t: AscensionText
}

export function SquadVsSoloCard({ patterns, t }: SquadVsSoloCardProps) {
  const squadPatterns = patterns.filter((p) => p.type === 'by_squad')
  if (squadPatterns.length === 0) return null

  const solo = squadPatterns.find((p) => p.key === 'solo')
  const squad = squadPatterns.find((p) => p.key === 'with_friends')
  if (!solo || !squad) return null

  const better = squad.win_rate >= solo.win_rate ? 'squad' : 'solo'

  return (
    <div className="rounded-md border border-border bg-card p-4">
      <p className="mb-3 text-sm font-semibold">{t.squadVsSoloTitle}</p>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground">{t.squadVsSoloSolo}</p>
          <StatRow label={t.patternWinRate} value={pct(solo.win_rate)} />
          <StatRow label={t.metric?.kda ?? 'KDA'} value={solo.avg_kda.toFixed(2)} />
          <StatRow label="OC" value={solo.avg_oc.toFixed(2)} />
          <StatRow label="DR" value={solo.avg_dr.toFixed(2)} />
          <p className="text-[10px] text-muted-foreground">{solo.match_count} {t.patternMatches}</p>
        </div>
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground">{t.squadVsSoloSquad}</p>
          <StatRow
            label={t.patternWinRate}
            value={pct(squad.win_rate)}
            highlight={better === 'squad'}
          />
          <StatRow label={t.metric?.kda ?? 'KDA'} value={squad.avg_kda.toFixed(2)} highlight={squad.avg_kda > solo.avg_kda} />
          <StatRow label="OC" value={squad.avg_oc.toFixed(2)} />
          <StatRow label="DR" value={squad.avg_dr.toFixed(2)} />
          <p className="text-[10px] text-muted-foreground">{squad.match_count} {t.patternMatches}</p>
        </div>
      </div>
    </div>
  )
}

function StatRow({ label, value, highlight = false }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="flex items-center justify-between text-xs">
      <span className="text-muted-foreground">{label}</span>
      <span className={highlight ? 'font-semibold text-green-600 dark:text-green-400' /* color-allow: highlight stat — CLAUDE.md §20 */ : ''}>{value}</span>
    </div>
  )
}

function pct(v: number): string {
  return `${Math.round(v * 100)}%`
}
