/**
 * PlayerScoreCard — carte compacte d'un joueur dans le bandeau Squad header.
 *
 * Conformément au PLAN_SQUAD_GO_PORTAGE § 1.1 : score 0..100 + label qualitatif
 * + badge ▲▼= comparant au score moyen squad.
 *
 * i18n-naïf : tous les labels sont déjà localisés en amont.
 */
import type { PlayerScoreComparison } from '@/features/squad/types'

export interface PlayerScoreCardProps {
  /** Gamertag affichable (déjà résolu côté backend). */
  gamertag: string
  /** Score 0..100. */
  score: number
  /** Label qualitatif déjà localisé (ex. "Excellent", "Bon", "Moyen"). */
  label: string
  /** Comparaison vs moyenne squad. */
  comparison: PlayerScoreComparison
  /** True si c'est la carte du joueur principal (highlight visuel). */
  isMainPlayer?: boolean
  className?: string
}

const COMPARISON_GLYPH: Record<PlayerScoreComparison, string> = {
  above: '▲',
  below: '▼',
  near: '=',
}

const COMPARISON_VAR: Record<PlayerScoreComparison, string> = {
  above: '--narrative-trend-positive',
  below: '--narrative-trend-negative',
  near: '--narrative-trend-neutral',
}

const SCORE_LABEL_VAR: Record<string, string> = {
  excellent: '--score-excellent',
  good: '--score-good',
  average: '--score-average',
  poor: '--score-poor',
  bad: '--score-bad',
}

/**
 * PlayerScoreCard rend une carte compacte avec le score + label + comparaison.
 * Les couleurs sont appliquées via CSS variables (CLAUDE.md §20).
 */
export function PlayerScoreCard({
  gamertag,
  score,
  label,
  comparison,
  isMainPlayer = false,
  className = '',
}: PlayerScoreCardProps) {
  const scoreColorVar = SCORE_LABEL_VAR[label.toLowerCase()] ?? '--score-average'
  return (
    <div
      className={`flex flex-col gap-1 rounded-lg border bg-card px-3 py-2 ${
        isMainPlayer ? 'border-primary/40' : 'border-border'
      } ${className}`}
      data-testid="player-score-card"
      data-gamertag={gamertag}
      data-main={isMainPlayer ? 'true' : 'false'}
    >
      <div className="flex items-center justify-between text-xs font-medium text-muted-foreground">
        <span data-testid="player-score-card-gamertag">{gamertag}</span>
        <span
          className="text-sm font-bold leading-none"
          style={{ color: `var(${COMPARISON_VAR[comparison]})` }}
          data-testid="player-score-card-comparison"
          data-comparison={comparison}
          aria-hidden="true"
        >
          {COMPARISON_GLYPH[comparison]}
        </span>
      </div>
      <div className="flex items-baseline gap-2">
        <span
          className="text-2xl font-semibold leading-none"
          style={{ color: `var(${scoreColorVar})` }}
          data-testid="player-score-card-score"
        >
          {Math.round(score)}
        </span>
        <span
          className="text-xs uppercase tracking-wide"
          style={{ color: `var(${scoreColorVar})` }}
          data-testid="player-score-card-label"
        >
          {label}
        </span>
      </div>
    </div>
  )
}
