/**
 * ResultsRail — bande descriptive du SessionBriefing.
 *
 * Affiche le scope de la session (Matchs / Durée / ⌀ par match) à gauche,
 * et la barre Résultats segmentée 4 couleurs avec libellés complets à droite.
 *
 * Libellés outcomes : tirés de outcomes.toml via useOutcomeLabel() (multi-titres).
 */
import { tokenCssVar } from '@/lib/accessibility'
import { useOutcomeLabel } from '@/lib/i18n/fieldMappings'

import type { KPIStats } from '@/lib/api/types'
import { formatDurationDhm, formatMmss } from './format'
import type { BriefingTexts } from './i18n'

interface ResultsRailProps {
  kpis: KPIStats
  texts: BriefingTexts
}

interface OutcomeSeg {
  count: number
  token: 'outcome-win' | 'outcome-loss' | 'outcome-draw' | 'outcome-dnf'
  outcomeKey: string
  fallbackLabel: string
}

export function ResultsRail({ kpis, texts }: ResultsRailProps) {
  const winLabel = useOutcomeLabel('win')
  const lossLabel = useOutcomeLabel('loss')
  const tieLabel = useOutcomeLabel('tie')
  const dnfLabel = useOutcomeLabel('dnf')

  const segs: OutcomeSeg[] = [
    { count: kpis.outcomes.wins, token: 'outcome-win', outcomeKey: 'win', fallbackLabel: winLabel },
    { count: kpis.outcomes.losses, token: 'outcome-loss', outcomeKey: 'loss', fallbackLabel: lossLabel },
    { count: kpis.outcomes.ties, token: 'outcome-draw', outcomeKey: 'tie', fallbackLabel: tieLabel },
    { count: kpis.outcomes.dnf, token: 'outcome-dnf', outcomeKey: 'dnf', fallbackLabel: dnfLabel },
  ]
  const total = segs.reduce((acc, s) => acc + s.count, 0)
  const hasResults = total > 0

  return (
    <div className="flex flex-wrap items-center justify-between gap-6 rounded border border-border bg-[#16191d] px-4 py-3">
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
          {texts.rail.sessionLabel}
        </span>
        <span className="text-sm">
          <strong>{kpis.matches_count}</strong> {texts.rail.matchesUnit}
        </span>
        <span className="text-xs text-muted-foreground">·</span>
        <span className="text-sm">
          {texts.rail.avgMatchPrefix}
          {formatMmss(kpis.avg_match_seconds)}
          {texts.rail.avgMatchSuffix}
        </span>
        <span className="text-xs text-muted-foreground">·</span>
        <span className="text-sm">{formatDurationDhm(kpis.total_play_seconds)}</span>
      </div>

      {hasResults && (
        <div className="flex min-w-[280px] flex-col gap-1.5">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {texts.rail.resultsLabel}
          </span>
          <div className="flex h-3.5 overflow-hidden rounded-sm">
            {segs.map((s) =>
              s.count > 0 ? (
                <div
                  key={s.outcomeKey}
                  style={{ flex: s.count, backgroundColor: tokenCssVar(s.token) }}
                  title={`${s.count} ${texts.pluralize(s.count, s.fallbackLabel)}`}
                />
              ) : null,
            )}
          </div>
          <div className="flex flex-wrap gap-3 text-[11px]">
            {segs.map((s) =>
              s.count > 0 ? (
                <span key={s.outcomeKey} className="inline-flex items-center gap-1.5">
                  <span
                    className="inline-block h-2 w-2 rounded-sm"
                    style={{ backgroundColor: tokenCssVar(s.token) }}
                  />
                  <strong>{s.count}</strong>{' '}
                  {texts.pluralize(s.count, s.fallbackLabel)}
                </span>
              ) : null,
            )}
          </div>
        </div>
      )}
    </div>
  )
}
