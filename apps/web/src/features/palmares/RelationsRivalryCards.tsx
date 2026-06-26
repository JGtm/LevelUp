/**
 * RelationsRivalryCards — cartes « Revanche » (Phase 3a).
 *
 * Pour chaque rival (bête noire + autres), une carte :
 *   - frise des duels (OutcomeSequenceTape, ancien→récent, tokens outcome-*)
 *   - taux de victoire glissant (TimeseriesLineChart, 1 série, lissée)
 *   - KPIs : récent vs global, série en cours, écart de frags cumulé.
 *
 * Aucune couleur hex : tokens outcome-* (via OutcomeSequenceTape) +
 * colorToken 'outcome-win' (courbe). Strings via palmares.toml (FR/EN).
 */
import { useMemo } from 'react'

import { OutcomeSequenceTape, type OutcomePoint, type OutcomeSequenceLabels } from '@/components/charts/OutcomeSequenceTape'
import { TimeseriesLineChart, type ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { formatPercent } from '@/lib/formatters'
import type { RelationRivalry } from '@/lib/api/types'

import type { PalmaresText } from './i18n'

type MomentsText = PalmaresText['relations']['moments']

// outcome backend "win"|"loss"|"other" → clé OutcomeSequenceTape ("other"→tie,
// couleur neutre via outcome-draw token, déjà câblé dans le wrapper).
function toTapePoints(rivalry: RelationRivalry): OutcomePoint[] {
  return (rivalry.duels ?? []).map((d) => ({
    outcome: d.outcome === 'win' ? 'win' : d.outcome === 'loss' ? 'loss' : 'tie',
    matchId: d.match_id,
  }))
}

function streakLabel(streak: number, t: MomentsText): string {
  if (streak > 0) return t.streakWins(String(streak))
  if (streak < 0) return t.streakLosses(String(-streak))
  return t.streakNone
}

function fragGapLabel(gap: number, t: MomentsText): string {
  if (gap > 0) return t.fragGapAhead(String(gap))
  if (gap < 0) return t.fragGapBehind(String(gap))
  return t.fragGapEven
}

function RivalryCard({ rivalry, t }: { rivalry: RelationRivalry; t: MomentsText }) {
  const tapeLabels: OutcomeSequenceLabels = {
    win: t.outcomeWin,
    loss: t.outcomeLoss,
    tie: t.outcomeOther,
    dnf: t.outcomeOther,
  }
  const tapePoints = useMemo(() => toTapePoints(rivalry), [rivalry])

  // WR glissant : un point par duel (index), valeur en % (0..100). Les points
  // nuls (fenêtre sans duel décisif) sont omis.
  const rollingSeries: ChartSeries<ChartPoint2D>[] = useMemo(() => {
    const dp: ChartPoint2D[] = []
    ;(rivalry.rolling_win_rate ?? []).forEach((v, i) => {
      if (v != null) dp.push({ x: i + 1, y: Math.round(v * 1000) / 10 })
    })
    return dp.length > 0 ? [{ key: 'rolling', colorToken: 'outcome-win', datapoints: dp }] : []
  }, [rivalry])

  return (
    <div className="flex flex-col gap-2 rounded-lg bg-card p-4">
      <div className="flex items-baseline justify-between gap-2">
        <p className="truncate text-sm font-semibold text-foreground">{rivalry.gamertag}</p>
        <p className="shrink-0 text-xs text-muted-foreground">{t.enemyMatches(String(rivalry.enemy_matches))}</p>
      </div>

      <OutcomeSequenceTape matches={tapePoints} labels={tapeLabels} height={64} />

      <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
        <div>
          <span className="text-muted-foreground">{t.recentWinRate} : </span>
          <span className="font-mono text-foreground">{formatPercent(rivalry.recent_win_rate, 0)}</span>
        </div>
        <div>
          <span className="text-muted-foreground">{t.globalWinRate} : </span>
          <span className="font-mono text-foreground">{formatPercent(rivalry.global_win_rate, 0)}</span>
        </div>
        <div className="col-span-2 text-muted-foreground">{streakLabel(rivalry.current_streak, t)}</div>
        <div className="col-span-2 text-muted-foreground">{fragGapLabel(rivalry.frag_gap, t)}</div>
      </div>

      {rollingSeries.length > 0 && (
        <div>
          <p className="mb-1 text-xs text-muted-foreground">{t.rollingTitle}</p>
          <TimeseriesLineChart
            series={rollingSeries}
            xAxisType="category"
            outcomeMarkers={false}
            showSymbol={false}
            smooth
            height={120}
          />
        </div>
      )}
    </div>
  )
}

export function RelationsRivalryCards({ rivalries, t }: { rivalries: RelationRivalry[]; t: MomentsText }) {
  if (rivalries.length === 0) {
    return <p className="text-sm text-muted-foreground">{t.rivalriesEmpty}</p>
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {rivalries.map((r) => (
        <RivalryCard key={r.xuid} rivalry={r} t={t} />
      ))}
    </div>
  )
}
