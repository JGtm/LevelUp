/**
 * MatchHeader — PerfRankRow + barre de progression composite dans le tier.
 *
 * Découpé depuis MatchHeader.card.tsx (audit #6 god-file split).
 * tierSize = 50 pts (même constante que le backend buildRankBlock).
 */
import { tokenCssVar } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'
import { MATCH_VIEW_TEXT, type MatchViewLocale } from './i18n'
import type { MatchViewHeader as MatchViewHeaderData, MatchViewRank } from '@/lib/api/types'
import { nextTierLabel } from './MatchHeader.utils'

interface PerfRankRowProps {
  header: MatchViewHeaderData
  rank: MatchViewRank
  perfColor: string
  locale: MatchViewLocale
}

export function PerfRankRow({ header, rank, perfColor, locale }: PerfRankRowProps) {
  const t = MATCH_VIEW_TEXT[locale]

  const deltaColor =
    rank.delta_value != null
      ? tokenCssVar(skillDeltaScale(rank.delta_value))
      : 'inherit'

  // Barre composite de progression dans le tier.
  // tierSize = 50 pts (même constante que le backend buildRankBlock).
  const TIER_SIZE = 50
  const rankDeltaPct = rank.delta_value != null ? rank.delta_value / TIER_SIZE : 0
  const rankCurrentFill = rank.progress_pct ?? null
  // Position avant ce match (clampée à [0, 1] si le delta a changé de tier)
  const rankBeforeFill =
    rankCurrentFill != null
      ? Math.max(0, Math.min(1, rankCurrentFill - rankDeltaPct))
      : null
  // Portion stable = la plus petite des deux positions
  const rankBaseFill = rankCurrentFill != null && rankBeforeFill != null
    ? Math.min(rankCurrentFill, rankBeforeFill)
    : null
  // Segment delta : début et largeur en %
  const rankDeltaStart =
    rankCurrentFill != null && rankBeforeFill != null
      ? Math.min(rankCurrentFill, rankBeforeFill) * 100
      : 0
  const rankDeltaWidth =
    rankCurrentFill != null && rankBeforeFill != null
      ? Math.abs(rankCurrentFill - rankBeforeFill) * 100
      : 0
  const rankDeltaColor =
    rankDeltaPct > 0
      ? tokenCssVar('divergent-pos')
      : rankDeltaPct < 0
        ? tokenCssVar('divergent-neg')
        : tokenCssVar('divergent-neutral')

  return (
    <div className="mt-auto flex flex-wrap items-start gap-y-3 border-t pt-3">
      {header.performance_display && header.performance_display !== '-' && (
        <div className="flex flex-col items-center">
          <span className="text-2xs font-semibold uppercase tracking-wider text-muted-foreground">
            {t.performance}
          </span>
          <span
            className="text-4xl font-black leading-none tabular-nums"
            style={{ color: perfColor }}
          >
            {header.performance_display}
          </span>
        </div>
      )}

      {header.performance_display && header.performance_display !== '-' && rank.rating_type !== 'none' && (
        <div className="mx-6 w-px self-stretch bg-border" />
      )}

      {rank.rating_type !== 'none' && (
        <div className="flex flex-1 items-center gap-3 min-w-0">
          {rank.icon_url && (
            <img
              src={rank.icon_url}
              alt={rank.tier_label ?? rank.rating_type}
              className="h-[44px] w-[44px] shrink-0 object-contain"
              loading="lazy"
            />
          )}
          <div className="flex flex-col gap-0.5 shrink-0">
            <span className="text-2xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t.rank}
            </span>
            {rank.tier_label && (
              <span className="text-base font-bold text-foreground leading-none">
                {rank.tier_label}
              </span>
            )}
            <div className="flex items-center gap-2 text-xs">
              {rank.numeric_value != null && (
                <span className="tabular-nums text-muted-foreground">
                  {rank.rating_type} {rank.numeric_value.toFixed(0)}
                </span>
              )}
              {rank.delta_value != null && (
                <span
                  className="font-bold tabular-nums"
                  style={{ color: deltaColor }}
                >
                  {rank.delta_value >= 0
                    ? `▲ +${rank.delta_value.toFixed(0)}`
                    : `▼ ${rank.delta_value.toFixed(0)}`}
                </span>
              )}
            </div>
          </div>

          {rankCurrentFill != null && rankBaseFill != null && (
            // Conteneur relatif — les labels sont en absolute top-full pour
            // ne pas gonfler la hauteur et décentrer la barre.
            <div className="relative flex flex-1 items-center min-w-[80px]">
              <div className="relative h-2 w-full overflow-hidden rounded-sm bg-muted">
                {/* Portion stable (position avant le match) */}
                <div
                  className="absolute inset-y-0 left-0"
                  style={{
                    width: `${(rankBaseFill * 100).toFixed(1)}%`,
                    backgroundColor: tokenCssVar('divergent-neutral'),
                  }}
                />
                {/* Segment delta (gain ou perte ce match) */}
                {rankDeltaWidth > 0.1 && (
                  <div
                    className="absolute inset-y-0"
                    style={{
                      left: `${rankDeltaStart.toFixed(1)}%`,
                      width: `${rankDeltaWidth.toFixed(1)}%`,
                      backgroundColor: rankDeltaColor,
                    }}
                  />
                )}
              </div>
              <div className="absolute inset-x-0 top-full mt-1 flex justify-between text-2xs text-muted-foreground tabular-nums">
                <span>{rank.tier_label ?? ''}</span>
                <span>{nextTierLabel(rank.tier_label)}</span>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
