/**
 * LUSRComponentsGrid — Section B : 8 composantes LUSR avec barres de progression.
 *
 * Pour chaque composante : current_avg (barre pleine), target_for_tier (repère),
 * personal_top_20 (borne haute), tendance (flèche).
 */
import type { LUSRComponentBreakdown, SkillRatingSnapshot } from './types'
import type { AscensionText } from './i18n'

interface LUSRComponentsGridProps {
  components: LUSRComponentBreakdown[]
  skillRating: SkillRatingSnapshot
  t: AscensionText
}

export function LUSRComponentsGrid({ components, skillRating, t }: LUSRComponentsGridProps) {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm font-semibold text-muted-foreground">
          {t.lusrTierLabel} : <span className="text-foreground">{skillRating.label}</span>
        </p>
        {skillRating.gap_to_next != null && skillRating.gap_to_next > 0 && (
          <p className="text-xs text-muted-foreground">
            {t.lusrGapToNext?.replace('{n}', skillRating.gap_to_next.toFixed(1)) ?? `+${skillRating.gap_to_next.toFixed(1)} μ`}
            {' → '}{skillRating.next_tier_label}
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        {components.map((c) => (
          <ComponentBar key={c.name} component={c} t={t} />
        ))}
      </div>
    </div>
  )
}

function ComponentBar({ component: c, t }: { component: LUSRComponentBreakdown; t: AscensionText }) {
  const top = Math.max(c.personal_top_20, c.target_for_tier, c.current_avg, 0.01)
  const pctCurrent = Math.min((c.current_avg / top) * 100, 100)
  const pctTarget = Math.min((c.target_for_tier / top) * 100, 100)

  const trendIcon = c.trend > 0.02 ? '↑' : c.trend < -0.02 ? '↓' : '→'
  const trendClass =
    c.trend > 0.02
      ? 'text-green-600 dark:text-green-400' // color-allow: trend indicator — CLAUDE.md §20
      : c.trend < -0.02
        ? 'text-red-500 dark:text-red-400' // color-allow: trend indicator — CLAUDE.md §20
        : 'text-muted-foreground'

  const label = t.lusrComponent?.[c.name] ?? c.name

  return (
    <div className="rounded border border-border bg-card p-2">
      <div className="mb-1 flex items-center justify-between text-xs">
        <span className="font-medium">{label}</span>
        <span className={trendClass} aria-label={`trend ${trendIcon}`}>{trendIcon}</span>
      </div>
      <div className="relative h-2 overflow-hidden rounded-full bg-muted">
        <div
          className="absolute inset-y-0 left-0 rounded-full bg-primary/70"
          style={{ width: `${pctCurrent}%` }}
        />
        {pctTarget > 0 && (
          <div
            className={'absolute inset-y-0 w-px bg-amber-500' /* color-allow: tier target marker line — CLAUDE.md §20 */}
            style={{ left: `${pctTarget}%` }}
            title={t.lusrTargetForTier}
          />
        )}
      </div>
      <div className="mt-1 flex justify-between text-[10px] text-muted-foreground">
        <span>{c.current_avg.toFixed(2)}</span>
        <span>{t.lusrTop20} {c.personal_top_20.toFixed(2)}</span>
      </div>
    </div>
  )
}
