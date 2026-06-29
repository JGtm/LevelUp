/**
 * RelationsRivalryCards — cartes « Revanche » (Phase 3a / v3).
 *
 * Pour chaque rival (bête noire + autres), une carte :
 *   - frise des duels (OutcomeSequenceTape, ancien→récent) ; au survol, chaque
 *     duel affiche date · mode · map — frags/morts (plus d'UUID brut).
 *   - écart de frags cumulé (CumulativeFragGapChart) coloré par le signe :
 *     vert quand tu mènes, rouge quand tu es derrière.
 *   - KPIs : récent vs global, série en cours, écart de frags total.
 *
 * Aucune couleur hex : tokens outcome-* (via les wrappers chart).
 */
import { useMemo } from 'react'

import {
  OutcomeSequenceTape,
  type OutcomePoint,
  type OutcomeSequenceLabels,
} from '@/components/charts/OutcomeSequenceTape'
import { tokenCssVar } from '@/lib/accessibility'
import { formatPercent } from '@/lib/formatters'
import type { RelationRivalry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { CumulativeFragGapChart } from './CumulativeFragGapChart'
import { normalizePalmaresLocale, type PalmaresText } from './i18n'

type MomentsText = PalmaresText['relations']['moments']

// wrColor : vert si WR >= 50 %, rouge sinon (undefined si inconnu).
function wrColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  return v >= 0.5 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
}

// streakChip : pastille compacte « N victoires/défaites de suite » (>0 victoires,
// <0 défaites). Masquée si |série| < 2 (pas une vraie série).
function streakChip(streak: number, t: MomentsText) {
  if (streak >= 2) {
    return (
      <span className="font-mono font-bold" style={{ color: tokenCssVar('outcome-win') }}>
        {t.streakWins(String(streak))}
      </span>
    )
  }
  if (streak <= -2) {
    return (
      <span className="font-mono font-bold" style={{ color: tokenCssVar('outcome-loss') }}>
        {t.streakLosses(String(-streak))}
      </span>
    )
  }
  return null
}

// toTapePoints : duels backend → points de frise. Le tooltip d'un duel affiche un
// libellé pré-formaté (date · mode · map — frags/morts) au lieu de l'UUID.
function toTapePoints(rivalry: RelationRivalry, locale: 'fr' | 'en'): OutcomePoint[] {
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

function RivalryCard({ rivalry, t, locale }: { rivalry: RelationRivalry; t: MomentsText; locale: 'fr' | 'en' }) {
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
    <div className="flex flex-col gap-2 rounded-lg bg-card p-4">
      <div className="flex items-baseline justify-between gap-2">
        <p className="truncate text-sm font-semibold text-foreground">{rivalry.gamertag}</p>
        <p className="shrink-0 text-xs text-muted-foreground">{t.enemyMatches(String(rivalry.enemy_matches))}</p>
      </div>

      <OutcomeSequenceTape matches={tapePoints} labels={tapeLabels} height={64} />

      {/* compact : récent vs global + série (écart de frags total retiré = fin du graphe) */}
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-xs">
        <span>
          <span className="text-muted-foreground">{t.recentShort} </span>
          <span className="font-mono font-bold" style={{ color: wrColor(rivalry.recent_win_rate) }}>
            {formatPercent(rivalry.recent_win_rate, 0)}
          </span>
        </span>
        <span>
          <span className="text-muted-foreground">{t.globalShort} </span>
          <span className="font-mono font-bold" style={{ color: wrColor(rivalry.global_win_rate) }}>
            {formatPercent(rivalry.global_win_rate, 0)}
          </span>
        </span>
        {streakChip(rivalry.current_streak, t)}
      </div>

      {cumulativePoints.length > 0 && (
        <div className="flex flex-col gap-1">
          <p className="text-xs text-muted-foreground">{t.cumulativeFragTitle}</p>
          <CumulativeFragGapChart points={cumulativePoints} height={120} />
        </div>
      )}
    </div>
  )
}

export function RelationsRivalryCards({ rivalries, t }: { rivalries: RelationRivalry[]; t: MomentsText }) {
  const locale = normalizePalmaresLocale(useAppShellStore((s) => s.locale))
  if (rivalries.length === 0) {
    return <p className="text-sm text-muted-foreground">{t.rivalriesEmpty}</p>
  }
  return (
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {rivalries.map((r) => (
        <RivalryCard key={r.xuid} rivalry={r} t={t} locale={locale} />
      ))}
    </div>
  )
}
