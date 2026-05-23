/**
 * KpiGrid — bande KPI du SessionBriefing.
 *
 * Mode SOLO (par défaut) : 7 cards permanentes (Matchs+durée moy / Durée
 * totale / Frags / Morts / Assists / Précision / Vie) + 1 conditionnelle
 * (Delta rang) → 7 ou 8 colonnes.
 *
 * Mode SQUAD (omitSummaryCards=true) : "Matchs joués" et "Durée totale"
 * sont déplacées dans SquadVerdict (à droite de la Results bar). La grille
 * descend à 5 cards permanentes + 1 conditionnelle → 5 ou 6 colonnes.
 *
 * "Matchs joués" affiche le count + la durée moyenne par match en inline-sub
 * (ex: "12  8min07/match"). Cette présentation est répliquée dans
 * SquadVerdict en mode squad pour préserver l'info.
 *
 * Trends ▲/▼ vs teamAvgKpis (mode squad uniquement) sur les 5 cards
 * évaluatives historiques (frags, morts, assists, précision, vie). Morts =
 * lower_is_better. Si teamAvgKpis est null (mode solo) → trends 'none'.
 *
 * Delta rang : couleur ABSOLUE par SIGNE (divergent-pos/neg/neutral). Kind
 * détermine le label (Delta CSR / Delta LUSR) et la précision (CSR=int,
 * LUSR=2 décimales). Pas de glyphe trend (la couleur + le signe explicite
 * sur la valeur suffisent).
 */
import type { CSSProperties } from 'react'

import type { KPIStats, RankDelta } from '@/lib/api/types'

import { formatDurationDhm, formatMinSec, formatMmss } from './format'
import type { BriefingTexts } from './i18n'
import { computeTrend, trendSymbol, type TrendState } from './trends'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

interface KpiGridProps {
  kpis: KPIStats
  /** Référence pour calcul trends ▲/▼. Null = mode solo, pas de trends. */
  teamAvgKpis: KPIStats | null
  texts: BriefingTexts
  /** Titre de la grille — gestionnaire externe (drilled vs self) */
  title: string
  /** Hint affiché à droite du titre — affiché seulement si teamAvgKpis présent */
  hint?: string
  /** Si true, retire les cards "Matchs joués" et "Durée totale" de la grille
   *  car elles sont rendues ailleurs (typiquement SquadVerdict en mode squad). */
  omitSummaryCards?: boolean
}

interface CellProps {
  label: string
  value: string
  /** Sub-info affichée en dessous (ex: "1.20/min" sous frags par partie). */
  sub?: string
  /** Sub-info affichée à droite de la valeur sur la même ligne, plus petite
   *  et muted. Utile pour combiner deux infos liées dans une seule colonne
   *  (ex: count matchs + durée moyenne par match). */
  inlineSub?: string
  trend?: TrendState
  /** Couleur absolue de la valeur. Si fourni, override la couleur trend. */
  valueColorToken?: SemanticToken
}

function KpiCell({ label, value, sub, inlineSub, trend = 'none', valueColorToken }: CellProps) {
  const trendToken =
    trend === 'above'
      ? 'divergent-pos'
      : trend === 'below'
        ? 'divergent-neg'
        : trend === 'near'
          ? 'divergent-neutral'
          : null

  const valueStyle: CSSProperties | undefined = valueColorToken
    ? { color: tokenCssVar(valueColorToken) }
    : undefined

  return (
    <div className="rounded border border-border bg-card px-3 py-2">
      <p className="text-3xs uppercase tracking-wide text-muted-foreground">{label}</p>
      <div className="mt-0.5 flex items-baseline">
        <span className="text-lg font-bold" style={valueStyle}>{value}</span>
        {inlineSub && (
          <span className="ml-1.5 text-3xs text-muted-foreground">{inlineSub}</span>
        )}
        {!valueColorToken && trendToken && (
          <span
            className="ml-1 text-xs font-bold"
            style={{ color: tokenCssVar(trendToken) }}
          >
            {trendSymbol(trend)}
          </span>
        )}
      </div>
      {sub && <p className="mt-0.5 text-2xs text-muted-foreground">{sub}</p>}
    </div>
  )
}

