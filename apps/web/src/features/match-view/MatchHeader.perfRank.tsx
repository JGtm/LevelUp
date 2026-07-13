/**
 * MatchHeader — PerfRankRow + barre de progression composite dans le tier.
 *
 * Découpé depuis MatchHeader.card.tsx (audit #6 god-file split).
 * tierSize = 50 pts (même constante que le backend buildRankBlock).
 */
import { tokenCssVar } from '@/lib/accessibility'
import { skillDeltaScale } from '@/lib/accessibility/scales'
import { gridForRatingTypes, subTierPosition } from '@/lib/skillTiers'
import { formatRankDelta } from '@/lib/formatters'
import { MATCH_VIEW_TEXT, type MatchViewLocale } from './i18n'
import type { MatchViewHeader as MatchViewHeaderData, MatchViewRank } from '@/lib/api/types'
import { nextTierLabel, displayTierLabel } from './MatchHeader.utils'

const clamp01 = (x: number) => Math.max(0, Math.min(1, x))

interface PerfRankRowProps {
  header: MatchViewHeaderData
  rank: MatchViewRank
  perfColor: string
  locale: MatchViewLocale
}

export function PerfRankRow({ header, rank, perfColor, locale }: PerfRankRowProps) {
  const t = MATCH_VIEW_TEXT[locale]
  // « Placement » (sentinelle back) → libellé localisé « En placement », aligné
  // sur les cards d'accueil.
  const tierLabel = displayTierLabel(rank.tier_label, t.rankPlacement)

  const deltaColor =
    rank.delta_value != null
      ? tokenCssVar(skillDeltaScale(rank.delta_value))
      : 'inherit'

  // Barre composite de progression DANS le sous-palier courant.
  //
  // Calculée depuis `numeric_value` + la grille réelle du type de rating (CSR =
  // sous-paliers de 50 pts ; LUSR = échelle legacy 1000-2000 avec sous-paliers
  // de largeur variable 33/67/100 selon le tier). L'ancienne reconstruction
  // `progress_pct - delta/50` supposait à tort 50 pts pour TOUS les ratings :
  // sur LUSR la base "avant match" tombait sous 0 et la barre passait entièrement
  // en vert même sans changement de sous-palier (cf. thought_log 2026-06-09).
  const grid = gridForRatingTypes([rank.rating_type])
  const currentPos =
    rank.numeric_value != null ? subTierPosition(grid, rank.numeric_value) : null

  // Position courante (après match) dans le sous-palier.
  const rankCurrentFill = currentPos ? currentPos.pct : null
  // Position avant ce match, ramenée DANS le sous-palier courant : si le match
  // a fait entrer dans un nouveau sous-palier, "avant" était sous la borne basse
  // → clampé à 0 (barre qui se remplit depuis le bas du nouveau palier).
  const rankBeforeFill =
    currentPos && rank.numeric_value != null
      ? clamp01(
          (rank.numeric_value - (rank.delta_value ?? 0) - currentPos.subTierMin) /
            currentPos.subTierWidth,
        )
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
    (rank.delta_value ?? 0) > 0
      ? tokenCssVar('divergent-pos')
      : (rank.delta_value ?? 0) < 0
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
              alt={tierLabel ?? rank.rating_type}
              className="h-[44px] w-[44px] shrink-0 object-contain"
              loading="lazy"
            />
          )}
          <div className="flex flex-col gap-0.5 shrink-0">
            <span className="text-2xs font-semibold uppercase tracking-wider text-muted-foreground">
              {t.rank}
            </span>
            {tierLabel && (
              <span className="text-base font-bold text-foreground leading-none">
                {tierLabel}
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
                  {rank.delta_value > 0 ? '▲ ' : rank.delta_value < 0 ? '▼ ' : ''}
                  {formatRankDelta(rank.delta_value, rank.rating_type)}
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
                  data-testid="rank-progress-base"
                  className="absolute inset-y-0 left-0"
                  style={{
                    width: `${(rankBaseFill * 100).toFixed(1)}%`,
                    backgroundColor: tokenCssVar('divergent-neutral'),
                  }}
                />
                {/* Segment delta (gain ou perte ce match) */}
                {rankDeltaWidth > 0.1 && (
                  <div
                    data-testid="rank-progress-delta"
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
                <span>{tierLabel ?? ''}</span>
                <span>{nextTierLabel(rank.tier_label)}</span>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
