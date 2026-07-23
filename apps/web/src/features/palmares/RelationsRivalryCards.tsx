/**
 * RelationsRivalryCards — cartes « Revanche » (Phase 3a / v3).
 *
 * Pour chaque rival (bête noire + autres), une carte :
 *   - frise des duels (OutcomeSequenceTape, ancien→récent) ; au survol, chaque
 *     duel affiche date · mode · map — frags/morts (plus d'UUID brut).
 *   - écart de frags cumulé (CumulativeFragGapChart) coloré par le signe :
 *     vert quand tu mènes, rouge quand tu es derrière.
 *   - KPIs : taux de victoire récent vs global.
 *
 * Aucune couleur hex : tokens outcome-* (via les wrappers chart).
 */
import { useMemo } from 'react'
import { winRateColor } from '@/lib/colors/outcomePalette'

import { KpiCard } from '@/components/cards/KpiCard'
import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeSequenceLabels,
} from '@/components/charts/OutcomeSequenceTape'
import { formatPercent } from '@/lib/formatters'
import type { RelationRivalry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { CumulativeFragGapChart } from './CumulativeFragGapChart'
import { normalizePalmaresLocale, type PalmaresText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'

type MomentsText = PalmaresText['relations']['moments']

// wrColor : vert si WR >= 50 %, rouge sinon (undefined si inconnu).
// toTapePoints : duels backend → points de frise. Le tooltip d'un duel affiche un
// libellé pré-formaté (date · mode · map — frags/morts) au lieu de l'UUID.
function toTapePoints(rivalry: RelationRivalry, locale: Locale): OutcomePoint[] {
  return (rivalry.duels ?? []).map((d) => {
    const date = d.started_at
      ? new Date(d.started_at).toLocaleDateString(locale, { day: 'numeric', month: 'short' })
      : ''
    const parts = [date, d.mode, d.map_name].filter((s): s is string => !!s)
    const label = `${parts.join(' · ')} — ${d.kills_on_rival}/${d.deaths_by_rival}`
    return {
      outcome: d.outcome === 'win' ? 'win' : d.outcome === 'loss' ? 'loss' : 'tie',
      matchId: d.match_id,
      mode: d.mode || undefined,
      map: d.map_name || undefined,
      label,
    }
  })
}

function RivalryCard({
  rivalry,
  t,
  locale,
  onMatchClick,
}: {
  rivalry: RelationRivalry
  t: MomentsText
  locale: Locale
  onMatchClick?: (matchId: string) => void
}) {
  const tapeLabels: OutcomeSequenceLabels = {
    win: t.outcomeWin,
    loss: t.outcomeLoss,
    tie: t.outcomeOther,
    dnf: t.outcomeOther,
  }
  const tapePoints = useMemo(() => toTapePoints(rivalry, locale), [rivalry, locale])

  // Écart de frags cumulé (Σ frags − Σ morts) par duel, ancien→récent, + l'issue
  // du duel (couleur du symbole). Somme préfixe sans mutation (n ≤ 20).
  const cumulativePoints = useMemo(() => {
    const duels = rivalry.duels ?? []
    const deltas = duels.map((d) => d.kills_on_rival - d.deaths_by_rival)
    return duels.map((d, i) => ({
      cumulative: deltas.slice(0, i + 1).reduce((a, b) => a + b, 0),
      outcome: d.outcome,
    }))
  }, [rivalry])

  return (
    <KpiCard accent="outcome-loss" accentSide="top" className="flex flex-1 flex-col">
      <div className="flex flex-1 flex-col gap-2 p-4">
        {/* identité du rival (en blanc) + volume de duels */}
        <div className="flex items-baseline justify-between gap-2">
          <p className="truncate text-sm font-semibold text-foreground">{rivalry.gamertag}</p>
          <p className="shrink-0 text-xs text-muted-foreground">{t.enemyMatches(String(rivalry.enemy_matches))}</p>
        </div>

        <OutcomeSequenceTape matches={tapePoints} labels={tapeLabels} height={64} onMatchClick={onMatchClick} />

        {/* compact : taux de victoire récent vs global */}
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-xs">
          <span>
            <span className="text-muted-foreground">{t.recentShort} </span>
            <span className="font-mono font-bold" style={{ color: winRateColor(rivalry.recent_win_rate) }}>
              {formatPercent(rivalry.recent_win_rate, 0)}
            </span>
          </span>
          <span>
            <span className="text-muted-foreground">{t.globalShort} </span>
            <span className="font-mono font-bold" style={{ color: winRateColor(rivalry.global_win_rate) }}>
              {formatPercent(rivalry.global_win_rate, 0)}
            </span>
          </span>
        </div>

        {cumulativePoints.length > 0 && (
          <div className="flex flex-col gap-1">
            <p className="text-xs text-muted-foreground">{t.cumulativeFragTitle}</p>
            <CumulativeFragGapChart points={cumulativePoints} height={120} />
          </div>
        )}
      </div>
    </KpiCard>
  )
}

export function RelationsRivalryCards({
  rivalries,
  t,
  onMatchClick,
}: {
  rivalries: RelationRivalry[]
  t: MomentsText
  onMatchClick?: (matchId: string) => void
}) {
  const locale = normalizePalmaresLocale(useAppShellStore((s) => s.locale))
  if (rivalries.length === 0) {
    return <p className="text-sm text-muted-foreground">{t.rivalriesEmpty}</p>
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {rivalries.map((r) => (
        <RivalryCard key={r.xuid} rivalry={r} t={t} locale={locale} onMatchClick={onMatchClick} />
      ))}
    </div>
  )
}
