/**
 * SessionBriefing — composant principal.
 *
 * Fusion KPI bar + Squad verdict en un seul briefing structuré en 3 bandes :
 *   1. <ResultsRail>  : descriptif (Matchs / Durée / Résultats avec libellés)
 *   2. <SquadVerdict> : score d'équipe + N+1 cards joueurs cliquables (squad only)
 *   3. <KpiGrid>      : 7 cards KPI avec trends ▲/▼ vs moyenne d'équipe
 *
 * Mode solo : `squad` undefined → bande verdict masquée + trends ▲/▼ masqués
 * dans la grille (pas de référence sans escouade).
 *
 * Mode squad : `squad` fourni → verdict visible, drill-down click sur card joueur
 * recalcule la KpiGrid pour ce joueur via state local. La moyenne d'équipe
 * (`squad.teamAvgKpis`) reste la référence des trends quel que soit le drilled.
 *
 * Multi-titres : libellés outcomes via useOutcomeLabel() (outcomes.toml).
 */
import { useState } from 'react'

import type { KPIStats } from '@/lib/api/types'
import type { PlayerScoreCard, SquadScoreCard } from '@/features/squad/v2/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { KpiGrid } from './KpiGrid'
import { ResultsRail } from './ResultsRail'
import { SquadVerdict } from './SquadVerdict'
import { getBriefingTexts, type BriefingLocale } from './i18n'

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

function normalizeLocale(input: string | undefined): BriefingLocale {
  return input === 'en' ? 'en' : 'fr'
}

export function SessionBriefing({ kpis, squad }: SessionBriefingProps) {
  const locale = useAppShellStore((s) => s.locale)
  const texts = getBriefingTexts(normalizeLocale(locale))

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
    ? squad?.players.find((p) => p.xuid === viewedXuid)?.gamertag ?? viewedXuid
    : ''

  // Quand drillé, on n'affiche pas de titre redondant : la verdict band
  // highlight déjà la card du joueur viewé + la reset bar "Vue active : X"
  // indique le scope. Le titre "Mes stats sur cette session" reste pour le
  // mode self.
  const gridTitle = isDrilledIn ? '' : texts.grid.titleSelf

  return (
    <div className="flex flex-col gap-2">
      <ResultsRail kpis={viewedKpis} texts={texts} />

      {squad && (
        <SquadVerdict
          squadScore={squad.score}
          players={squad.players}
          activeXuid={activeXuid}
          viewedXuid={viewedXuid}
          onSelectXuid={setViewedXuid}
          texts={texts}
        />
      )}

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
      />
    </div>
  )
}
