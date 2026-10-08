/**
 * SquadImpactHistoryCard — « Points d'impact par soirée et par rôle » (Escouade › Contributions,
 * sous la matrice d'impact ; maquette validée, forme « Détail des rôles »).
 *
 * Données : `squad_impact_history` (comptes, points et nets calculés côté Go). Mise en forme :
 * `impactHistory.logic.ts` ; option ECharts : `impactHistoryChart.ts`. Bloc absent quand la
 * réponse n'en porte pas (titre sans événements horodatés ni équipe alliée) : l'appelant ne le
 * monte pas.
 */
import { useCallback, useMemo, type CSSProperties } from 'react'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { SquadImpactHistory } from '@/lib/api/types'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'
import type { Locale } from '@/lib/i18n/locale'

import { buildImpactHistoryView, type ImpactHistoryView, type ImpactLegendRole } from './impactHistory.logic'
import { buildImpactHistoryOption, resolveImpactColors } from './impactHistoryChart'
import { IMPACT_HISTORY_TEXT, type ImpactHistoryText } from './impactHistoryStrings'

/** Hauteur du graphe (maquette : 360 px). */
const IMPACT_HISTORY_HEIGHT = 360
/** Largeur sous laquelle le graphe passe en forme téléphone (barres fines, une date sur deux). */
const NARROW_QUERY = '(max-width: 640px)'

interface Props {
  history: SquadImpactHistory
  /** gamertag → couleur résolue (palette des joueurs de l'Escouade). */
  colorByPlayer: Record<string, string>
  /** gamertag → encre CSS du joueur (légende). */
  inkOf: (gamertag: string) => string
  locale: Locale
}

export function SquadImpactHistoryCard({ history, colorByPlayer, inkOf, locale }: Props) {
  const t = IMPACT_HISTORY_TEXT[locale]
  const narrow = useMediaQuery(NARROW_QUERY)
  const view = useMemo(() => buildImpactHistoryView(history, locale, t), [history, locale, t])
  const series = useMemo<ChartSeries<ImpactHistoryView>[]>(
    () => (view ? [{ key: 'squad-impact-history', datapoints: [view] }] : []),
    [view],
  )
  const buildOption = useCallback(
    () =>
      view
        ? buildImpactHistoryOption(view, resolveImpactColors(view.players, colorByPlayer), t, narrow)
        : {},
    [view, colorByPlayer, t, narrow],
  )
  if (!view) return null
  return (
    <div className="min-w-0" data-testid="squad-impact-history">
      <ChartCard
        title={
          <span className="flex items-center gap-1.5">
            {t.title}
            <InfoTooltip content={view.info} />
          </span>
        }
        series={series}
        height={IMPACT_HISTORY_HEIGHT}
        renderer="svg"
        buildOption={buildOption}
        legend={<ImpactHistoryLegend view={view} inkOf={inkOf} t={t} />}
      />
    </div>
  )
}

/** La légende, en bas et centrée, sur trois lignes : nets des joueurs, gains, pertes. */
function ImpactHistoryLegend({ view, inkOf, t }: { view: ImpactHistoryView; inkOf: (gt: string) => string; t: ImpactHistoryText }) {
  return (
    <div className="grid justify-items-center gap-1.5 text-xs text-muted-foreground" aria-label={t.title}>
      <ul className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
        <li>{t.legendNet}</li>
        {view.players.map((gt) => (
          <li key={gt} className="inline-flex items-center gap-1.5">
            <NetSwatch color={inkOf(gt)} />
            {gt}
          </li>
        ))}
      </ul>
      <RoleRow roles={view.legend.gains} />
      <RoleRow roles={view.legend.losses} />
    </div>
  )
}

function RoleRow({ roles }: { roles: ImpactLegendRole[] }) {
  return (
    <ul className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
      {roles.map((r) => (
        <li key={r.label} className="inline-flex items-center gap-1.5">
          <span className="inline-block h-2.5 w-2.5 shrink-0 rounded-sm" style={{ backgroundColor: tokenCssVar(r.token) }} aria-hidden />
          <span className="tabular-nums">{r.label}</span>
        </li>
      ))}
    </ul>
  )
}

/** Le repère d'une courbe du net : un trait et un losange à la couleur du joueur. */
function NetSwatch({ color }: { color: string }) {
  const line: CSSProperties = { backgroundColor: color }
  return (
    <span className="relative inline-flex h-2 w-[18px] shrink-0 items-center justify-center" aria-hidden>
      <span className="absolute inset-x-0 h-0.5" style={line} />
      <span className="relative h-1.5 w-1.5 rotate-45" style={line} />
    </span>
  )
}
