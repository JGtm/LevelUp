/**
 * SquadVerdict — bande verdict squad du SessionBriefing.
 *
 * 3 zones séparées visuellement :
 *   - LEFT   : team card (Score d'équipe + palier + Δ vs base)
 *   - CENTER : N+1 player cards cliquables (drill-down click)
 *   - RIGHT  : Results bar (4 segments + libellés Victoire/Défaite/...) +
 *              2 mini-cards "Matchs joués" + "Durée totale" en colonne à droite.
 *
 * Le team card, la Results bar et les mini-cards sont des "résumés" de la
 * session collective ; ils encadrent la zone clickable des player cards.
 * Bande pas affichée en mode solo (les cards Matchs/Durée restent alors dans
 * KpiGrid).
 *
 * Libellés outcomes : tirés de outcomes.toml via useOutcomeLabel() (multi-titres).
 */
import type { KPIStats } from '@/lib/api/types'
import type { PlayerScoreCard, SquadScoreCard } from '@/features/squad/v2/types'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { useOutcomeLabel } from '@/lib/i18n/fieldMappings'

import { formatDurationDhm, formatMinSec } from './format'
import { getScoreTier, type TierKey } from './tier'
import { trendSymbol, type TrendState } from './trends'
import type { BriefingTexts } from './i18n'

interface SquadVerdictProps {
  squadScore: SquadScoreCard
  players: PlayerScoreCard[]
  /** xuid du joueur principal (main) — affichage badge "(moi)" sur sa card. */
  activeXuid: string
  /** xuid actuellement affiché dans la KpiGrid (drill-down). */
  viewedXuid: string
  onSelectXuid: (xuid: string) => void
  /** KPIs du joueur viewé — utilisé pour la Results bar à droite. */
  kpis: KPIStats
  texts: BriefingTexts
}

const TIER_LABEL_FALLBACK: Record<TierKey, string> = {
  excellent: 'Excellent',
  good: 'Solide',
  average: 'Correct',
  poor: 'Mauvais',
  bad: 'Pourri',
}

function tierLabelOf(card: PlayerScoreCard): string {
  if (card.label) return TIER_LABEL_FALLBACK[card.label]
  return TIER_LABEL_FALLBACK[getScoreTier(card.score).key]
}

interface OutcomeSeg {
  count: number
  token: 'outcome-win' | 'outcome-loss' | 'outcome-draw' | 'outcome-dnf'
  outcomeKey: string
  fallbackLabel: string
}

export function SquadVerdict({
  squadScore,
  players,
  activeXuid,
  viewedXuid,
  onSelectXuid,
  kpis,
  texts,
}: SquadVerdictProps) {
  const teamTier = getScoreTier(squadScore.score)
  const teamLabel = TIER_LABEL_FALLBACK[teamTier.key]

  // delta = bonus de cohésion (winrate >60%, min KDA >1, faible variance kills).
  // Quand le bonus vaut 0, on n'affiche RIEN — pas la peine d'encombrer la card
  // avec « base only » : l'absence de label = score brut sans bonus.
  const delta = Math.round(squadScore.score - squadScore.base_avg)
  const deltaText =
    delta > 0
      ? texts.verdict.deltaBonusPositive(delta)
      : delta < 0
        ? texts.verdict.deltaBonusNegative(delta)
        : null
  const deltaToken: SemanticToken =
    delta > 0 ? 'divergent-pos' : delta < 0 ? 'divergent-neg' : 'divergent-neutral'

  // Results bar (right section)
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
    <div className="flex flex-wrap items-stretch gap-4 rounded border border-border bg-background px-4 py-3">
      {/* LEFT : team card — non cliquable */}
      <div className="min-w-[180px] rounded border border-border bg-card px-3 py-2">
        <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
          {texts.verdict.teamScore}
        </p>
        <div className="mt-1 flex items-baseline gap-2">
          <span
            className="text-2xl font-bold"
            style={{ color: tokenCssVar(teamTier.token) }}
          >
            {Math.round(squadScore.score)}
          </span>
          <span className="text-xs text-muted-foreground">{teamLabel}</span>
        </div>
        {deltaText !== null && (
          <p
            className="mt-0.5 text-[10px]"
            style={{ color: tokenCssVar(deltaToken) }}
          >
            {deltaText}
          </p>
        )}
      </div>

      {/* CENTER : player cards — cliquables */}
      <div className="flex flex-wrap items-stretch gap-2">
        {players.map((p) => {
          const tier = getScoreTier(p.score)
          const isActive = p.xuid === activeXuid
          const isViewed = p.xuid === viewedXuid
          const trendState: TrendState =
            p.comparison === 'above' || p.comparison === 'below' || p.comparison === 'near'
              ? p.comparison
              : 'none'
          return (
            <button
              key={p.xuid || p.gamertag}
              type="button"
              onClick={() => p.xuid && onSelectXuid(p.xuid)}
              disabled={!p.xuid}
              className={[
                'rounded border px-3 py-2 text-left transition cursor-pointer',
                'hover:border-foreground/40 disabled:cursor-default disabled:opacity-60',
                isViewed
                  ? 'border-foreground/60 bg-secondary'
                  : 'border-border bg-card',
              ].join(' ')}
            >
              <div className="flex items-center justify-between gap-2">
                <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
                  {p.gamertag}
                  {isActive && (
                    <span className="ml-1 text-[10px] opacity-60">(moi)</span>
                  )}
                </p>
                {trendState !== 'none' && (
                  <span
                    className="text-xs font-bold"
                    style={{
                      color: tokenCssVar(
                        trendState === 'above'
                          ? 'divergent-pos'
                          : trendState === 'below'
                            ? 'divergent-neg'
                            : 'divergent-neutral',
                      ),
                    }}
                  >
                    {trendSymbol(trendState)}
                  </span>
                )}
              </div>
              <div className="mt-1 flex items-baseline gap-2">
                <span
                  className="text-2xl font-bold"
                  style={{ color: tokenCssVar(tier.token) }}
                >
                  {Math.round(p.score)}
                </span>
                <span className="text-xs text-muted-foreground">{tierLabelOf(p)}</span>
              </div>
            </button>
          )
        })}
      </div>

      {/* RIGHT : Results bar + mini-cards Matchs/Durée alignées à droite. */}
      <div className="ml-auto flex items-stretch gap-3">
        {hasResults && (
          <div className="flex min-w-[260px] flex-col gap-1.5">
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
        {/* Mini-cards : Matchs joués (avec durée moy/match en inline-sub) +
            Durée totale. Affichées côte à côte à droite de la Results bar pour
            libérer 2 colonnes du KpiGrid en mode squad. Pas de fond ni de
            bordure : ces cards sont des résumés contextuels intégrés au
            bandeau verdict, le bandeau lui-même portant déjà le cadre. */}
        <div className="flex items-center gap-6 px-2">
          <div>
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
              {texts.grid.matchesPlayed}
            </p>
            <div className="mt-0.5 flex items-baseline">
              <span className="text-lg font-bold">{kpis.matches_count}</span>
              <span className="ml-1.5 text-[11px] text-muted-foreground">
                {formatMinSec(kpis.avg_match_seconds)}{texts.grid.perMatch}
              </span>
            </div>
          </div>
          <div>
            <p className="text-[11px] uppercase tracking-wide text-muted-foreground">
              {texts.grid.totalDuration}
            </p>
            <div className="mt-0.5">
              <span className="text-lg font-bold">
                {formatDurationDhm(kpis.total_play_seconds)}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
