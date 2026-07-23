/**
 * SessionBriefing — composant principal.
 *
 * Structure :
 *   - <SquadVerdict> (squad only) : team card + N+1 player cards + Results bar (right)
 *   - <KpiGrid> (toujours) : 8 cards KPI (Matchs / Durée totale / Durée moy/match
 *     / Frags / Morts / Assists / Précision / Vie). Trends ▲/▼ vs moyenne d'équipe
 *     (squad mode uniquement).
 *
 * Mode solo : pas de bande verdict, pas de Results bar (la Results bar n'est
 * affichée qu'en mode squad — décision UX). Les KPIs descriptifs (Matchs joués,
 * durées) sont dans la grille.
 *
 * Mode squad : verdict band visible, drill-down click sur card joueur recalcule
 * la KpiGrid pour ce joueur via state local. La moyenne d'équipe reste la
 * référence des trends quel que soit le drilled.
 *
 * Multi-titres : libellés outcomes via useOutcomeLabel() (outcomes.toml).
 */
import { useMemo, useState } from 'react'

import type { KPIStats } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'
import type { PlayerScoreCard, SquadScoreCard } from '@/features/squad/v2/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { KpiGrid } from './KpiGrid'
import { SquadVerdict } from './SquadVerdict'
import { getBriefingTexts } from './i18n'
import type { Locale } from '@/lib/i18n/locale'
import { getScoreTier } from './tier'

export interface SessionBriefingSquadProps {
  score: SquadScoreCard
  players: PlayerScoreCard[]
  /** map xuid → KPIStats — drill-down click. */
  kpisByXuid: Record<string, KPIStats>
  /** Moyenne d'équipe — référence pour trends ▲/▼. */
  teamAvgKpis: KPIStats
  /** xuid du joueur principal (main). */
  activeXuid: string
}

export interface SessionBriefingProps {
  /** KPIs du joueur principal sur le scope filtré (mode solo + initial mode squad). */
  kpis: KPIStats
  /** Données squad. Sans = mode solo (verdict masqué, trends masqués). */
  squad?: SessionBriefingSquadProps
}

function normalizeLocale(input: string | undefined): Locale {
  return input === 'en' ? 'en' : 'fr'
}

export function SessionBriefing({ kpis, squad }: SessionBriefingProps) {
  const locale = useAppShellStore((s) => s.locale)
  const gamertag = useAppShellStore((s) => s.currentPlayer?.gamertag ?? '')
  const texts = getBriefingTexts(normalizeLocale(locale))

  // Mode solo avec performance_score : construit une carte joueur unique (moi).
  const soloPlayers = useMemo<PlayerScoreCard[] | undefined>(() => {
    if (squad != null || kpis.performance_score == null) return undefined
    const score = kpis.performance_score
    const total = kpis.outcomes.wins + kpis.outcomes.losses + kpis.outcomes.ties + kpis.outcomes.dnf
    return [{
      xuid: '',
      gamertag,
      score,
      label: getScoreTier(score).key,
      comparison: 'near' as const,
      kd_ratio: kpis.deaths_per_game > 0 ? kpis.kills_per_game / kpis.deaths_per_game : kpis.kills_per_game,
      win_rate: total > 0 ? kpis.outcomes.wins / total : 0,
      accuracy: kpis.avg_accuracy,
      kills: Math.round(kpis.kills_per_game * kpis.matches_count),
    }]
  }, [squad, kpis, gamertag])

  const [viewedXuid, setViewedXuid] = useState<string>(squad?.activeXuid ?? '')

  // Active player xuid : si squad fourni, c'est squad.activeXuid. Sinon vide.
  const activeXuid = squad?.activeXuid ?? ''
  const isDrilledIn = squad != null && viewedXuid !== '' && viewedXuid !== activeXuid

  // KPIs affichés dans la grille : drill-down si applicable, sinon kpis du main.
  const viewedKpis: KPIStats =
    isDrilledIn && squad
      ? (squad.kpisByXuid[viewedXuid] ?? kpis)
      : kpis

  // Référence des trends : team avg (squad mode) ou null (solo).
  const teamAvgKpis = squad?.teamAvgKpis ?? null

  // Gamertag affiché dans le titre de la grille (drilled).
  const drilledGamertag = isDrilledIn
    ? displayPlayerName(
        squad?.players?.find((p) => p.xuid === viewedXuid)?.gamertag,
        viewedXuid,
      )
    : ''

  // Quand drillé, on n'affiche pas de titre redondant : la verdict band
  // highlight déjà la card du joueur viewé + la reset bar "Vue active : X"
  // indique le scope. Le titre "Mes stats sur cette session" reste pour le
  // mode self.
  const gridTitle = isDrilledIn ? '' : texts.grid.titleSelf

  return (
    <div className="flex flex-col gap-2">
      <SquadVerdict
        squadScore={squad?.score}
        players={squad?.players ?? soloPlayers}
        activeXuid={activeXuid}
        viewedXuid={viewedXuid}
        onSelectXuid={setViewedXuid}
        kpis={viewedKpis}
        texts={texts}
      />

      {isDrilledIn && (
        <div className="flex items-center gap-2 px-1">
          <span className="text-xs text-muted-foreground">
            {texts.drill.activeView(drilledGamertag)}
          </span>
          <button
            type="button"
            onClick={() => setViewedXuid(activeXuid)}
            className="text-xs text-muted-foreground underline hover:text-foreground"
          >
            {texts.drill.resetButton}
          </button>
        </div>
      )}

      <KpiGrid
        kpis={viewedKpis}
        teamAvgKpis={teamAvgKpis}
        texts={texts}
        title={gridTitle}
        hint={texts.grid.trendHint}
        omitSummaryCards={true}
      />
    </div>
  )
}
