/**
 * SquadVerdict — bande verdict squad du SessionBriefing.
 *
 * Affichée uniquement en mode squad (≥1 coéquipier sélectionné). Composée de :
 *   - Team card fixe (score d'équipe + grade lettre + Δ vs base)
 *   - N+1 cards joueurs cliquables (main + chaque coéquipier) avec ▲/▼ vs avg
 *
 * Le clic sur une card joueur déclenche onSelectXuid(xuid) → drill-down dans la
 * KpiGrid en aval. La card "viewée" est mise en évidence (border + bg).
 */
import type { PlayerScoreCard, SquadScoreCard } from '@/features/squad/v2/types'
import { tokenCssVar } from '@/lib/accessibility'

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
  texts: BriefingTexts
}

const TIER_LABEL_FALLBACK: Record<TierKey, string> = {
  excellent: 'Excellent',
  good: 'Solide',
  average: 'Correct',
  poor: 'Mauvais',
  bad: 'Pourri',
}

function tierLabelOf(card: PlayerScoreCard | { score: number; label?: TierKey }): string {
  // Si le card vient avec un label (PlayerScoreCard), on l'utilise. Sinon
  // dérivation depuis le score (cas SquadScore qui n'a pas de label).
  const label = (card as PlayerScoreCard).label
  if (label) return TIER_LABEL_FALLBACK[label]
  return TIER_LABEL_FALLBACK[getScoreTier(card.score).key]
}

export function SquadVerdict({
  squadScore,
  players,
  activeXuid,
  viewedXuid,
  onSelectXuid,
  texts,
}: SquadVerdictProps) {
  const teamTier = getScoreTier(squadScore.score)
  const teamLabel = tierLabelOf({ score: squadScore.score })

  // Δ vs base : score = base + bonus_winrate + bonus_min_kd + bonus_balance
  const delta = Math.round(squadScore.score - squadScore.base_avg)
  const deltaText =
    delta === 0
      ? texts.verdict.baseOnly
      : delta > 0
        ? texts.verdict.deltaBonusPositive(delta)
        : texts.verdict.deltaBonusNegative(delta)
  const deltaToken =
    delta > 0 ? 'divergent-pos' : delta < 0 ? 'divergent-neg' : 'divergent-neutral'

  return (
    <div className="flex flex-wrap items-stretch gap-2 rounded border border-border bg-[#16191d] px-4 py-3">
      {/* Team card — non cliquable */}
      <div className="min-w-[180px] rounded border border-border bg-[#1d2328] px-3 py-2">
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
          <span className="ml-1 text-base font-bold">[{squadScore.grade}]</span>
        </div>
        <p
          className="mt-0.5 text-[10px]"
          style={{ color: tokenCssVar(deltaToken) }}
        >
          {deltaText}
        </p>
      </div>

      {/* Player cards — cliquables */}
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
                ? 'border-foreground/60 bg-[#252b32]'
                : 'border-border bg-[#1d2328]',
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
  )
}
