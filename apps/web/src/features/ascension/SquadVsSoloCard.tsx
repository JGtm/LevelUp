/**
 * SquadVsSoloCard — Phase 1 : comparaison perfs solo vs squad.
 *
 * Affiché uniquement si le PatternReport contient des patterns bySquad.
 */
import type { ReactNode } from 'react'
import { metricLabel } from '@/lib/i18n/metricLabel'
import type { ContextualPattern } from './types'
import type { AscensionText, AscensionLocale } from './i18n'
import { CombatAbbr } from './CombatAbbr'

interface SquadVsSoloCardProps {
  patterns: ContextualPattern[]
  t: AscensionText
  locale: AscensionLocale
}

export function SquadVsSoloCard({ patterns, t, locale }: SquadVsSoloCardProps) {
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
          <StatRow label={metricLabel('kda', locale)} value={solo.avg_kda.toFixed(2)} />
          <StatRow label={<CombatAbbr metric="oc" t={t} />} value={`${Math.round(solo.avg_oc * 100)}%`} />
          <StatRow label={<CombatAbbr metric="dr" t={t} />} value={`${Math.round((solo.avg_dr - 1) * 100)}%`} />
          <p className="text-[10px] text-muted-foreground">{solo.match_count} {t.patternMatches}</p>
        </div>
        <div className="space-y-1">
          <p className="text-xs font-medium text-muted-foreground">{t.squadVsSoloSquad}</p>
          <StatRow
            label={t.patternWinRate}
            value={pct(squad.win_rate)}
            highlight={better === 'squad'}
          />
          <StatRow label={metricLabel('kda', locale)} value={squad.avg_kda.toFixed(2)} highlight={squad.avg_kda > solo.avg_kda} />
          <StatRow label={<CombatAbbr metric="oc" t={t} />} value={`${Math.round(squad.avg_oc * 100)}%`} />
          <StatRow label={<CombatAbbr metric="dr" t={t} />} value={`${Math.round((squad.avg_dr - 1) * 100)}%`} />
          <p className="text-[10px] text-muted-foreground">{squad.match_count} {t.patternMatches}</p>
        </div>
      </div>
    </div>
  )
}

function StatRow({ label, value, highlight = false }: { label: ReactNode; value: string; highlight?: boolean }) {
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
