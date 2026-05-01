/**
 * KpiGrid — bande KPI 8 cards du SessionBriefing.
 *
 * Affiche : Matchs joués, Durée totale, Durée moy/match, Frags/match, Morts/match,
 * Assists/match, Précision, Vie moyenne. Trends ▲/▼ (vs teamAvgKpis quand fourni
 * en mode squad) sur les 5 cards évaluatives (frags, morts, assists, précision,
 * vie). Morts = lower_is_better (▲ quand value < team_avg).
 *
 * Si teamAvgKpis est null (mode solo), trends affichés en 'none' (rien).
 */
import type { KPIStats } from '@/lib/api/types'

import { formatDurationDhm, formatMmss } from './format'
import type { BriefingTexts } from './i18n'
import { computeTrend, trendSymbol, type TrendState } from './trends'
import { tokenCssVar } from '@/lib/accessibility'

interface KpiGridProps {
  kpis: KPIStats
  /** Référence pour calcul trends ▲/▼. Null = mode solo, pas de trends. */
  teamAvgKpis: KPIStats | null
  texts: BriefingTexts
  /** Titre de la grille — gestionnaire externe (drilled vs self) */
  title: string
  /** Hint affiché à droite du titre — affiché seulement si teamAvgKpis présent */
  hint?: string
}

interface CellProps {
  label: string
  value: string
  sub?: string
  trend?: TrendState
}

function KpiCell({ label, value, sub, trend = 'none' }: CellProps) {
  const trendToken =
    trend === 'above'
      ? 'divergent-pos'
      : trend === 'below'
        ? 'divergent-neg'
        : trend === 'near'
          ? 'divergent-neutral'
          : null

  return (
    <div className="rounded border border-border bg-[#1d2328] px-3 py-2">
      <p className="text-[11px] uppercase tracking-wide text-muted-foreground">{label}</p>
      <div className="mt-0.5 flex items-baseline">
        <span className="text-lg font-bold">{value}</span>
        {trendToken && (
          <span
            className="ml-1 text-xs font-bold"
            style={{ color: tokenCssVar(trendToken) }}
          >
            {trendSymbol(trend)}
          </span>
        )}
      </div>
      {sub && <p className="mt-0.5 text-[10px] text-muted-foreground">{sub}</p>}
    </div>
  )
}

export function KpiGrid({ kpis, teamAvgKpis, texts, title, hint }: KpiGridProps) {
  const trendKills = teamAvgKpis
    ? computeTrend(kpis.kills_per_game, teamAvgKpis.kills_per_game)
    : 'none'
  const trendDeaths = teamAvgKpis
    ? computeTrend(kpis.deaths_per_game, teamAvgKpis.deaths_per_game, { lowerIsBetter: true })
    : 'none'
  const trendAssists = teamAvgKpis
    ? computeTrend(kpis.assists_per_game, teamAvgKpis.assists_per_game)
    : 'none'
  const trendAcc = teamAvgKpis
    ? computeTrend(kpis.avg_accuracy, teamAvgKpis.avg_accuracy)
    : 'none'
  const trendLife = teamAvgKpis
    ? computeTrend(kpis.avg_life_seconds, teamAvgKpis.avg_life_seconds)
    : 'none'

  // Header row : affichée uniquement quand un titre existe (mode self).
  // En mode drilled (title vide), la reset bar "Vue active : X" indique déjà
  // le scope ; on évite l'empty line + on supprime le hint trend redondant
  // (les ▲/▼ colorés à côté de chaque value sont auto-explicites).
  return (
    <div>
      {title && (
        <div className="mb-1.5 flex items-center justify-between px-1">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {title}
          </span>
          {teamAvgKpis && hint && (
            <span className="text-[10px] text-muted-foreground">{hint}</span>
          )}
        </div>
      )}
      <div className="grid grid-cols-8 gap-2">
        <KpiCell
          label={texts.grid.matchesPlayed}
          value={String(kpis.matches_count)}
        />
        <KpiCell
          label={texts.grid.totalDuration}
          value={formatDurationDhm(kpis.total_play_seconds)}
        />
        <KpiCell
          label={texts.grid.avgMatchDuration}
          value={formatMmss(kpis.avg_match_seconds)}
        />
        <KpiCell
          label={texts.grid.fragsPerMatch}
          value={kpis.kills_per_game.toFixed(2)}
          sub={`${kpis.kills_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendKills}
        />
        <KpiCell
          label={texts.grid.deathsPerMatch}
          value={kpis.deaths_per_game.toFixed(2)}
          sub={`${kpis.deaths_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendDeaths}
        />
        <KpiCell
          label={texts.grid.assistsPerMatch}
          value={kpis.assists_per_game.toFixed(2)}
          sub={`${kpis.assists_per_minute.toFixed(2)}${texts.grid.perMin}`}
          trend={trendAssists}
        />
        <KpiCell
          label={texts.grid.accuracy}
          value={`${kpis.avg_accuracy.toFixed(2)}%`}
          trend={trendAcc}
        />
        <KpiCell
          label={texts.grid.lifespan}
          value={formatMmss(kpis.avg_life_seconds)}
          trend={trendLife}
        />
      </div>
    </div>
  )
}
