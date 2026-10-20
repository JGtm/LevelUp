/**
 * MetricWithTrend — primitive PARTAGÉE de flèche de tendance temporelle
 * (▲ up / ▼ down / = stable) colorée par token sémantique de narration.
 *
 * Foyer canonique (règle ≤2 copies) : réutilisé par LeaderboardBlock (delta
 * inter-saison des stats mondiales) et CareerRankingBlock (delta CSR de la
 * saison sélectionnée vs la précédente). NE PAS réinliner TREND_GLYPH /
 * TREND_VAR / MetricWithTrend ailleurs — garde-rail : metric-trend.guard.test.ts.
 *
 * Distinct du vocabulaire `above/below/near` (PlayerScoreCard) qui compare une valeur à une
 * RÉFÉRENCE (moyenne, all-time), pas à un instant antérieur : ne pas les fusionner sans mapper leur
 * sémantique.
 */
export type Trend = 'up' | 'down' | 'stable'

const TREND_GLYPH: Record<Trend, string> = { up: '▲', down: '▼', stable: '=' }
// Tokens sémantiques de tendance — jamais de hex direct.
const TREND_VAR: Record<Trend, string> = {
  up: '--narrative-trend-positive',
  down: '--narrative-trend-negative',
  stable: '--narrative-trend-neutral',
}

const isTrend = (v: unknown): v is Trend => v === 'up' || v === 'down' || v === 'stable'

/** Valeur d'une métrique suivie d'une flèche de tendance colorée (optionnelle). */
export function MetricWithTrend({
  text,
  trend,
  tooltip,
}: {
  text: string
  trend?: string | null
  tooltip?: string
}) {
  return (
    <span className="inline-flex items-baseline gap-1">
      <span>{text}</span>
      {isTrend(trend) && (
        <span
          className="text-[10px] font-bold leading-none"
          style={{ color: `var(${TREND_VAR[trend]})` }}
          title={tooltip}
          aria-label={tooltip}
        >
          {TREND_GLYPH[trend]}
        </span>
      )}
    </span>
  )
}