/** Formate la valeur signée d'un RankDelta. CSR = entier, LUSR = 2 décimales.
 *  Préfixe explicite (+/−/±) pour que le signe soit lisible sans dépendre
 *  uniquement de la couleur. */
function formatRankDeltaValue(delta: RankDelta): string {
  const isCsr = delta.kind === 'csr'
  const v = delta.value
  if (v === 0) return isCsr ? '±0' : '±0.00'
  const abs = Math.abs(v)
  const formatted = isCsr ? String(Math.round(abs)) : abs.toFixed(2)
  return v > 0 ? `+${formatted}` : `−${formatted}`
}

/** Token couleur absolue selon le signe du delta : pos/neg/neutral. */
function rankDeltaColorToken(value: number): SemanticToken {
  if (value > 0) return 'divergent-pos'
  if (value < 0) return 'divergent-neg'
  return 'divergent-neutral'
}

export function KpiGrid({ kpis, teamAvgKpis, texts, title, hint, omitSummaryCards = false }: KpiGridProps) {
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

  // Cards conditionnelles : rank_delta et rendement/résistance si données dispo.
  const hasDelta = kpis.rank_delta != null
  const hasOC = kpis.avg_offensive_conversion != null
  const hasDR = kpis.avg_defensive_resistance != null
  const baseCols = omitSummaryCards ? 5 : 7
  const colCount = baseCols + (hasDelta ? 1 : 0) + (hasOC ? 1 : 0) + (hasDR ? 1 : 0)
  // Tailwind ne purge pas les `grid-cols-N` dynamiques → utiliser inline
  // grid-template-columns pour un colCount calculé.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `repeat(${colCount}, minmax(0, 1fr))`,
  }

  // Header row : affichée uniquement quand un titre existe (mode self).
  // En mode drilled (title vide), la reset bar "Vue active : X" indique déjà
  // le scope ; on évite l'empty line + on supprime le hint trend redondant
  // (les ▲/▼ colorés à côté de chaque value sont auto-explicites).
  return (
    <div>
      {title && (
        <div className="mb-1.5 flex items-center justify-between px-1">
          <span className="text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
            {title}
          </span>
          {teamAvgKpis && hint && (
            <span className="text-2xs text-muted-foreground">{hint}</span>
          )}
        </div>
      )}
      <div className="grid gap-2" style={gridStyle}>
        {!omitSummaryCards && (
          <>
            <KpiCell
              label={texts.grid.matchesPlayed}
              value={String(kpis.matches_count)}
              inlineSub={`${formatMinSec(kpis.avg_match_seconds)}${texts.grid.perMatch}`}
            />
            <KpiCell
              label={texts.grid.totalDuration}
              value={formatDurationDhm(kpis.total_play_seconds)}
            />
          </>
        )}
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
        {hasOC && (
          <KpiCell
            label={texts.grid.rendement}
            value={`${((kpis.avg_offensive_conversion ?? 0) * 100).toFixed(0)}%`}
            sub={texts.grid.refBaseline}
          />
        )}
        {hasDR && (
          <KpiCell
            label={texts.grid.resistance}
            value={`${((kpis.avg_defensive_resistance ?? 0) * 100).toFixed(0)}%`}
            sub={texts.grid.refBaseline}
          />
        )}
        {hasDelta && (
          <KpiCell
            label={
              kpis.rank_delta!.kind === 'csr'
                ? texts.grid.rankDeltaCSR
                : texts.grid.rankDeltaLUSR
            }
            value={formatRankDeltaValue(kpis.rank_delta!)}
            valueColorToken={rankDeltaColorToken(kpis.rank_delta!.value)}
          />
        )}
      </div>
    </div>
  )
}
